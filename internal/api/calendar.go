package api

import (
	"context"
	"net/url"
)

// CalendarEvent is a financial calendar event for a company (earnings report,
// earnings call, dividend, AGM, M&A milestone, etc.).
type CalendarEvent struct {
	ID                   string    `json:"id"`
	Title                string    `json:"title"`
	Description          string    `json:"description,omitempty"`
	EventType            string    `json:"event_type"`
	Status               string    `json:"status"`
	DatePrecision        string    `json:"date_precision"`
	FiscalPeriod         string    `json:"fiscal_period,omitempty"`
	ScheduledAt          string    `json:"scheduled_at"`
	Timezone             string    `json:"timezone"`
	LocalTime            string    `json:"local_time"`
	ScheduledAtChangedAt string    `json:"scheduled_at_changed_at,omitempty"`
	SourceArticleID      string    `json:"source_article_id,omitempty"`
	CompanyID            string    `json:"company_id"`
	Country              string    `json:"country"`
	Dividend             *Dividend `json:"dividend"`
	UpdatedAt            string    `json:"updated_at,omitempty"`
}

// Dividend holds the dividend-specific dates for a dividend calendar event.
// It is null for non-dividend events. Amount and Currency are currently
// withheld by the API but may appear in future responses.
type Dividend struct {
	ExDate          string `json:"ex_date,omitempty"`
	RecordDate      string `json:"record_date,omitempty"`
	PaymentDate     string `json:"payment_date,omitempty"`
	DeclarationDate string `json:"declaration_date,omitempty"`
	Kind            string `json:"kind,omitempty"`
	Amount          string `json:"amount,omitempty"`
	Currency        string `json:"currency,omitempty"`
}

func (c *Client) ListCalendarEvents(ctx context.Context, params url.Values) ([]CalendarEvent, *Pagination, *Response, error) {
	return ListPage[CalendarEvent](ctx, c, "/calendar_events", params, "calendar_events")
}

func (c *Client) ListAllCalendarEvents(ctx context.Context, params url.Values) ([]CalendarEvent, *Response, error) {
	return ListAll[CalendarEvent](ctx, c, "/calendar_events", params, "calendar_events")
}

func (c *Client) GetCalendarEvent(ctx context.Context, id string) (*CalendarEvent, *Response, error) {
	var wrapper struct {
		CalendarEvent CalendarEvent `json:"calendar_event"`
	}
	resp, err := c.Get(ctx, "/calendar_events/"+url.PathEscape(id), nil, &wrapper)
	if err != nil {
		return nil, resp, err
	}
	return &wrapper.CalendarEvent, resp, nil
}
