package autoupdate

import (
	"testing"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name    string
		latest  string
		current string
		want    bool
	}{
		{
			name:    "newer major version",
			latest:  "2.0.0",
			current: "1.0.0",
			want:    true,
		},
		{
			name:    "newer minor version",
			latest:  "1.2.0",
			current: "1.1.0",
			want:    true,
		},
		{
			name:    "newer patch version",
			latest:  "1.0.1",
			current: "1.0.0",
			want:    true,
		},
		{
			name:    "same version",
			latest:  "1.0.0",
			current: "1.0.0",
			want:    false,
		},
		{
			name:    "older version",
			latest:  "1.0.0",
			current: "2.0.0",
			want:    false,
		},
		{
			name:    "with v prefix on latest",
			latest:  "v1.2.0",
			current: "1.1.0",
			want:    true,
		},
		{
			name:    "with v prefix on both",
			latest:  "v1.2.0",
			current: "v1.1.0",
			want:    true,
		},
		{
			name:    "invalid latest version",
			latest:  "invalid",
			current: "1.0.0",
			want:    false,
		},
		{
			name:    "invalid current version",
			latest:  "1.0.0",
			current: "invalid",
			want:    false,
		},
		{
			name:    "complex newer version",
			latest:  "1.10.5",
			current: "1.9.99",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNewer(tt.latest, tt.current)
			if got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
			}
		})
	}
}
