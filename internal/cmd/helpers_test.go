package cmd

import (
	"testing"
)

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
