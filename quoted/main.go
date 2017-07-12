package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/satori/go.uuid"

	"github.com/akb/gdax-quote/gdax"
)

var (
	api        *gdax.API
	client     *http.Client
	orderbooks map[string]*gdax.LiveOrderBook
	done       chan struct{}
)

const (
	origin = "http://localhost"
)

var (
	listenPort   string
	url          string
	websocketURL string
)

// these are all the product ids, but GDAX seems to limit you to local currency
//var productIDs = []string{
//	"BTC-USD", "BTC-GBP", "BTC-EUR",
//	"ETH-USD", "ETH-EUR", "ETH-BTC",
//	"LTC-USD", "LTC-EUR", "LTC-BTC",
//}

var productIDs = []string{
	"BTC-USD",
	"ETH-USD", "ETH-BTC",
	"LTC-USD", "LTC-BTC",
}

func init() {
	orderbooks = map[string](*gdax.LiveOrderBook){}
	done = make(chan struct{})

	listenPort = os.Getenv("GDAX_QUOTE_LISTEN_PORT")
	if len(listenPort) == 0 {
		listenPort = "3000"
	}

	url = os.Getenv("GDAX_API_URL")
	if len(url) == 0 {
		url = "https://api.gdax.com"
	}

	websocketURL = os.Getenv("GDAX_WEBSOCKET_URL")
	if len(websocketURL) == 0 {
		websocketURL = "wss://ws-feed.gdax.com"
	}
}

func main() {
	client = &http.Client{
		Timeout:   time.Second * 10,
		Transport: newClientLogger(),
	}

	var err error
	api, err = gdax.NewAPI(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error connecting to REST API")
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	feed, err := gdax.NewFeed(websocketURL, origin, productIDs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error establishing websocket connection")
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, traceIDKey, uuid.NewV4().String())

	for _, p := range productIDs {
		lob, err := api.NewLiveOrderBook(client, ctx, feed, p, done)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error while establishing order books")
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		orderbooks[p] = lob
		go func() {
			for err := range lob.ErrorChan {
				fmt.Fprintln(os.Stderr, err)
			}
		}()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/quote", handleQuote)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%v", listenPort),
		Handler: &serverLogger{mux},
	}

	log.Printf("Listening on port %s\n", listenPort)
	log.Fatal(server.ListenAndServe())
}
