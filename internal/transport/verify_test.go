package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Note: md5Hash is defined in test_helpers_test.go

func TestVerifyMD5Logic(t *testing.T) {
	data := "test data content"
	expectedHash := md5Hash(data)
	if expectedHash != md5Hash(data) {
		t.Error("MD5 hash mismatch")
	}
	if md5Hash("different data") == expectedHash {
		t.Error("different data should produce different hash")
	}
}

func TestVerifyMD5(t *testing.T) {
	dataContent := sampleDelegatedData
	dataHash := md5Hash(dataContent)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, ".md5") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(dataHash + "  delegated-apnic-latest"))
		} else {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(dataContent))
		}
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	err := client.VerifyMD5(context.Background(), "delegated", "")
	if err != nil {
		t.Fatalf("VerifyMD5() error: %v", err)
	}
}

func TestVerifyMD5Mismatch(t *testing.T) {
	dataContent := sampleDelegatedData

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, ".md5") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("00000000000000000000000000000000  delegated-apnic-latest"))
		} else {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(dataContent))
		}
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	err := client.VerifyMD5(context.Background(), "delegated", "")
	if err == nil {
		t.Error("expected error for MD5 mismatch")
	}
}

func TestVerifyMD5EmptyChecksumFile(t *testing.T) {
	dataContent := sampleDelegatedData

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, ".md5") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(""))
		} else {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(dataContent))
		}
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	err := client.VerifyMD5(context.Background(), "delegated", "")
	if err == nil {
		t.Error("expected error for empty MD5 checksum file")
	}
}

func TestVerifyMD5DataFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	err := client.VerifyMD5(context.Background(), "delegated", "")
	if err == nil {
		t.Error("expected error for data fetch failure")
	}
}

func TestVerifyMD5ChecksumFetchError(t *testing.T) {
	dataContent := sampleDelegatedData

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".md5") {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Write([]byte(dataContent))
		}
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	err := client.VerifyMD5(context.Background(), "delegated", "")
	if err == nil {
		t.Error("expected error for MD5 fetch failure")
	}
}

func TestFetchMD5Checksum(t *testing.T) {
	// GNU-style checksum file: "<hash>  filename"
	expectedHash := "ad1b2eeeb7986f4f600a14ed4470b7ef"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(expectedHash + "  delegated-apnic-latest"))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	hash, err := client.FetchMD5Checksum(context.Background(), "delegated", "")
	if err != nil {
		t.Fatalf("FetchMD5Checksum() error: %v", err)
	}
	if hash != expectedHash {
		t.Errorf("hash = %q, want %q", hash, expectedHash)
	}
}

func TestFetchMD5ChecksumBSDStyle(t *testing.T) {
	// BSD-style checksum file as published by APNIC: "MD5 (file) = <hash>"
	expectedHash := "ad1b2eeeb7986f4f600a14ed4470b7ef"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("MD5 (delegated-apnic-latest) = " + expectedHash))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	hash, err := client.FetchMD5Checksum(context.Background(), "delegated", "")
	if err != nil {
		t.Fatalf("FetchMD5Checksum() error: %v", err)
	}
	if hash != expectedHash {
		t.Errorf("hash = %q, want %q", hash, expectedHash)
	}
}

func TestFetchMD5ChecksumWithDate(t *testing.T) {
	expectedHash := "ad1b2eeeb7986f4f600a14ed4470b7ef"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveDated(w, r, "MD5 (delegated-apnic-20260627) = "+expectedHash)
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	hash, err := client.FetchMD5Checksum(context.Background(), "delegated", "20260627")
	if err != nil {
		t.Fatalf("FetchMD5Checksum() error: %v", err)
	}
	if hash != expectedHash {
		t.Errorf("hash = %q, want %q", hash, expectedHash)
	}
}

func TestFetchMD5ChecksumEmptyFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(""))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	_, err := client.FetchMD5Checksum(context.Background(), "delegated", "")
	if err == nil {
		t.Error("expected error for empty MD5 file")
	}
}

func TestFetchMD5ChecksumUnparseable(t *testing.T) {
	// Non-empty content with no recognizable 32-hex hash should fail parsing.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("garbage content with no hash here"))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	_, err := client.FetchMD5Checksum(context.Background(), "delegated", "")
	if err == nil {
		t.Error("expected error for unparseable MD5 file")
	}
}

func TestIsMD5Hex(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"ad1b2eeeb7986f4f600a14ed4470b7ef", true},  // lowercase
		{"AD1B2EEEB7986F4F600A14ED4470B7EF", true},  // uppercase
		{"Ad1B2eeeb7986f4f600a14ed4470b7ef", true},  // mixed case
		{"abc123", false},                           // too short
		{"ad1b2eeeb7986f4f600a14ed4470b7eg", false}, // 32 chars but 'g' is not hex
		{"z1b2eeeb7986f4f600a14ed4470b7ef", false},  // 32 chars but 'z' is not hex
		{"", false},
	}
	for _, tt := range tests {
		if got := isMD5Hex(tt.input); got != tt.want {
			t.Errorf("isMD5Hex(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFetchMD5ChecksumUppercaseHex(t *testing.T) {
	expectedHash := "AD1B2EEEB7986F4F600A14ED4470B7EF"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(expectedHash + "  delegated-apnic-latest"))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	hash, err := client.FetchMD5Checksum(context.Background(), "delegated", "")
	if err != nil {
		t.Fatalf("FetchMD5Checksum() error: %v", err)
	}
	if hash != expectedHash {
		t.Errorf("hash = %q, want %q", hash, expectedHash)
	}
}

func TestFetchMD5ChecksumFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	_, err := client.FetchMD5Checksum(context.Background(), "delegated", "")
	if err == nil {
		t.Error("expected error for MD5 fetch failure")
	}
}

func TestFetchASCSignature(t *testing.T) {
	sig := "-----BEGIN PGP SIGNATURE-----\ntest\n-----END PGP SIGNATURE-----"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(sig))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	result, err := client.FetchASCSignature(context.Background(), "delegated", "")
	if err != nil {
		t.Fatalf("FetchASCSignature() error: %v", err)
	}
	if result != sig {
		t.Errorf("signature = %q, want %q", result, sig)
	}
}

func TestFetchASCSignatureFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	_, err := client.FetchASCSignature(context.Background(), "delegated", "")
	if err == nil {
		t.Error("expected error for ASC fetch failure")
	}
}

func TestFetchPublicKey(t *testing.T) {
	key := "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest\n-----END PGP PUBLIC KEY BLOCK-----"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(key))
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	result, err := client.FetchPublicKey(context.Background())
	if err != nil {
		t.Fatalf("FetchPublicKey() error: %v", err)
	}
	if result != key {
		t.Errorf("key = %q, want %q", result, key)
	}
}

func TestFetchPublicKeyFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithStatsBaseURL(server.URL+"/"),
		WithCacheTTL(1*time.Hour),
	)

	_, err := client.FetchPublicKey(context.Background())
	if err == nil {
		t.Error("expected error for public key fetch failure")
	}
}
