package api

import "testing"

func TestAPIError_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{
			name: "detail present",
			err:  APIError{StatusCode: 400, Detail: "invalid parameter"},
			want: "invalid parameter",
		},
		{
			name: "title only",
			err:  APIError{StatusCode: 404, Title: "Not Found"},
			want: "Not Found",
		},
		{
			name: "fallback to status code",
			err:  APIError{StatusCode: 500},
			want: "API error: HTTP 500",
		},
		{
			name: "detail takes precedence over title",
			err:  APIError{StatusCode: 400, Title: "Bad Request", Detail: "missing field"},
			want: "missing field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIError_IsUnauthorized(t *testing.T) {
	t.Parallel()
	tests := []struct {
		code int
		want bool
	}{
		{401, true},
		{403, false},
		{200, false},
		{429, false},
	}

	for _, tt := range tests {
		e := &APIError{StatusCode: tt.code}
		if got := e.IsUnauthorized(); got != tt.want {
			t.Errorf("IsUnauthorized() for %d = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestAPIError_IsRateLimited(t *testing.T) {
	t.Parallel()
	tests := []struct {
		code int
		want bool
	}{
		{429, true},
		{401, false},
		{200, false},
		{500, false},
	}

	for _, tt := range tests {
		e := &APIError{StatusCode: tt.code}
		if got := e.IsRateLimited(); got != tt.want {
			t.Errorf("IsRateLimited() for %d = %v, want %v", tt.code, got, tt.want)
		}
	}
}
