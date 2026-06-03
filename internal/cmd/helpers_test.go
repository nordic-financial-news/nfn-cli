package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
)

func TestWatchlistFilterError(t *testing.T) {
	t.Parallel()

	t.Run("403 with watchlist filter becomes scope message", func(t *testing.T) {
		t.Parallel()
		got := watchlistFilterError(&api.APIError{StatusCode: 403, Detail: "Forbidden"}, "wl_abc")
		if got == nil || !strings.Contains(got.Error(), "read:watchlist scope") {
			t.Errorf("expected read:watchlist scope message, got %v", got)
		}
	})

	t.Run("403 without watchlist filter passes through", func(t *testing.T) {
		t.Parallel()
		orig := &api.APIError{StatusCode: 403, Detail: "Forbidden"}
		if got := watchlistFilterError(orig, ""); got != orig {
			t.Errorf("expected original error to pass through, got %v", got)
		}
	})

	t.Run("non-403 API error passes through", func(t *testing.T) {
		t.Parallel()
		orig := &api.APIError{StatusCode: 500, Detail: "boom"}
		if got := watchlistFilterError(orig, "wl_abc"); got != orig {
			t.Errorf("expected original error to pass through, got %v", got)
		}
	})

	t.Run("plain error passes through", func(t *testing.T) {
		t.Parallel()
		orig := errors.New("network down")
		if got := watchlistFilterError(orig, "wl_abc"); got != orig {
			t.Errorf("expected original error to pass through, got %v", got)
		}
	})

	t.Run("nil error stays nil", func(t *testing.T) {
		t.Parallel()
		if got := watchlistFilterError(nil, "wl_abc"); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func TestIsTrustedHost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"production", "https://nordicfinancialnews.com/api/v1", true},
		{"subdomain", "https://api.nordicfinancialnews.com/v1", true},
		{"deep subdomain", "https://staging.api.nordicfinancialnews.com/v1", true},
		{"different domain", "https://evil.com/api/v1", false},
		{"suffix trick", "https://notnordicfinancialnews.com/api/v1", false},
		{"subdomain trick", "https://nordicfinancialnews.com.evil.com/api/v1", false},
		{"empty", "", false},
		{"http", "http://nordicfinancialnews.com/api/v1", true},
		{"with port", "https://nordicfinancialnews.com:8443/api/v1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isTrustedHost(tt.url)
			if got != tt.want {
				t.Errorf("isTrustedHost(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}
