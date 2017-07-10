package gdax

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const orderBookPath = "/products/%s/book?level=%d"

var ProductIDs = []string{
	"BTC-USD", "ETH-USD", "LTC-USD",
	"ETH-BTC", "LTC-BTC",
}

func CurrencyPrecision(currency string) int {
	switch currency {
	case "USD", "EUR", "GBP":
		return 2
	case "BTC", "LTC", "ETH":
		return 8
	default:
		return -1
	}
}

func ProductIDForCurrencyPair(a, b string) (productID string) {
	switch a {
	case "USD", "EUR":
		if b == "BTC" || b == "ETH" || b == "LTC" {
			productID = fmt.Sprintf("%s-%s", b, a)
		}
	case "GBP":
		if b == "BTC" {
			productID = "BTC-GBP"
		}
	case "BTC":
		if b == "USD" || b == "GBP" || b == "EUR" {
			productID = fmt.Sprintf("%s-%s", a, b)
		} else if b == "LTC" || b == "ETH" {
			productID = fmt.Sprintf("%s-%s", b, a)
		}
	case "ETH", "LTC":
		if b == "USD" || b == "EUR" || b == "BTC" {
			productID = fmt.Sprintf("%s-%s", a, b)
		}
	}
	return
}

func IsValidProductID(productID string) (valid bool) {
	for _, id := range ProductIDs {
		if productID == id {
			valid = true
		}
	}
	return
}

type API struct {
	URL string
}

func NewAPI(url string) (*API, error) {
	if len(url) < 1 {
		return nil, fmt.Errorf("Missing GDAX REST API URL\n")
	}

	return &API{url}, nil
}

func (a API) Request(
	c *http.Client, ctx context.Context, method, path, message string,
) ([]byte, error) {
	request, err := http.NewRequest(method, a.URL+path, strings.NewReader(message))
	if err != nil {
		return nil, err
	}

	request = request.WithContext(ctx)
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json; charset=utf-8")

	response, err := c.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return body, err
	}

	return body, nil
}

func (a API) GetOrderBook(
	c *http.Client, ctx context.Context, productID string, level int,
) (*OrderBook, error) {

	if !IsValidProductID(productID) {
		return nil, fmt.Errorf("%s is not a valid product ID", productID)
	}

	if level < 1 || level > 3 {
		return nil, fmt.Errorf("Level must be 1, 2, or 3")
	}

	path := fmt.Sprintf(orderBookPath, productID, level)

	body, err := a.Request(c, ctx, http.MethodGet, path, "")
	if err != nil {
		return nil, err
	}

	ob := OrderBook{}
	if err := json.Unmarshal(body, &ob); err != nil {
		return nil, err
	}

	return &ob, nil
}
