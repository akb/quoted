package gdax

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
)

type liveOrderBookState string

const (
	newState           liveOrderBookState = "new"
	loadingState       liveOrderBookState = "loading"
	synchronizingState liveOrderBookState = "synchronizing"
	runningState       liveOrderBookState = "running"
)

type liveOrderBookAction string

const (
	resetAction       liveOrderBookAction = "reset"
	synchronizeAction liveOrderBookAction = "synchronize"
	runAction         liveOrderBookAction = "run"
)

type LiveOrderBook struct {
	*sync.RWMutex
	*OrderBook

	api     API
	client  *http.Client
	context context.Context

	productID       string
	state           liveOrderBookState
	droppedMessages int64

	// second mutex is used to avoid deadlocks
	queueLock *sync.RWMutex
	queue     []Message

	actionChan chan liveOrderBookAction
	ErrorChan  chan error
}

func (a API) NewLiveOrderBook(
	c *http.Client, ctx context.Context, feed *Feed,
	productID string, done <-chan struct{},
) (*LiveOrderBook, error) {
	if !IsValidProductID(productID) {
		return nil, fmt.Errorf("%s is not a valid product id", productID)
	}

	lob := LiveOrderBook{
		RWMutex:   &sync.RWMutex{},
		OrderBook: nil,

		api:     a,
		client:  c,
		context: ctx,

		productID:       productID,
		state:           newState,
		droppedMessages: 0,

		queueLock: &sync.RWMutex{},
		queue:     []Message{},

		actionChan: make(chan liveOrderBookAction, 1),
		ErrorChan:  make(chan error),
	}

	go lob.listen(feed)
	go lob.loop(done)

	defer lob.Reset()

	return &lob, nil
}

// Reset clears the order book, fetches a new state and re-synchronizes it
func (lob *LiveOrderBook) Reset() {
	lob.actionChan <- resetAction
}

// Quote is a thread-safe method proxy for OrderBook::Quote
func (lob *LiveOrderBook) Quote(
	action, currency string, amount float64, inverse bool,
) (price, total float64, err error) {
	lob.RLock()
	defer lob.RUnlock()
	return lob.OrderBook.Quote(action, currency, amount, inverse)
}

// DroppedMessageCount return the total number of messages dropped since the
// last reset
func (l *LiveOrderBook) DroppedMessageCount() int64 {
	l.RLock()
	defer l.RUnlock()
	return l.droppedMessages
}

// the main event loop, runs in a goroutine
func (lob *LiveOrderBook) loop(done <-chan struct{}) {
loop:
	for {
		select {
		case a := <-lob.actionChan:
			if err := lob.do(a); err != nil {
				lob.ErrorChan <- err
			}
		case <-done:
			break loop
		}
	}
}

// performs actions and manages state transitions
func (lob *LiveOrderBook) do(a liveOrderBookAction) error {
	switch lob.state {
	case newState:
		switch a {
		case resetAction:
			if err := lob.doReset(); err != nil {
				return err
			}

			lob.Lock()
			lob.state = loadingState
			lob.Unlock()

			lob.actionChan <- synchronizeAction
		}

	case loadingState:
		switch a {
		case synchronizeAction:
			lob.Lock()
			lob.state = synchronizingState
			lob.Unlock()

			if err := lob.doSynchronize(); err != nil {
				return err
			}

			lob.actionChan <- runAction
		}

	case synchronizingState:
		switch a {
		case runAction:
			lob.Lock()
			lob.state = runningState
			lob.Unlock()
		}

	case runningState:
		switch a {
		case resetAction:
			lob.Lock()
			lob.state = newState
			lob.OrderBook = nil
			lob.Unlock()

			if err := lob.doReset(); err != nil {
				return err
			}

			lob.Lock()
			lob.state = loadingState
			lob.Unlock()

			lob.actionChan <- synchronizeAction
		}
	}

	return nil
}

func (lob *LiveOrderBook) doReset() error {
	lob.Lock()
	lob.state = newState
	lob.droppedMessages = 0
	lob.queue = []Message{}
	lob.Unlock()

	orderbook, err := lob.api.GetOrderBook(lob.client, lob.context, lob.productID, 3)
	if err != nil {
		return err
	}

	lob.Lock()
	lob.OrderBook = orderbook
	lob.Unlock()

	return nil
}

