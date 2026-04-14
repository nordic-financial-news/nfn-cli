package api

import (
	"context"
	"net/url"
)

type CountryExchange struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	MicCode string `json:"mic_code"`
	Acronym string `json:"acronym"`
}

type CountryDetail struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	ISO2Code  string            `json:"iso2_code"`
	Exchanges []CountryExchange `json:"exchanges,omitempty"`
}

func (c *Client) ListCountries(ctx context.Context, params url.Values) ([]CountryDetail, *Pagination, *Response, error) {
	return ListPage[CountryDetail](ctx, c, "/countries", params, "countries")
}

func (c *Client) GetCountry(ctx context.Context, idOrCode string) (*CountryDetail, *Response, error) {
	var wrapper struct {
		Country CountryDetail `json:"country"`
	}
	resp, err := c.Get(ctx, "/countries/"+url.PathEscape(idOrCode), nil, &wrapper)
	if err != nil {
		return nil, resp, err
	}
	return &wrapper.Country, resp, nil
}
