package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/akb/gdax-quote/gdax"
)

// singletons
var listenPort string
var api *gdax.API
var client *http.Client

func init() {
	listenPort = os.Getenv("GDAX_QUOTE_LISTEN_PORT")
	if len(listenPort) == 0 {
		listenPort = "3000"
	}

	url := os.Getenv("GDAX_API_URL")
	if len(url) == 0 {
		url = "https://api.gdax.com"
	}

	client = &http.Client{
		Timeout:   time.Second * 10,
		Transport: newClientLogger(),
	}

	var err error
	api, err = gdax.NewAPI(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func main() {
	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%v", listenPort),
		Handler: &serverLogger{mux},
	}

	mux.HandleFunc("/quote", handleQuote)

	log.Printf("Listening on port %s\n", listenPort)
	log.Fatal(server.ListenAndServe())
}
