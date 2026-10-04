package api

import (
	"context"
	"net/url"
)

// Event is one real-world development — an earnings report, an acquisition, an
// executive change — grouping the articles that report it. It is distinct from
// CalendarEvent, which is a scheduled future date.
//
// A merged or removed event comes back as a tombstone: the API omits every
// field but ID, Status, MergedInto and UpdatedAt, so a tombstone carries no
// content rather than content that is null. Check IsTombstone before reading
// anything else.
type Event struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
	// Status is "active", "closed", "merged" or "removed". Both active and
	// closed are live — closed means the coverage window lapsed, and it can
	// reopen. The other two are terminal and yield a tombstone.
	Status       string     `json:"status"`
	Significance string     `json:"significance,omitempty"`
	EventType    *EventType `json:"event_type,omitempty"`
	// ArticleCount counts every live article reporting the event, not the
	// subset this plan can fetch, so it can exceed len(ArticleIDs). No
	// omitempty: the API can legitimately report 0, and that has to survive
	// into JSON output rather than vanishing. The cost is that a tombstone,
	// where the API omits the field, re-marshals as 0 — Status already marks
	// that case, whereas a live event's real 0 would be indistinguishable.
	ArticleCount int `json:"article_count"`
	// FirstReportedAt is the earliest publication time among the member
	// articles: editorial time, not when the record reached the API.
	FirstReportedAt string `json:"first_reported_at,omitempty"`
	LastArticleAt   string `json:"last_article_at,omitempty"`
	UpdatedAt       string `json:"updated_at"`
	// CompanyIDs is limited to active-involvement roles unless
	// include_peripheral=true widens it.
	CompanyIDs []string `json:"company_ids,omitempty"`
	StoryIDs   []string `json:"story_ids,omitempty"`
	// MergedInto is the surviving event this one was absorbed into. Set only
	// when Status is "merged", and absent on some older merged events.
	MergedInto string `json:"merged_into,omitempty"`
	// RelatedTo is commentary-only: the factual event this piece comments on.
	// Empty for non-commentary events, and for commentary that is not about a
	// particular event.
	RelatedTo string `json:"related_to,omitempty"`

	// The fields below appear on the detail response only.

	// Summary is empty until one has been generated for the event.
	Summary string `json:"summary,omitempty"`
	// ArticleIDs are the live member articles this plan can fetch, oldest
	// first. See ArticleCount for the gap on a restricted plan.
	ArticleIDs []string `json:"article_ids,omitempty"`
	// PrimaryDisclosureArticleID is the earliest member article from a
	// primary-disclosure source (wire service, exchange feed, PR platform or
	// government body) that this plan can fetch. When set it is one of
	// ArticleIDs.
	PrimaryDisclosureArticleID string `json:"primary_disclosure_article_id,omitempty"`
	// MemberFiguresDisagree is true when member articles report divergent
	// values for the same headline figure. It is a pointer so that "figures
	// agree" survives a round trip into JSON output, staying distinct from the
	// tombstone case where the API omits the field.
	MemberFiguresDisagree *bool             `json:"member_figures_disagree,omitempty"`
	FigureDivergence      *FigureDivergence `json:"figure_divergence,omitempty"`
	// Companies is the same set as CompanyIDs with detail per company, primary
	// company first.
	Companies []EventCompany `json:"companies,omitempty"`

	// Distance is the raw cosine distance to the query on a semantic search
	// result: the closest member article this plan can access, so the same
	// event can carry different distances on different plans. See
	// Article.Distance.
	Distance *float64 `json:"distance,omitempty"`
}

// IsTombstone reports whether the event reached a terminal state. The API
// answers a GET on a merged or removed event with 200 and the tombstone shape
// rather than a 404, so the id still resolves.
func (e Event) IsTombstone() bool {
	return e.Status == "merged" || e.Status == "removed"
}

