package api

import (
	"context"
	"net/url"
)

type Exchange struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	MicCode string `json:"mic_code"`
}

type ExchangeIndex struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

type ExchangeDetail struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	MicCode          string          `json:"mic_code"`
	Acronym          string          `json:"acronym"`
	Country          *Country        `json:"country"`
	OperatingMicCode *string         `json:"operating_mic_code"`
	Currency         string          `json:"currency"`
	TickerSuffix     string          `json:"ticker_suffix"`
	Timezone         string          `json:"timezone"`
	City             string          `json:"city"`
	Indices          []ExchangeIndex `json:"indices,omitempty"`
}

func (c *Client) ListExchanges(ctx context.Context, params url.Values) ([]ExchangeDetail, *Pagination, *Response, error) {
	return ListPage[ExchangeDetail](ctx, c, "/exchanges", params, "exchanges")
}

func (c *Client) GetExchange(ctx context.Context, idOrMic string) (*ExchangeDetail, *Response, error) {
	var wrapper struct {
		Exchange ExchangeDetail `json:"exchange"`
	}
	resp, err := c.Get(ctx, "/exchanges/"+url.PathEscape(idOrMic), nil, &wrapper)
	if err != nil {
		return nil, resp, err
	}
	return &wrapper.Exchange, resp, nil
}

func (c *Client) ListExchangeCompanies(ctx context.Context, idOrMic string, params url.Values) ([]Company, *Pagination, *Response, error) {
	return ListPage[Company](ctx, c, "/exchanges/"+url.PathEscape(idOrMic)+"/companies", params, "companies")
}

func (c *Client) ListAllExchangeCompanies(ctx context.Context, idOrMic string, params url.Values) ([]Company, *Response, error) {
	return ListAll[Company](ctx, c, "/exchanges/"+url.PathEscape(idOrMic)+"/companies", params, "companies")
}
