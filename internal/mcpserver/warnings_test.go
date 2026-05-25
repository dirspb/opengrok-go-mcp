// SPDX-License-Identifier: Apache-2.0

package mcpserver

import "testing"

func TestAppendWarning(t *testing.T) {
	tests := []struct {
		name     string
		existing *string
		msg      string
		want     *string
	}{
		{"nil existing", nil, "first", strPtr("first")},
		{"empty existing", strPtr(""), "first", strPtr("first")},
		{"empty msg keeps existing", strPtr("first"), "", strPtr("first")},
		{"empty msg nil existing", nil, "", nil},
		{"joins with space", strPtr("first"), "second", strPtr("first second")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendWarning(tt.existing, tt.msg)
			switch {
			case got == nil && tt.want == nil:
				// ok
			case got == nil || tt.want == nil:
				t.Fatalf("appendWarning = %v, want %v", got, tt.want)
			case *got != *tt.want:
				t.Fatalf("appendWarning = %q, want %q", *got, *tt.want)
			}
		})
	}
}
