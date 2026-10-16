package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func localImageProvider(endpoint string) imageProvider {
	return imageProvider{APIKey: "test-key", Endpoint: endpoint, Model: "test-image", Timeout: 5 * time.Second}
}

func TestImageTimeoutConfiguration(t *testing.T) {
	if defaultImageTimeout != 600*time.Second {
		t.Fatalf("default image timeout = %v; want 10 minutes", defaultImageTimeout)
	}
	for _, tc := range []struct {
		raw     string
		want    time.Duration
		invalid bool
	}{
		{"", 600 * time.Second, false},
		{" 600 ", 600 * time.Second, false},
		{"30", 30 * time.Second, false},
		{"1800", 1800 * time.Second, false},
		{"29", 0, true},
		{"1801", 0, true},
		{"0", 0, true},
		{"-1", 0, true},
		{"600.5", 0, true},
		{"ten minutes", 0, true},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := imageTimeoutFromValue(tc.raw)
			if tc.invalid {
				if err == nil {
					t.Fatalf("invalid timeout %q accepted: %v", tc.raw, got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("timeout %q = %v, %v; want %v", tc.raw, got, err, tc.want)
			}
		})
	}
	provider, err := newImageProvider("bai", "test-key", "", "", "")
	if err != nil || provider.Timeout != defaultImageTimeout {
		t.Fatalf("new provider timeout = %v, err = %v", provider.Timeout, err)
	}
}

func TestImageTimeoutReadsYAMLAndEnvironment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-config.yaml")
	config := "image_model:\n  provider: bai\n  api_key: test-key\n  timeout_seconds: 900\n"
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	provider, configured, err := imageProviderFromConfigFile(path)
	if err != nil || !configured || provider.Timeout != 900*time.Second {
		t.Fatalf("YAML timeout = %v, configured = %v, err = %v", provider.Timeout, configured, err)
	}
	t.Chdir(dir)
	provider, err = imageProviderFromEnv(func(key string) string {
		switch key {
		case "BAI_API_KEY":
			return "test-key"
		case "BAI_IMAGE_TIMEOUT_SECONDS":
			return "1200"
		default:
			return ""
		}
	})
	if err != nil || provider.Timeout != 1200*time.Second {
		t.Fatalf("environment timeout = %v, err = %v", provider.Timeout, err)
	}
}

func TestImageHeaderTimeoutIsUnknownAndNeverRetried(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("Authorization") != "Bearer test-key" || r.Method != http.MethodPost {
			t.Errorf("unexpected image request method/auth")
		}
		select {
		case <-r.Context().Done():
		case <-time.After(300 * time.Millisecond):
			_, _ = io.WriteString(w, `{"data":[]}`)
		}
	}))
	defer server.Close()
	provider := localImageProvider(server.URL)
	provider.Timeout = 30 * time.Millisecond
	started := time.Now()
	_, err := postImageJSON(context.Background(), provider, []byte(`{"prompt":"test"}`))
	if err == nil || imageGenerationStatus(err) != "unknown" {
		t.Fatalf("timeout must be unknown: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("image timeout sent %d requests; want one", requests.Load())
	}
	if time.Since(started) > time.Second {
		t.Fatal("image timeout unexpectedly entered retry backoff")
	}
}

func TestImageEOFIsUnknownAndNeverRetried(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		connection, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = connection.Close()
	}))
	defer server.Close()
	_, err := postImageJSON(context.Background(), localImageProvider(server.URL), []byte(`{}`))
	if err == nil || imageGenerationStatus(err) != "unknown" {
		t.Fatalf("EOF must be unknown: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("EOF sent %d requests; want one", requests.Load())
	}
}

func TestImageSharedIdleConnectionCannotTriggerTransportReplay(t *testing.T) {
	var posts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, "warm")
			return
		}
		if posts.Add(1) == 1 {
			connection, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			_ = connection.Close()
			return
		}
		_, _ = io.WriteString(w, `{"data":[{"url":"https://example.invalid/already-generated.png"}]}`)
	}))
	defer server.Close()
	// The default transport is shared by clients whose Transport is nil.
	// Its replay rules treat Idempotency-Key POSTs as idempotent, even though
	// the image provider has not guaranteed idempotency or billing semantics.
	warm, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, warm.Body)
	_ = warm.Body.Close()
	_, err = postImageJSON(context.Background(), localImageProvider(server.URL), []byte(`{}`))
	if posts.Load() != 1 {
		t.Fatalf("HTTP transport silently replayed image POST %d times", posts.Load())
	}
	if err == nil || imageGenerationStatus(err) != "unknown" {
		t.Fatalf("disconnected generated image must be unknown: %v", err)
	}
}

func TestImageBodyTimeoutIsUnknownAndNeverRetried(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-time.After(300 * time.Millisecond):
			_, _ = io.WriteString(w, `{"data":[]}`)
		}
	}))
	defer server.Close()
	provider := localImageProvider(server.URL)
	provider.Timeout = 30 * time.Millisecond
	_, err := postImageJSON(context.Background(), provider, []byte(`{}`))
	if err == nil || imageGenerationStatus(err) != "unknown" {
		t.Fatalf("body timeout must be unknown: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("body timeout sent %d requests; want one", requests.Load())
	}
}

