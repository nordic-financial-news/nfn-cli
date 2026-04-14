package api

import (
	"context"
	"net/url"
)

type Company struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Slug          string          `json:"slug"`
	TickerSymbol  string          `json:"ticker"`
	IsActive      bool            `json:"is_active"`
	Exchange      *Exchange       `json:"exchange"`
	Country       *CompanyCountry `json:"country,omitempty"`
	Description   string          `json:"description,omitempty"`
	Sector        string          `json:"sector,omitempty"`
	Website       string          `json:"website,omitempty"`
	ParentCompany *Company        `json:"parent_company,omitempty"`
	Subsidiaries  []Company       `json:"subsidiaries,omitempty"`
	Indices       []IndexSummary  `json:"indices,omitempty"`
	ArticleCount  int             `json:"article_count,omitempty"`
	StoryCount    int             `json:"story_count,omitempty"`
	StockListings []StockListing  `json:"stock_listings,omitempty"`
	Registry      *Registry       `json:"registry,omitempty"`
	UpdatedAt     string          `json:"updated_at,omitempty"`
}

type CompanyCountry struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ISO2Code string `json:"iso2_code"`
}

type StockListing struct {
	TickerSymbol string   `json:"ticker"`
	Primary      bool     `json:"primary"`
	Exchange     Exchange `json:"exchange"`
}

type Registry struct {
	Source               string        `json:"source,omitempty"`
	RegisteredName       string        `json:"registered_name,omitempty"`
	CompanyForm          string        `json:"company_form,omitempty"`
	City                 string        `json:"city,omitempty"`
	Founded              string        `json:"founded,omitempty"`
	Employees            *int          `json:"employees,omitempty"`
	Industry             string        `json:"industry,omitempty"`
	Bankruptcy           *bool         `json:"bankruptcy,omitempty"`
	UnderLiquidation     *bool         `json:"under_liquidation,omitempty"`
	ShareCapital         *ShareCapital `json:"share_capital,omitempty"`
	InstitutionalSector  string        `json:"institutional_sector,omitempty"`
	DeregisteredAt       string        `json:"deregistered_at,omitempty"`
	DeregistrationReason string        `json:"deregistration_reason,omitempty"`
}

type ShareCapital struct {
	Amount   *float64 `json:"amount"`
	Currency *string  `json:"currency"`
}

func (c *Client) ListCompanies(ctx context.Context, params url.Values) ([]Company, *Pagination, *Response, error) {
	return ListPage[Company](ctx, c, "/companies", params, "companies")
}

func (c *Client) ListAllCompanies(ctx context.Context, params url.Values) ([]Company, *Response, error) {
	return ListAll[Company](ctx, c, "/companies", params, "companies")
}

func (c *Client) GetCompany(ctx context.Context, idOrTicker string) (*Company, *Response, error) {
	var wrapper struct {
		Company Company `json:"company"`
	}
	resp, err := c.Get(ctx, "/companies/"+url.PathEscape(idOrTicker), nil, &wrapper)
	if err != nil {
		return nil, resp, err
	}
	return &wrapper.Company, resp, nil
}

func (c *Client) ListCompanyArticles(ctx context.Context, idOrTicker string, params url.Values) ([]Article, *Pagination, *Response, error) {
	return ListPage[Article](ctx, c, "/companies/"+url.PathEscape(idOrTicker)+"/articles", params, "articles")
}

func (c *Client) ListAllCompanyArticles(ctx context.Context, idOrTicker string, params url.Values) ([]Article, *Response, error) {
	return ListAll[Article](ctx, c, "/companies/"+url.PathEscape(idOrTicker)+"/articles", params, "articles")
}

func (c *Client) ListCompanyStories(ctx context.Context, idOrTicker string, params url.Values) ([]Story, *Pagination, *Response, error) {
	return ListPage[Story](ctx, c, "/companies/"+url.PathEscape(idOrTicker)+"/stories", params, "stories")
}

func (c *Client) ListAllCompanyStories(ctx context.Context, idOrTicker string, params url.Values) ([]Story, *Response, error) {
	return ListAll[Story](ctx, c, "/companies/"+url.PathEscape(idOrTicker)+"/stories", params, "stories")
}
