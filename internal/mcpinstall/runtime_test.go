// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"errors"
	"reflect"
	"testing"
)

type fakeRuntime struct {
	name     string
	detected bool
	detErr   error
}

func (f *fakeRuntime) Name() string                          { return f.name }
func (f *fakeRuntime) Detect() (bool, error)                 { return f.detected, f.detErr }
func (f *fakeRuntime) ConfigPath(Scope) (string, error)      { return "", nil }
func (f *fakeRuntime) ReadEntry(string) (Entry, bool, error) { return Entry{}, false, nil }
func (f *fakeRuntime) WriteEntry(string, Entry) error        { return nil }
func (f *fakeRuntime) RemoveEntry(string) (bool, error)      { return false, nil }

// withFakes swaps the package-level `registered` slice for the
// duration of a test. Cannot be used with t.Parallel since it mutates
// shared state.
func withFakes(t *testing.T, fakes ...Runtime) {
	t.Helper()
	prev := registered
	t.Cleanup(func() { registered = prev })
	registered = nil
	for _, f := range fakes {
		register(f)
	}
}

func TestAllReturnsSortedCopy(t *testing.T) {
	t.Parallel()
	got := All()
	for i := 1; i < len(got); i++ {
		if got[i-1].Name() > got[i].Name() {
			t.Fatalf("All() not sorted: %v", names(got))
		}
	}
}

func TestByNameMiss(t *testing.T) {
	t.Parallel()
	if _, ok := ByName("does-not-exist-xyz"); ok {
		t.Fatal("expected ByName to miss")
	}
}

func TestSelectionFromFlagsEmpty(t *testing.T) {
	t.Parallel()
	s := SelectionFromFlags(false, false, false)
	if len(s.Explicit) != 0 {
		t.Fatalf("expected empty selection, got %v", s.Explicit)
	}
}

func TestSelectionFromFlagsAll(t *testing.T) {
	t.Parallel()
	s := SelectionFromFlags(true, true, true)
	want := []string{"claude", "codex", "opencode"}
	if !reflect.DeepEqual(s.Explicit, want) {
		t.Fatalf("got %v want %v", s.Explicit, want)
	}
}

func TestResolveExplicit(t *testing.T) {
	withFakes(t, &fakeRuntime{name: "a"}, &fakeRuntime{name: "b"})
	got, err := Selection{Explicit: []string{"b", "a"}}.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if names(got)[0] != "b" || names(got)[1] != "a" {
		t.Fatalf("expected Explicit order preserved, got %v", names(got))
	}
}

func TestResolveExplicitUnknown(t *testing.T) {
	withFakes(t, &fakeRuntime{name: "a"})
	_, err := Selection{Explicit: []string{"nope"}}.Resolve()
	if err == nil {
		t.Fatal("expected error for unknown runtime")
	}
}

func TestResolveAutoDetect(t *testing.T) {
	withFakes(t,
		&fakeRuntime{name: "present", detected: true},
		&fakeRuntime{name: "absent", detected: false},
	)
	got, err := Selection{}.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name() != "present" {
		t.Fatalf("expected only 'present', got %v", names(got))
	}
}

func TestResolveAutoDetectError(t *testing.T) {
	withFakes(t, &fakeRuntime{name: "broken", detErr: errors.New("boom")})
	_, err := Selection{}.Resolve()
	if err == nil {
		t.Fatal("expected Detect error to propagate")
	}
}

func TestAllSortsFakeRuntimes(t *testing.T) {
	withFakes(t,
		&fakeRuntime{name: "gamma"},
		&fakeRuntime{name: "alpha"},
		&fakeRuntime{name: "beta"},
	)
	got := All()
	if len(got) != 3 {
		t.Fatalf("expected 3 runtimes, got %d", len(got))
	}
	want := []string{"alpha", "beta", "gamma"}
	for i, r := range got {
		if r.Name() != want[i] {
			t.Fatalf("index %d: got %q want %q (full order: %v)", i, r.Name(), want[i], names(got))
		}
	}
}

func TestByNameHit(t *testing.T) {
	withFakes(t, &fakeRuntime{name: "alpha"}, &fakeRuntime{name: "beta"})
	r, ok := ByName("beta")
	if !ok {
		t.Fatal("expected hit")
	}
	if r.Name() != "beta" {
		t.Fatalf("got %q want beta", r.Name())
	}
}

func TestByNameMissAgainstNonEmpty(t *testing.T) {
	withFakes(t, &fakeRuntime{name: "alpha"})
	if _, ok := ByName("beta"); ok {
		t.Fatal("expected miss against non-empty registry")
	}
}

func TestScopeString(t *testing.T) {
	t.Parallel()
	cases := []struct {
		s    Scope
		want string
	}{
		{ScopeUser, "user"},
		{ScopeProject, "project"},
		{Scope(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.s.String(); got != tc.want {
			t.Fatalf("Scope(%d).String(): got %q want %q", int(tc.s), got, tc.want)
		}
	}
}

func names(rs []Runtime) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Name()
	}
	return out
}
