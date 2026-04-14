package config

import "testing"

func TestGetAPIKey_FromEnv(t *testing.T) {
	t.Setenv("NFN_API_KEY", "test-key-from-env")

	key, err := GetAPIKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "test-key-from-env" {
		t.Errorf("key = %q, want %q", key, "test-key-from-env")
	}
}

func TestGetAPIKey_EnvTakesPrecedence(t *testing.T) {
	// When NFN_API_KEY is set, it should be used regardless of keyring state
	t.Setenv("NFN_API_KEY", "env-key")

	key, err := GetAPIKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "env-key" {
		t.Errorf("key = %q, want %q", key, "env-key")
	}

	// Change the env var — should get the new value
	t.Setenv("NFN_API_KEY", "different-key")
	key, err = GetAPIKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "different-key" {
		t.Errorf("key = %q, want %q", key, "different-key")
	}
}

func TestInit_DefaultBaseURL(t *testing.T) {
	Init()

	got := GetBaseURL()
	want := "https://nordicfinancialnews.com/api/v1"
	if got != want {
		t.Errorf("GetBaseURL() = %q, want %q", got, want)
	}
}

func TestInit_DefaultFormat(t *testing.T) {
	Init()

	got := GetFormat()
	if got != "table" {
		t.Errorf("GetFormat() = %q, want %q", got, "table")
	}
}
