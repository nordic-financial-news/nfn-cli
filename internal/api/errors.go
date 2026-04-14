package api

import "fmt"

// APIError represents an error response from the API, following RFC 9457 problem detail.
type APIError struct {
	StatusCode int    `json:"-"`
	Type       string `json:"type,omitempty"`
	Title      string `json:"title,omitempty"`
	Detail     string `json:"detail,omitempty"`
	Instance   string `json:"instance,omitempty"`
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

func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == 429
}
