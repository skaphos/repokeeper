// SPDX-License-Identifier: MIT
//go:build integration

package compatibility

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadVerifiedRemovesFailedDownload(t *testing.T) {
	t.Parallel()
	payload := []byte("verified archive")
	client := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(bytes.NewReader(payload)),
			Header:     make(http.Header),
		}, nil
	})}

	destination := filepath.Join(t.TempDir(), "archive")
	err := downloadVerifiedWithClient(context.Background(), client, "https://example.invalid/archive", strings.Repeat("0", sha256.Size*2), destination)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("downloadVerified() error = %v, want checksum mismatch", err)
	}
	if _, statErr := os.Stat(destination); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed download remains at destination: %v", statErr)
	}

	digest := sha256.Sum256(payload)
	if err := downloadVerifiedWithClient(context.Background(), client, "https://example.invalid/archive", hex.EncodeToString(digest[:]), destination); err != nil {
		t.Fatalf("retry downloadVerified() error = %v", err)
	}
	actual, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read verified download: %v", err)
	}
	if string(actual) != string(payload) {
		t.Fatalf("verified download = %q, want %q", actual, payload)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
