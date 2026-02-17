// SPDX-License-Identifier: MIT
package repokeeper

import "testing"

func TestParseLabelSelector(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{name: "empty", input: "", wantLen: 0},
		{name: "exists", input: "team", wantLen: 1},
		{name: "equals", input: "team=platform", wantLen: 1},
		{name: "and", input: "team=platform,env=prod", wantLen: 2},
		{name: "invalid key", input: "bad key=1", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLabelSelector(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tc.wantLen {
				t.Fatalf("requirements length = %d, want %d", len(got), tc.wantLen)
			}
		})
	}
}

func TestLabelsMatchSelector(t *testing.T) {
	labels := map[string]string{
		"team": "platform",
		"env":  "prod",
	}
	reqs, err := parseLabelSelector("team=platform,env")
	if err != nil {
		t.Fatalf("parse selector: %v", err)
	}
	if !labelsMatchSelector(labels, reqs) {
		t.Fatal("expected selector match")
	}
	reqs, err = parseLabelSelector("team=app")
	if err != nil {
		t.Fatalf("parse selector: %v", err)
	}
	if labelsMatchSelector(labels, reqs) {
		t.Fatal("expected selector mismatch")
	}
}
