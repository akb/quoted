quoted
======

HTTP service for providing digital currency price quotes.

## Usage

If you're on a Mac, you can probably run the prebuilt binary, `bin/quoted`.

## Build Instructions

You will need [Go](https://golang.org) installed and your GOPATH environment
variable set. The directory containing this file needs to resolve to
`$GOPATH/src/github.com/akb/gdax-quote`.

To build and run `bin/quoted`:

    make

## Running Tests

Start the server then in another terminal run:

    make test

Tests are written in Ruby. I didn't want to depend on any gems, so rather than
use rspec, the test is a standalone script.

## Environment Variables

`GDAX_QUOTE_LISTEN_PORT`   The port on which `quoted` listens. Default: 3000
`GDAX_API_URL`             URL for the GDAX REST API. Default: Public API
`GDAX_WEBSOCKET_URL`       URL for the GDAX websocket API. Default: Public API

## Directory Layout

```
README.md                 This file
Makefile                  Build scripts
bin/quoted                Binary executable
test.rb                   Ruby script containing integration tests
quoted/                   API server source code
quoted/main.go            Main entry point for API server
quoted/logger.go          Tools for HTTP logging
quoted/quote.go           "/quote" API endpoint
gdax/                     GDAX API client
gdax/api.go               Client for the GDAX REST API
gdax/orderbook.go         Orderbook model
gdax/live-orderbook.go    Maintains an orderbook in realtime using the GDAX
                          REST API and websocket feed. Thread safe.
gdax/websocket.go         Client for the GDAX websocket feed
vendor/                   3rd-party libraries, managed with gvt
```