func (lob *LiveOrderBook) doSynchronize() error {
	for {
		lob.queueLock.Lock()
		if len(lob.queue) == 0 {
			lob.queueLock.Unlock()
			break
		}
		m := lob.queue[0]
		lob.queue = lob.queue[1:]
		lob.queueLock.Unlock() // this is repeated instead of using defer because we want to release the lock before calling `handle`
		if err := lob.handle(m); err != nil {
			return err
		}
	}
	return nil
}

// listens for events from GDAX feed and dispatches
func (lob *LiveOrderBook) listen(feed *Feed) {
	messageChan := make(chan Message)
	feed.Subscribe(messageChan)

	for m := range messageChan {
		lob.RLock()
		state := lob.state
		lob.RUnlock()

		switch state {
		case newState, loadingState, synchronizingState:
			lob.enqueue(m)
		case runningState:
			if err := lob.handle(m); err != nil {
				lob.ErrorChan <- err
			}
		}
	}

}

// enqueue appends an event to the queue. this queue is used during
// initialization to capture any events that occur while the initial order book
// state is loading via HTTP
func (lob *LiveOrderBook) enqueue(m Message) {
	lob.queueLock.Lock()
	lob.queue = append(lob.queue, m)
	lob.queueLock.Unlock()
}

// only this function should access the sequence number to avoid locking
func (lob *LiveOrderBook) handle(m Message) error {
	// throw out irrelevant messages
	if m.ProductID != lob.productID {
		return nil
	}

	// throw out stale messages
	if m.Sequence <= lob.Sequence {
		return nil
	}

	// cache previous value, and advance internal sequence number
	sequence := lob.Sequence
	lob.Sequence = m.Sequence

	// detect if we missed any messages and track how many
	droppedMessages := m.Sequence - sequence - 1
	if droppedMessages > 0 {
		fmt.Fprintf(os.Stderr, "Dropped %v messages.\n", droppedMessages)
		lob.Lock()
		lob.droppedMessages += droppedMessages // TODO: reset if dropped messages?
		lob.Unlock()
	}

	switch m.Type {
	case "open":
		return lob.handleOpen(m)
	case "done":
		return lob.handleDone(m)
	case "match":
		return lob.handleMatch(m)
	case "change":
		return lob.handleChange(m)
	}

	return nil
}

func (lob *LiveOrderBook) handleOpen(m Message) error {
	price, err := strconv.ParseFloat(m.Price, 64)
	if err != nil {
		return fmt.Errorf("error parsing float from entry price (%s)\n", m.Price)
	}

	size, err := strconv.ParseFloat(m.RemainingSize, 64)
	if err != nil {
		return fmt.Errorf("error parsing float from entry remaining_size (%s)\n", m.RemainingSize)
	}

	lob.Lock()
	defer lob.Unlock()
	if err := lob.Insert(m.Side, price, size, m.OrderID); err != nil {
		return err
	}
	return nil
}

func (lob *LiveOrderBook) handleMatch(m Message) error {
	size, err := strconv.ParseFloat(m.Size, 64)
	if err != nil {
		return fmt.Errorf("error parsing float from entry size (%s)\n", m.Size)
	}

	lob.Lock()
	defer lob.Unlock()
	if err := lob.Match(m.MakerOrderID, m.Side, size); err != nil {
		return err
	}
	return nil
}

func (lob *LiveOrderBook) handleDone(m Message) error {
	lob.Lock()
	defer lob.Unlock()
	if err := lob.Delete(m.Side, m.OrderID); err != nil {
		return err
	}
	return nil
}

func (lob *LiveOrderBook) handleChange(m Message) error {
	size, err := strconv.ParseFloat(m.NewSize, 64)
	if err != nil {
		return fmt.Errorf("error parsing float from entry size (%s)\n", m.Size)
	}

	lob.Lock()
	defer lob.Unlock()
	if err := lob.Change(m.OrderID, m.Side, size); err != nil {
		return err
	}
	return nil
}
