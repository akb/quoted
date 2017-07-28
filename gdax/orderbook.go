package gdax

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

const (
	BuyAction  = "buy"
	SellAction = "sell"

	AskSide = "sell"
	BidSide = "buy"
)

// OrderBook contains a snapshot of limit orders on GDAX
type OrderBook struct {
	Sequence int64 `json:"sequence"`
	entries  map[string]*OrderBookEntry

	Bids []*OrderBookEntry `json:"bids"`
	Asks []*OrderBookEntry `json:"asks"`
}

// OrderBookEntry contains values found in 3-tuples in the API response. Go
// doesn't handle polymorphism or non-homogenous slices so instead we use this
// struct and some complex unmarshaling code
type OrderBookEntry struct {
	Price     float64 `json:"price"`
	Size      float64 `json:"size"`
	NumOrders float64 `json:"num_orders"`
	OrderID   string  `json:"order_id"`

	Side string `json:"-"`
}

func (ob *OrderBook) Find(orderID string) *OrderBookEntry {
	return ob.entries[orderID]
}

// Insert will add a new order into the book, maintaining the price-sorted
// order of the entries
func (ob *OrderBook) Insert(side string, price, size float64, orderID string) error {
	return ob.mutateSide(side,
		func(entries []*OrderBookEntry) []*OrderBookEntry {
			var i int
			var e *OrderBookEntry
			for i, e = range entries {
				if side == BuyAction && e.Price < price {
					break
				} else if side == SellAction && e.Price > price {
					break
				}
			}

			entry := OrderBookEntry{price, size, 1, orderID, side}
			entries = append(entries[:i], append([]*OrderBookEntry{&entry}, entries[i:]...)...)
			ob.entries[orderID] = &entry
			return entries
		})
}

// Delete will remove the order with the specified ID from the order book,
// shrinking the size by one
func (ob *OrderBook) Delete(orderID string) error {
	e := ob.entries[orderID]

	return ob.mutateSide(e.Side,
		func(entries []*OrderBookEntry) []*OrderBookEntry {
			for i, e := range entries {
				if e.OrderID == orderID {
					entries = append(entries[:i], entries[i+1:]...)
					break
				}
			}
			return entries
		})
}

// Match will subtract the matched size from an existing order. If the new size
// reaches 0, the order will not be deleted because there will be a subsequent
// "Delete" call that will do so.
func (ob *OrderBook) Match(orderID string, size float64) error {
	e := ob.entries[orderID]
	return ob.mutateSide(e.Side,
		func(entries []*OrderBookEntry) []*OrderBookEntry {
			for _, e := range entries {
				if e.OrderID == orderID {
					e.Size = e.Size - size
					break
				}
			}
			return entries
		})
}

// Change updates the size of an order
func (ob *OrderBook) Change(orderID string, size float64) error {
	e := ob.entries[orderID]
	return ob.mutateSide(e.Side,
		func(entries []*OrderBookEntry) []*OrderBookEntry {
			for _, e := range entries {
				if e.OrderID == orderID {
					e.Size = size
					break
				}
			}
			return entries
		})
}

func (ob *OrderBook) mutateSide(
	side string, fn func(entries []*OrderBookEntry) []*OrderBookEntry,
) error {
	var entries []*OrderBookEntry
	switch side {
	case BidSide:
		entries = ob.Bids
	case AskSide:
		entries = ob.Asks
	default:
		return fmt.Errorf("Received invalid order book side, %s\n", side)
	}

	entries = fn(entries)

	switch side {
	case BidSide:
		ob.Bids = entries
	case AskSide:
		ob.Asks = entries
	}

	return nil
}

// Quote tallies order book entries until the requested amount is met, then
// subtracts any overage then returns it and the price
func (ob *OrderBook) Quote(
	action, currency string, amount float64, inverse bool,
) (price, total float64, err error) {
	var side []*OrderBookEntry
	if action == BuyAction {
		side = ob.Asks
	} else if action == SellAction {
		side = ob.Bids
	} else {
		return price, total, fmt.Errorf("invalid action %s", action)
	}

	// total order entries until the quote amount can be fulfilled
	var quantity, lastPrice float64
	for _, entry := range side {
		lastPrice = entry.Price
		total += entry.Price * entry.Size
		quantity += entry.Size
		var check float64
		if inverse {
			check = total
		} else {
			check = quantity
		}
		if check >= amount {
			break
		}
	}

	var check float64
	if inverse {
		check = total
	} else {
		check = quantity
	}
	if check < amount {
		return price, total,
			fmt.Errorf("not enough %s available to fill order", currency)
	}

	// subtract overage from total
	overage := quantity - amount
	quantity = quantity - overage
	total = total - (overage * lastPrice)
	price = total / quantity

	// account for prices that are too precise for their currency
	precision := CurrencyPrecision(currency)
	roundPrice := round(price, precision)
	difference := price - roundPrice

	price = price - difference
	total = round(price*quantity, precision)
	return
}

func round(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return math.Floor(f*shift+.5) / shift
}

// UnmarshalJSON implements the json.Unmarshaler interface. The custom
// unmarshaler is required to handle polymorphism in the order book returned by
// the API
func (ob *OrderBook) UnmarshalJSON(buf []byte) error {
	sob := struct {
		Sequence int64           `json:"sequence"`
		Bids     [][]interface{} `json:"bids"`
		Asks     [][]interface{} `json:"asks"`
	}{}

	if err := json.Unmarshal(buf, &sob); err != nil {
		return err
	}

	ob.Sequence = sob.Sequence
	ob.entries = map[string]*OrderBookEntry{}

	for _, b := range sob.Bids {
		entry, err := newOrderBookEntry(b)
		if err != nil {
			return err
		}
		ob.Bids = append(ob.Bids, entry)
	}

	for _, b := range sob.Asks {
		entry, err := newOrderBookEntry(b)
		if err != nil {
			return err
		}
		ob.Asks = append(ob.Asks, entry)
	}

	return nil
}

func newOrderBookEntry(serverEntry []interface{},
) (*OrderBookEntry, error) {
	entry := OrderBookEntry{}
	price, ok := serverEntry[0].(string)
	if !ok {
		return nil, fmt.Errorf("API returned non-string for order price")
	}

	floatPrice, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse price as floating point number: %s", price)
	}

	entry.Price = floatPrice

	size, ok := serverEntry[1].(string)
	if !ok {
		return nil, fmt.Errorf("API returned non-string for order size")
	}

	floatSize, err := strconv.ParseFloat(size, 64)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse size as floating point number: %s", size)
	}

	entry.Size = floatSize

	switch v := serverEntry[2].(type) {
	case float64:
		entry.NumOrders = v
	case string:
		entry.OrderID = v
	default:
		return nil, fmt.Errorf("Unable to parse order book entry")
	}

	return &entry, nil
}
