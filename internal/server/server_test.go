package server

import "testing"

func TestShouldVaryAccept(t *testing.T) {
	tests := []struct {
		name            string
		sourceMediaType string
		explicitFormat  string
		expected        bool
	}{
		{
			name:            "image source without explicit format",
			sourceMediaType: "image/png",
			explicitFormat:  "",
			expected:        true,
		},
		{
			name:            "image source with explicit format",
			sourceMediaType: "image/png",
			explicitFormat:  "jpeg",
			expected:        false,
		},
		{
			name:            "non-image source",
			sourceMediaType: "text/html; charset=utf-8",
			explicitFormat:  "",
			expected:        false,
		},
		{
			name:            "media type normalization",
			sourceMediaType: " IMAGE/JPEG ",
			explicitFormat:  "",
			expected:        true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := shouldVaryAccept(tt.sourceMediaType, tt.explicitFormat)
			if got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}
