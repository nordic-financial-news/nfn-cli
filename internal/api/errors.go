package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// APIError represents an error response from the API, following RFC 9457 problem detail.
type APIError struct {
	StatusCode int    `json:"-"`
	Type       string `json:"type,omitempty"`
	Title      string `json:"title,omitempty"`
	Detail     string `json:"detail,omitempty"`
	Instance   string `json:"instance,omitempty"`
	// RetryAfter is the wait advertised by the response's Retry-After header,
	// zero when it sent none. It is the value as received, uncapped, so a
	// caller can report an hour-long window truthfully.
	RetryAfter time.Duration `json:"-"`
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	if e.Title != "" {
		return e.Title
	}
	return fmt.Sprintf("API error: HTTP %d", e.StatusCode)
}

func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

func (e *APIError) IsForbidden() bool {
	return e.StatusCode == 403
}

func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == 429
}

// IsUnknownParameter reports whether the API rejected the request because it
// did not recognise the named query parameter. This is what a caller receives
// for a parameter gated behind a feature flag their key is not part of yet, so
// it is distinguishable from a parameter that is simply malformed.
func (e *APIError) IsUnknownParameter(name string) bool {
	if e.StatusCode != http.StatusBadRequest && e.StatusCode != http.StatusUnprocessableEntity {
		return false
	}
	msg := strings.ToLower(e.Title + " " + e.Detail)
	if !strings.Contains(msg, strings.ToLower(name)) || !strings.Contains(msg, "parameter") {
		return false
	}
	for _, adjective := range []string{"unknown", "unrecognized", "unrecognised", "invalid", "unsupported"} {
		if strings.Contains(msg, adjective) {
			return true
		}
	}
	return false
}

// IsSemanticUnavailable reports whether a semantic search failed because the
// embedding backend was unreachable. The API answers 503 with problem type
// "semantic-unavailable" rather than quietly falling back to lexical matching,
// so a 200 always means a genuine semantic result. The type is matched as a
// substring because it is delivered as a URI reference.
func (e *APIError) IsSemanticUnavailable() bool {
	return e.StatusCode == http.StatusServiceUnavailable &&
		strings.Contains(e.Type, "semantic-unavailable")
}
