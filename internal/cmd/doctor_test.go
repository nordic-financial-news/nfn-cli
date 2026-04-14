package cmd

import (
	"testing"
)

func TestCheckConfigDir(t *testing.T) {
	t.Parallel()

	result := checkConfigDir()
	// The config dir should exist since config.Init() creates it in tests too,
	// but at minimum the function should return a valid result.
	if result.Name != "Config directory" {
		t.Errorf("name = %q, want %q", result.Name, "Config directory")
	}
	if result.Status != statusPass && result.Status != statusWarn {
		t.Errorf("status = %q, want pass or warn", result.Status)
	}
}

func TestCheckCLIVersionDev(t *testing.T) {
	t.Parallel()

	// Version defaults to "dev" in tests
	result := checkCLIVersion()
	if result.Status != statusSkip {
		t.Errorf("status = %q, want %q for dev build", result.Status, statusSkip)
	}
	if result.Detail != "dev build" {
		t.Errorf("detail = %q, want %q", result.Detail, "dev build")
	}
}

func TestDoctorSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		results []checkResult
		want    string
	}{
		{
			name: "all pass",
			results: []checkResult{
				{Status: statusPass},
				{Status: statusPass},
				{Status: statusPass},
			},
			want: "all 3 checks passed",
		},
		{
			name: "with warnings",
			results: []checkResult{
				{Status: statusPass},
				{Status: statusPass},
				{Status: statusWarn},
			},
			want: "2 passed, 1 warning",
		},
		{
			name: "with failures",
			results: []checkResult{
				{Status: statusPass},
				{Status: statusFail},
				{Status: statusWarn},
			},
			want: "1 passed, 1 failed, 1 warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := doctorSummary(tt.results)
			if got != tt.want {
				t.Errorf("doctorSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatusIcon(t *testing.T) {
	t.Parallel()

	// With color
	for _, status := range []string{statusPass, statusFail, statusWarn, statusSkip} {
		icon := statusIcon(status, false)
		if icon == "" {
			t.Errorf("statusIcon(%q, false) returned empty string", status)
		}
	}

	// Without color — should return plain status text
	for _, status := range []string{statusPass, statusFail, statusWarn, statusSkip} {
		icon := statusIcon(status, true)
		if icon != status {
			t.Errorf("statusIcon(%q, true) = %q, want %q", status, icon, status)
		}
	}
}
