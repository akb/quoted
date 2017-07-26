package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/satori/go.uuid"

	"github.com/akb/quoted/gdax"
)

// QuoteRequest contains parameters needed for producing a quote. A JSON object
// that can be unmarshaled into this struct is expected to be received in the
// request body for a POST request to /quote
type QuoteRequest struct {
	Action        string `json:"action"`
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	Amount        string `json:"amount"`
}

// QuoteResponse contains fields representing a price quote for a quantity of a
// product. It is marshaled into a JSON object that is returned from POST
// requests to /quote
type QuoteResponse struct {
	Price    string `json:"price"`
	Total    string `json:"total"`
	Currency string `json:"currency"`
}

// POST /quote
func handleQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(r.Body)
	var q QuoteRequest
	err := decoder.Decode(&q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if q.Action != "buy" && q.Action != "sell" {
		writeError(w, http.StatusBadRequest, "action must be 'buy' or 'sell'")
		return
	}

	floatAmount, err := strconv.ParseFloat(q.Amount, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if floatAmount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be a positive number")
		return
	}

	productID := gdax.ProductIDForCurrencyPair(q.BaseCurrency, q.QuoteCurrency)
	if len(productID) < 1 {
		writeError(w, http.StatusBadRequest, "invalid currency pair")
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, traceIDKey, uuid.NewV4().String())

	action := q.Action
	inverse := false
	if productID[0:3] != q.BaseCurrency {
		inverse = true
		if q.Action == "buy" {
			action = "sell"
		} else {
			action = "buy"
		}
	}

	price, quantity, err := orderbooks[productID].Quote(
		action, q.QuoteCurrency, floatAmount, inverse)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	stringPrice := strconv.FormatFloat(price, 'f',
		gdax.CurrencyPrecision(q.QuoteCurrency), 64)
	stringQuantity := strconv.FormatFloat(quantity, 'f',
		gdax.CurrencyPrecision(q.QuoteCurrency), 64)

	body, err := json.Marshal(
		QuoteResponse{stringPrice, stringQuantity, q.QuoteCurrency})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func writeError(w http.ResponseWriter, status int, message interface{}) {
	w.WriteHeader(status)
	// fmt.Sprintf is used instead of json.Marshal because marshaling can produce
	// an error and we want to guarantee a JSON-formatted response
	w.Write([]byte(fmt.Sprintf(`{"message":"%s"}`, message)))
}
