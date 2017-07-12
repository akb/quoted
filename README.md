quoted
======

HTTP service for providing digital currency price quotes.

## Usage

If you're on a Mac, you can probably run the prebuilt binary, `bin/quoted`.

## Build Instructions

You will need [Go](https://golang.org) installed and your GOPATH environment
variable set. The directory containing this file needs to resolve to
`$GOPATH/src/github.com/akb/gdax-quote`.

To rebuild `bin/quoted` run:

    make

## Running Tests

Start the server `bin/quoted` server then in another terminal run:

    make test

Tests are written in Ruby. I didn't want to depend on any gems, so rather than
use rspec, the test is a standalone script.

## Environment Variables

The port which `quoted` listens on can be overridden by setting the
`GDAX_QUOTE_LISTEN_PORT` environment variable. It defaults to 3000.

The URL for the GDAX API can be set with the `GDAX_API_URL` environment
variable. It defaults to the public GDAX API.

## Directory Layout

```
README.md           This file
Makefile            Build scripts
bin/quoted          Binary executable
test.rb             Ruby script containing integration tests
quoted/             API server source code
quoted/main.go      Main entry point for API server
quoted/logger.go    Tools for HTTP logging
quoted/quote.go     "/quote" API endpoint
gdax/               GDAX API client
gdax/api.go         HTTP client
gdax/orderbook.go   Orderbook model
vendor/             3rd-party libraries, managed with gvt
```

## Known Issues

- Every request to the API issues another backend API request, this could be
  avoided by fetching the order book once, then maintaining its state based on
  events from the websocket feed. This would be a very significant performance
  increase and should be considered mandatory before use in production
- Validation could probably be more thorough
- Timeouts could be better thought-out
- There should be a limit on pending requests at which new requests get
  rejected
- There could be better OS signal handling than the default
- Shutdown is not graceful
- No TLS
