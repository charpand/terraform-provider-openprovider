// Package prices asks Openprovider what an operation on a domain costs, so a
// resource can refuse to spend more than it was told to.
package prices

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// Price is one amount in one currency.
type Price struct {
	Currency string  `json:"currency"`
	Price    float64 `json:"price"`
}

// Group holds the two amounts Openprovider quotes for the same operation:
// `Product` is the retail price, `Reseller` the price charged to the account
// making the call.
type Group struct {
	Product  Price `json:"product"`
	Reseller Price `json:"reseller"`
}

// Quote is what the price endpoint answers with.
type Quote struct {
	IsPremium bool  `json:"is_premium"`
	Price     Group `json:"price"`
}

type quoteResponse struct {
	Code int    `json:"code"`
	Desc string `json:"desc"`
	Data Quote  `json:"data"`
}

// Charge reports the amount that will actually be debited, and its currency.
// That is the reseller price where Openprovider quotes one; an account without
// a membership is quoted at the retail price instead, and pays that.
//
// A zero reseller amount is read as "not quoted" rather than "free": the
// endpoint leaves the member half empty for a non-member, and treating that as
// free would defeat the bound this exists to enforce.
func (q Quote) Charge() Price {
	if q.Price.Reseller.Price > 0 {
		return q.Price.Reseller
	}
	return q.Price.Product
}

// Create asks what registering `name`.`extension` for `period` years costs.
//
// Endpoint: GET /v1beta/domains/prices
//
// The domain parameters are dotted, not underscored: an underscored spelling
// is read as no domain at all and the endpoint answers `300`, "Your domain
// request contains an empty domain name".
func Create(c *client.Client, name, extension string, period int64) (*Quote, error) {
	query := url.Values{}
	query.Set("domain.name", name)
	query.Set("domain.extension", extension)
	query.Set("operation", "create")
	query.Set("period", fmt.Sprintf("%d", period))

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1beta/domains/prices?%s", c.BaseURL, query.Encode()), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var envelope quoteResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("could not read the quote (status %d): %w", resp.StatusCode, err)
	}

	if envelope.Code != 0 {
		return nil, fmt.Errorf("openprovider error %d: %s", envelope.Code, envelope.Desc)
	}

	// `resp.StatusCode` is 2xx whenever `err` is nil above (`client.Client.Do`
	// turns anything else into an error before returning), so no status check
	// is needed here.

	return &envelope.Data, nil
}
