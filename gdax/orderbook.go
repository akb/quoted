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
)

// OrderBook contains a snapshot of limit orders on GDAX
type OrderBook struct {
	Sequence int
	Bids     []OrderBookEntry
	Asks     []OrderBookEntry
}

// OrderBookEntry contains values found in 3-tuples in the API response. Go
// doesn't handle polymorphism or non-homogenous slices so instead we use this
// struct and some complex unmarshaling code
type OrderBookEntry struct {
	Price     float64
	Size      float64
	NumOrders float64
	OrderID   string
}

// Quote tallies order book entries until the requested amount is met, then
// subtracts any overage then returns it and the price
func (ob *OrderBook) Quote(
	action, currency string, amount float64, inverse bool,
) (price, total float64, err error) {
	var side []OrderBookEntry
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
		Sequence int             `json:"sequence"`
		Bids     [][]interface{} `json:"bids"`
		Asks     [][]interface{} `json:"asks"`
	}{}

	if err := json.Unmarshal(buf, &sob); err != nil {
		return err
	}

	ob.Sequence = sob.Sequence

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
) (entry OrderBookEntry, err error) {
	price, ok := serverEntry[0].(string)
	if !ok {
		return entry, fmt.Errorf("API returned non-string for order price")
	}

	floatPrice, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return entry, fmt.Errorf("Unable to parse price as floating point number: %s", price)
	}

	entry.Price = floatPrice

	size, ok := serverEntry[1].(string)
	if !ok {
		return entry, fmt.Errorf("API returned non-string for order size")
	}

	floatSize, err := strconv.ParseFloat(size, 64)
	if err != nil {
		return entry, fmt.Errorf("Unable to parse size as floating point number: %s", size)
	}

	entry.Size = floatSize

	switch v := serverEntry[2].(type) {
	case float64:
		entry.NumOrders = v
	case string:
		entry.OrderID = v
	default:
		return entry, fmt.Errorf("Unable to parse order book entry")
	}

	return
}
