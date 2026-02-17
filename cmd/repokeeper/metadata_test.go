// SPDX-License-Identifier: MIT
package repokeeper

import "testing"

func TestParseMetadataAssignments(t *testing.T) {
	got, err := parseMetadataAssignments([]string{"team=platform", "env=prod"}, "--label")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["team"] != "platform" || got["env"] != "prod" {
		t.Fatalf("unexpected parsed map: %#v", got)
	}

	if _, err := parseMetadataAssignments([]string{"bad key=x"}, "--label"); err == nil {
		t.Fatal("expected invalid key error")
	}
	if _, err := parseMetadataAssignments([]string{"novalue"}, "--label"); err == nil {
		t.Fatal("expected missing equals error")
	}
}

func TestParseMetadataKeys(t *testing.T) {
	got, err := parseMetadataKeys([]string{"team", "env"}, "--remove-label")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "team" || got[1] != "env" {
		t.Fatalf("unexpected keys: %#v", got)
	}

	if _, err := parseMetadataKeys([]string{"bad key"}, "--remove-label"); err == nil {
		t.Fatal("expected invalid key error")
	}
}
