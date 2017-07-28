package gdax

import (
	"math/rand"
	"strconv"
	"testing"
)

func makeOrderBook() *OrderBook {
	orderA := OrderBookEntry{49.97, 11.5, 0, "order-a", BidSide}
	orderB := OrderBookEntry{49.96, 9.5, 0, "order-b", BidSide}
	orderC := OrderBookEntry{49.92, 7.5, 0, "order-c", BidSide}
	orderD := OrderBookEntry{49.89, 5.5, 0, "order-d", BidSide}

	orderE := OrderBookEntry{50.01, 4.5, 0, "order-e", AskSide}
	orderF := OrderBookEntry{50.06, 6.5, 0, "order-f", AskSide}
	orderG := OrderBookEntry{50.13, 8.5, 0, "order-g", AskSide}
	orderH := OrderBookEntry{50.26, 10.5, 0, "order-h", AskSide}

	return &OrderBook{
		Sequence: 0,
		entries: map[string]*OrderBookEntry{
			"order-a": &orderA, "order-b": &orderB,
			"order-c": &orderC, "order-d": &orderD,
			"order-e": &orderE, "order-f": &orderF,
			"order-g": &orderG, "order-h": &orderH,
		},
		Bids: []*OrderBookEntry{&orderA, &orderB, &orderC, &orderD},
		Asks: []*OrderBookEntry{&orderE, &orderF, &orderG, &orderH},
	}
}

func TestFind(t *testing.T) {
	ob := makeOrderBook()
	e := ob.Find("order-f")
	if e == nil {
		t.Fatalf("could not find order")
	}
	if e.Price != 50.06 {
		t.Errorf("incorrect price, wanted %v, received %v", 50.06, e.Price)
	}
	if e.Size != 6.5 {
		t.Errorf("incorrect size, wanted %v, received %v", 6.5, e.Size)
	}
}

func TestInsert(t *testing.T) {
	ob := makeOrderBook()

	for _, e := range []struct {
		side  string
		price float64
		size  float64
		id    string
	}{
		{BidSide, 49.98, 10.2, "order-4"},
		{AskSide, 50.03, 13.3, "order-2"},
		{AskSide, 50.00, 5.3, "order-3"},
		{BidSide, 49.95, 13.3, "order-5"},
		{AskSide, 50.05, 10.2, "order-1"},
		{BidSide, 49.90, 5.3, "order-6"},
	} {
		if err := ob.Insert(e.side, e.price, e.size, e.id); err != nil {
			t.Errorf("%s", err)
		}
	}

	if len(ob.Bids) != 7 {
		t.Errorf("Failed to insert 3 bids")
	}

	if len(ob.Asks) != 7 {
		t.Errorf("Failed to insert 3 asks")
	}

	last := ob.Bids[0]
	for _, e := range ob.Bids[1:] {
		if e.Price >= last.Price {
			t.Errorf("Bids aren't sorted")
		}
		last = e
	}

	last = ob.Asks[0]
	for _, e := range ob.Asks[1:] {
		if e.Price <= last.Price {
			t.Errorf("Asks aren't sorted")
		}
		last = e
	}
}

func BenchmarkInsert(b *testing.B) {
	ob := makeOrderBook()
	for i := 0; i < b.N; i++ {
		ob.Insert(BidSide, rand.Float64(), rand.Float64(), strconv.Itoa(rand.Int()))
	}
}

func TestDelete(t *testing.T) {
	ob := makeOrderBook()

	ob.Delete("order-b")

	if len(ob.Bids) >= 4 {
		t.Errorf("Failed to delete an order")
	}

	if len(ob.Asks) < 4 {
		t.Errorf("Deleted an Ask instead of a Bid")
	}

	for _, b := range ob.Bids {
		if b.OrderID == "order-b" {
			t.Errorf("Failed to delete order-b")
		}
	}
}

func BenchmarkInsertDelete(b *testing.B) {
	ob := makeOrderBook()
	for i := 0; i < b.N; i++ {
		id := strconv.Itoa(rand.Int())
		ob.Insert(BidSide, rand.Float64(), rand.Float64(), id)
		ob.Delete(id)
	}
}

func TestMatch(t *testing.T) {
	ob := makeOrderBook()

	ob.Match("order-b", 1.0)
	e := ob.Find("order-b")
	if e.Size != 8.5 {
		t.Errorf("entry size should be 8.5, got %v", e.Size)
	}
}

func TestChange(t *testing.T) {
	ob := makeOrderBook()

	ob.Change("order-b", 1.0)
	e := ob.Find("order-b")
	if e.Size != 1.0 {
		t.Errorf("entry size should be 1.0, got %v", e.Size)
	}
}

func TestQuote(t *testing.T) {
	ob := makeOrderBook()

	price, total, _ := ob.Quote(BuyAction, "LTC", 2.0, false)
	if price != 50.01 {
		t.Errorf("Quote returned the wrong price")
	}
	if total != 100.02 {
		t.Errorf("Quote returned the wrong total")
	}
}