func TestTruncatedImageBodyIsUnknownAndNeverRetried(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Length", "1000")
		_, _ = io.WriteString(w, `{"data":[`)
	}))
	defer server.Close()
	_, err := postImageJSON(context.Background(), localImageProvider(server.URL), []byte(`{}`))
	if err == nil || imageGenerationStatus(err) != "unknown" {
		t.Fatalf("truncated body must be unknown: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("truncated body sent %d requests; want one", requests.Load())
	}
}

func TestImageRedirectDoesNotResubmitPOST(t *testing.T) {
	for _, status := range []int{http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.URL.Path == "/target" {
					_, _ = io.WriteString(w, `{"data":[]}`)
					return
				}
				w.Header().Set("Location", "/target")
				w.WriteHeader(status)
			}))
			defer server.Close()
			_, err := postImageJSON(context.Background(), localImageProvider(server.URL), []byte(`{}`))
			if err == nil {
				t.Fatal("redirect must not be treated as a generated image")
			}
			if requests.Load() != 1 {
				t.Fatalf("redirect resent POST: %d requests", requests.Load())
			}
		})
	}
}

func TestImageHTTPFailureClassificationDoesNotRetry(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   string
	}{
		{400, "failed"}, {401, "failed"}, {403, "failed"}, {429, "failed"},
		{408, "unknown"}, {500, "unknown"}, {502, "unknown"}, {503, "unknown"}, {504, "unknown"},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, `{"error":"local test failure"}`)
			}))
			defer server.Close()
			_, err := postImageJSON(context.Background(), localImageProvider(server.URL), []byte(`{}`))
			if err == nil || imageGenerationStatus(err) != tc.want {
				t.Fatalf("HTTP %d status = %q, err = %v; want %q", tc.status, imageGenerationStatus(err), err, tc.want)
			}
			if requests.Load() != 1 {
				t.Fatalf("HTTP %d sent %d requests; want one", tc.status, requests.Load())
			}
		})
	}
}

func TestImageResponseOverEightMiBIsNotTruncated(t *testing.T) {
	body := []byte(`{"data":[{"b64_json":"` + strings.Repeat("A", 9<<20) + `"}]}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()
	result, err := postImageJSON(context.Background(), localImageProvider(server.URL), []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.Body, body) || !json.Valid(result.Body) {
		t.Fatalf("image JSON truncated: got %d bytes, want %d", len(result.Body), len(body))
	}
}

type imageLimitTestReader struct{}

func (imageLimitTestReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

func TestOversizedImageResponseIsUnknown(t *testing.T) {
	if maxImageResponseBytes != 48<<20 {
		t.Fatalf("response limit = %d; want 48 MiB", maxImageResponseBytes)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = io.CopyN(w, imageLimitTestReader{}, int64(maxImageResponseBytes)+1)
	}))
	defer server.Close()
	provider := localImageProvider(server.URL)
	provider.Timeout = 5 * time.Second
	_, err := postImageJSON(context.Background(), provider, []byte(`{}`))
	if err == nil || imageGenerationStatus(err) != "unknown" {
		t.Fatalf("oversized response must be unknown: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("oversized response sent %d requests; want one", requests.Load())
	}
}

func TestUnusableGeneratedImageIsUnknownAndNeverRetried(t *testing.T) {
	for _, body := range []string{
		`{"data":[`, `{}`, `{"data":[]}`, `{"data":[{}]}`,
		`{"data":[{"url":"javascript:alert(1)"}]}`,
		`{"data":[{"url":"https:broken"}]}`,
		`{"data":[{"url":"https:///missing-host"}]}`,
		`{"data":[{"b64_json":"not-a-base64-png"}]}`,
		`{"data":[{"b64_json":"YWJj"}]}`,
	} {
		t.Run(body, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				_, _ = io.WriteString(w, body)
			}))
			defer server.Close()
			_, err := generatePRDImage(context.Background(), localImageProvider(server.URL), DeliveryCopilot{}, GeneratedPRDImage{ID: "prototype"}, "local-test")
			if err == nil || imageGenerationStatus(err) != "unknown" {
				t.Fatalf("unusable generated image must be unknown: %v", err)
			}
			if requests.Load() != 1 {
				t.Fatalf("unusable image sent %d requests; want one", requests.Load())
			}
		})
	}
}

func TestImageSaveFailureDoesNotResubmit(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("data", []byte("block generated image directory"), 0600); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(w, `{"data":[{"b64_json":"iVBORw0KGgo="}]}`)
	}))
	defer server.Close()
	_, err := generatePRDImage(context.Background(), localImageProvider(server.URL), DeliveryCopilot{}, GeneratedPRDImage{ID: "prototype"}, "local-test")
	if err == nil || imageGenerationStatus(err) != "unknown" {
		t.Fatalf("save failure must preserve unknown provider result: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("save failure sent %d requests; want one", requests.Load())
	}
}

func TestTextModelConnectionErrorCountsAttemptsAndRetriesSeparately(t *testing.T) {
	for _, err := range []error{io.EOF, context.DeadlineExceeded} {
		got := strings.ReplaceAll(modelConnectionError("报告模型", 3, err).Error(), " ", "")
		if !strings.Contains(got, "共尝试3次") || !strings.Contains(got, "重试2次") {
			t.Fatalf("text retry message miscounts attempts: %s", got)
		}
	}
}
