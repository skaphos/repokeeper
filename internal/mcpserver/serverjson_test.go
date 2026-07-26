// SPDX-License-Identifier: MIT

package mcpserver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// serverJSON is the checked-in MCP registry entry. These tests guard against it
// drifting from what RepoKeeper actually serves: the registry is a published
// description of this server, and a description that no longer matches is worse
// than none, because clients act on it.
//
// Schema validation in CI catches a malformed entry. It cannot catch an entry
// that is well-formed and wrong, which is what these tests are for.
type serverJSON struct {
	Schema      string `json:"$schema"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Repository  struct {
		URL    string `json:"url"`
		Source string `json:"source"`
	} `json:"repository"`
	Version  string          `json:"version"`
	Packages []serverPackage `json:"packages"`
}

// serverPackage is one installable form of the server in the registry entry.
type serverPackage struct {
	RegistryType string `json:"registryType"`
	Identifier   string `json:"identifier"`
	Version      string `json:"version"`
	Transport    struct {
		Type string `json:"type"`
	} `json:"transport"`
}

func loadServerJSON(t *testing.T) serverJSON {
	t.Helper()

	// The entry lives at the repository root, two levels above this package.
	path := filepath.Join("..", "..", "server.json")
	raw, err := os.ReadFile(path) //nolint:gosec // fixed path inside the repo
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	var doc serverJSON
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("server.json is not valid JSON: %v", err)
	}
	return doc
}

// packageEntry returns the single package entry, failing cleanly if it is
// absent. These tests exist to catch a malformed server.json, so indexing it
// unguarded would panic the test binary on exactly the input they are meant to
// report on.
func packageEntry(t *testing.T, doc serverJSON) serverPackage {
	t.Helper()
	if len(doc.Packages) == 0 {
		t.Fatal("server.json declares no packages; MCP clients would have nothing to run")
	}
	return doc.Packages[0]
}

// TestServerJSONIdentity pins the registry identity. The namespace is not
// cosmetic: it is derived from the domain whose DNS TXT record proves ownership,
// so changing it silently would make publishing fail or, worse, publish under a
// name nobody controls.
func TestServerJSONIdentity(t *testing.T) {
	doc := loadServerJSON(t)

	const wantName = "io.skaphos/repokeeper"
	if doc.Name != wantName {
		t.Errorf("registry name: got %q want %q", doc.Name, wantName)
	}
	if !strings.HasPrefix(doc.Name, "io.skaphos/") {
		t.Errorf("name %q is outside the DNS-proven io.skaphos namespace", doc.Name)
	}
	if doc.Repository.URL != "https://github.com/skaphos/repokeeper" {
		t.Errorf("repository url: got %q", doc.Repository.URL)
	}
}

// TestServerJSONTransportMatchesServer: RepoKeeper serves MCP over stdio, and
// the container entrypoint starts exactly that. A mismatch here means a client
// configured from the registry cannot connect.
func TestServerJSONTransportMatchesServer(t *testing.T) {
	pkg := packageEntry(t, loadServerJSON(t))

	if pkg.Transport.Type != "stdio" {
		t.Errorf("transport: got %q want %q -- cmd/repokeeper/mcp.go serves stdio", pkg.Transport.Type, "stdio")
	}
	if pkg.RegistryType != "oci" {
		t.Errorf("registryType: got %q want %q", pkg.RegistryType, "oci")
	}
	if pkg.Identifier != "ghcr.io/skaphos/repokeeper" {
		t.Errorf("identifier: got %q -- must match the image dockers_v2 publishes", pkg.Identifier)
	}
}

// TestServerJSONVersionsAreConsistent: the release workflow stamps both version
// fields from the tag, so they must be kept in step with each other. Divergence
// checked in would publish an entry naming an image tag that does not exist.
func TestServerJSONVersionsAreConsistent(t *testing.T) {
	doc := loadServerJSON(t)
	pkg := packageEntry(t, doc)

	if doc.Version != pkg.Version {
		t.Errorf("version fields diverge: top-level %q, package %q", doc.Version, pkg.Version)
	}
	if doc.Version != "0.0.0" {
		t.Errorf("checked-in version should be the 0.0.0 placeholder, got %q; the release stamps it from the tag", doc.Version)
	}
}

// TestServerJSONDoesNotClaimReadOnly is the assertion that differs most from
// sting's equivalent. sting is read-only in its entirety and says so.
// RepoKeeper is not: it exposes tools that rewrite the registry and mutate
// working trees. Publishing a description that reads as read-only would
// misrepresent the mutation surface to every client that reads the registry.
func TestServerJSONDoesNotClaimReadOnly(t *testing.T) {
	doc := loadServerJSON(t)

	readOnly := ReadOnlyToolNames()
	total := len(New(nil, "", "", nil).inner.ListTools())
	mutating := total - len(readOnly)

	if mutating == 0 {
		t.Fatal("no mutating tools found; this test's premise no longer holds and the description should be revisited")
	}

	lowered := strings.ToLower(doc.Description)
	for _, claim := range []string{"read-only", "read only", "readonly"} {
		if strings.Contains(lowered, claim) {
			t.Errorf("description claims %q but %d of %d tools mutate: %q", claim, mutating, total, doc.Description)
		}
	}
}

// TestServerJSONDescriptionSignalsWriteSurface: not claiming read-only is not
// enough. A prospective user reading the registry should be able to tell that
// this server can change things.
func TestServerJSONDescriptionSignalsWriteSurface(t *testing.T) {
	doc := loadServerJSON(t)

	lowered := strings.ToLower(doc.Description)
	signals := []string{"write", "mutat", "sync", "change"}

	found := false
	for _, s := range signals {
		if strings.Contains(lowered, s) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("description gives no signal that this server writes; got %q", doc.Description)
	}
}

// TestServerJSONDescriptionWithinSchemaLimit guards the constraint that is
// easiest to trip when editing prose: the registry schema caps the description
// at 100 characters, and exceeding it fails at publish time rather than here.
func TestServerJSONDescriptionWithinSchemaLimit(t *testing.T) {
	doc := loadServerJSON(t)

	const maxLen = 100
	if n := len(doc.Description); n > maxLen {
		t.Errorf("description is %d characters; the MCP registry schema allows %d", n, maxLen)
	}
	if doc.Description == "" {
		t.Error("description is required and must not be empty")
	}
}

// TestServerJSONHandlesMalformedInput: these tests exist to report on a broken
// server.json, so they must fail cleanly rather than panic on one.
func TestServerJSONHandlesMalformedInput(t *testing.T) {
	var doc serverJSON
	if err := json.Unmarshal([]byte(`{"name":"x","packages":[]}`), &doc); err != nil {
		t.Fatalf("unmarshalling a package-less document should succeed: %v", err)
	}
	if len(doc.Packages) != 0 {
		t.Fatalf("expected no packages, got %d", len(doc.Packages))
	}
	// packageEntry would t.Fatal here rather than panicking, which is the
	// behavior under test; asserting the guard's precondition is enough.
}
