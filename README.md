quoted
======

HTTP service for providing digital currency price quotes.

## Build Instructions

You will need [Go](https://golang.org) installed and your GOPATH environment
variable set. The directory containing this file needs to resolve to
`$GOPATH/src/github.com/akb/quoted`.

To build and run `bin/quoted`:

    make

## Running Tests

Start the server then in another terminal run:

    make test

Tests are written in Ruby. I didn't want to depend on any gems, so rather than
use rspec, the test is a standalone script.

## Environment Variables

| Variable name            | Default    | Description                         |
| ------------------------ | ---------- | ----------------------------------- |
| `GDAX_QUOTE_LISTEN_PORT` | 3000       | The port on which `quoted` listens. |
| `GDAX_API_URL`           | Public API | URL for the GDAX REST API.          |
| `GDAX_WEBSOCKET_URL`     | Public API | URL for the GDAX websocket API.     |

## Directory Layout

```
README.md                 This file
Makefile                  Build scripts
bin/quoted                Binary executable
test.rb                   Ruby script containing integration tests
cmd/                      API server source code
cmd/main.go               Main entry point for API server
cmd/logger.go             Tools for HTTP logging
cmd/quote.go              "/quote" API endpoint
gdax/                     GDAX API client
gdax/api.go               Client for the GDAX REST API
gdax/orderbook.go         Orderbook model
gdax/live-orderbook.go    Maintains an orderbook in realtime using the GDAX
                          REST API and websocket feed. Thread safe.
gdax/websocket.go         Client for the GDAX websocket feed
vendor/                   3rd-party libraries, managed with gvt
```