// EventType is the kind of event, inlined on each event row. Slug is the value
// the event_type filter takes — a hyphenated Events-taxonomy slug, a separate
// vocabulary from the underscored calendar event types, which are rejected
// here. A few spellings overlap while denoting different things.
type EventType struct {
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Category string `json:"category"`
}

// EventTypeDetail is one entry in the event taxonomy from /event_types, the
// reference data enumerating every valid event_type filter value.
type EventTypeDetail struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Category string `json:"category"`
	Summary  string `json:"summary,omitempty"`
}

// EventCompany is a company involved in an event, from the detail response.
type EventCompany struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	TickerSymbol string `json:"ticker,omitempty"`
	IsPrimary    bool   `json:"is_primary"`
	// AssociationTier is "principal" (subject, acquirer, target, investor,
	// issuer, partner) or "peripheral" (advisor, speaker, passing mention).
	// Peripheral entries appear only under include_peripheral=true.
	AssociationTier string `json:"association_tier"`
}

// FigureDivergence details the disagreement flagged by
// Event.MemberFiguresDisagree.
type FigureDivergence struct {
	Groups []FigureDivergenceGroup `json:"groups,omitempty"`
}

// FigureDivergenceGroup is one disagreeing metric, with the range across
// sources and the per-article figures behind it.
type FigureDivergenceGroup struct {
	Concept  string   `json:"concept"`
	Basis    string   `json:"basis,omitempty"`
	Role     string   `json:"role,omitempty"`
	Period   string   `json:"period,omitempty"`
	Currency string   `json:"currency,omitempty"`
	Labels   []string `json:"labels,omitempty"`
	// Min and Max bound the reported values, in absolute base units.
	Min    *float64      `json:"min,omitempty"`
	Max    *float64      `json:"max,omitempty"`
	Values []FigureValue `json:"values,omitempty"`
}

// FigureValue is one member article's reported figure for a diverging metric.
type FigureValue struct {
	ArticleID string `json:"id"`
	Source    string `json:"source,omitempty"`
	// NormalizedValue is the figure in absolute base units; Value is the figure
	// as the source originally stated it (e.g. "395 MSEK").
	NormalizedValue *float64 `json:"normalized_value,omitempty"`
	Value           string   `json:"value,omitempty"`
}

func (c *Client) ListEvents(ctx context.Context, params url.Values) ([]Event, *Pagination, *Response, error) {
	return ListPage[Event](ctx, c, "/events", params, "events")
}

func (c *Client) ListAllEvents(ctx context.Context, params url.Values) ([]Event, *Response, error) {
	return ListAllDeduped[Event](ctx, c, "/events", params, "events", eventID)
}

// GetEvent fetches one event by ID. A merged or removed event resolves to a
// tombstone rather than a 404 — see Event.IsTombstone. params carries the
// detail route's only filter, include_peripheral.
func (c *Client) GetEvent(ctx context.Context, id string, params url.Values) (*Event, *Response, error) {
	var wrapper struct {
		Event Event `json:"event"`
	}
	resp, err := c.Get(ctx, "/events/"+url.PathEscape(id), params, &wrapper)
	if err != nil {
		return nil, resp, err
	}
	return &wrapper.Event, resp, nil
}

func (c *Client) ListEventTypes(ctx context.Context, params url.Values) ([]EventTypeDetail, *Pagination, *Response, error) {
	return ListPage[EventTypeDetail](ctx, c, "/event_types", params, "event_types")
}

func (c *Client) ListCompanyEvents(ctx context.Context, idOrTicker string, params url.Values) ([]Event, *Pagination, *Response, error) {
	return ListPage[Event](ctx, c, "/companies/"+url.PathEscape(idOrTicker)+"/events", params, "events")
}

func (c *Client) ListAllCompanyEvents(ctx context.Context, idOrTicker string, params url.Values) ([]Event, *Response, error) {
	return ListAllDeduped[Event](ctx, c, "/companies/"+url.PathEscape(idOrTicker)+"/events", params, "events", eventID)
}

// eventID keys the dedupe that ListAllDeduped applies to the events feeds.
func eventID(e Event) string { return e.ID }
