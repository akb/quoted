package gdax

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/net/websocket"
)

const (
	MaxSubscribers = 32
)

type Feed struct {
	*sync.Mutex
	*websocket.Conn

	done        chan struct{}
	subscribers []chan Message
	parserStack []rune
}

const (
	ReceivedMessage            = "received"
	OpenMessage                = "open"
	DoneMessage                = "done"
	MatchMessage               = "match"
	ChangeMessage              = "change"
	MarginProfileUpdateMessage = "margin_profile_update"
	HeartbeatMessage           = "heartbeat"
	ErrorMessage               = "error"
)

type Message struct {
	Type               string `json:"type"`
	Time               string `json:"time,omitempty"`
	Sequence           int64  `json:"sequence,omitempty"`
	ProductID          string `json:"product_id,omitempty"`
	OrderID            string `json:"order_id,omitempty"`
	Size               string `json:"size,omitempty"`
	Price              string `json:"price,omitempty"`
	Side               string `json:"side,omitempty"`
	OrderType          string `json:"order_type,omitempty"`
	Funds              string `json:"funds,omitempty"`
	Reason             string `json:"reason,omitempty"`
	RemainingSize      string `json:"remaining_size,omitempty"`
	TradeID            int64  `json:"trade_id,omitempty"`
	MakerOrderID       string `json:"maker_order_id,omitempty"`
	TakerOrderID       string `json:"taker_order_id,omitempty"`
	TakerUserID        string `json:"taker_user_id,omitempty"`
	UserID             string `json:"user_id,omitempty"`
	TakerProfileID     string `json:"taker_profile_id,omitempty"`
	ProfileID          string `json:"profile_id,omitempty"`
	NewSize            string `json:"new_size,omitempty"`
	OldSize            string `json:"old_size,omitempty"`
	NewFunds           string `json:"new_funds,omitempty"`
	OldFunds           string `json:"old_funds,omitempty"`
	Timestamp          string `json:"timestamp,omitempty"`
	Nonce              int64  `json:"nonce,omitempty"`
	Position           string `json:"position,omitempty"`
	PositionSize       string `json:"position_size,omitempty"`
	PositionCompliment string `json:"position_compliment,omitempty"`
	PositionMaxSize    string `json:"position_max_size,omitempty"`
	CallSide           string `json:"call_side,omitempty"`
	CallPrice          string `json:"call_price,omitempty"`
	CallSize           string `json:"call_size,omitempty"`
	CallFunds          string `json:"call_funds,omitempty"`
	Covered            bool   `json:"covered,omitempty"`
	NextExpireTime     string `json:"next_expire_time,omitempty"`
	BaseBalance        string `json:"base_balance,omitempty"`
	BaseFunding        string `json:"base_funding,omitempty"`
	QuoteBalance       string `json:"quote_balance,omitempty"`
	QuoteFunding       string `json:"quote_funding,omitempty"`
	Private            bool   `json:"private,omitempty"`
	LastTradeID        int64  `json:"last_trade_id,omitempty"`
	Message            string `json:"message,omitempty"`
}

func NewFeed(url, origin string, productIDs []string) (*Feed, error) {
	conn, err := websocket.Dial(url, "", origin)
	if err != nil {
		return nil, err
	}

	f := &Feed{
		Mutex:       &sync.Mutex{},
		Conn:        conn,
		done:        make(chan struct{}),
		subscribers: make([]chan Message, MaxSubscribers),
	}

	go f.listen()

	marshaled, err := json.Marshal(struct {
		Type       string   `json:"type"`
		ProductIDs []string `json:"product_ids"`
		Signature  string   `json:"signature,omitempty"`
		Key        string   `json:"key,omitempty"`
		Passphrase string   `json:"passphrase,omitempty"`
		Timestamp  string   `json:"timestamp,omitempty"`
	}{
		Type:       "subscribe",
		ProductIDs: productIDs,
	})
	if err != nil {
		f.done <- struct{}{}
		return nil, err
	}

	_, err = f.Write(marshaled)
	if err != nil {
		f.done <- struct{}{}
		return nil, err
	}

	return f, nil
}

func (f *Feed) Subscribe(c chan Message) {
	f.Lock()
	f.subscribers = append(f.subscribers, c)
	f.Unlock()
}

func (f *Feed) Close() {
	f.Conn.Close()
	f.Conn = nil
	f.done <- struct{}{}
}

func (f *Feed) listen() {
	d := json.NewDecoder(f)
loop:
	for {
		select {
		case <-f.done:
			f.done = nil
			f.Lock()
			for _, subscriber := range f.subscribers {
				close(subscriber)
			}
			f.Unlock()
			break loop

		default:
			var message Message
			if err := d.Decode(&message); err == io.EOF {
				break loop
			} else if err != nil {
				fmt.Fprintln(os.Stderr, "Error reading from WebSocket")
			}

			f.Lock()
			for _, subscriber := range f.subscribers {
				if subscriber == nil {
					continue
				}
				subscriber <- message
			}
			f.Unlock()
		}
	}
}
