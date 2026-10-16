package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func imageJobTestCopilot(id string) DeliveryCopilot {
	return DeliveryCopilot{PRD: EnterprisePRD{Info: PRDDocumentInfo{PRDID: id, Product: "自媒体工作台", Feature: "母版与审批"}}}
}

func imageJobTestProvider() (imageProvider, error) {
	return imageProvider{APIKey: "private-test-image-key", Label: "Offline image test", Timeout: 5 * time.Second}, nil
}

func imageJobTestReady(img GeneratedPRDImage) GeneratedPRDImage {
	img.Status = "ready"
	img.URL = "/api/generated-images/" + img.AttemptID + "-" + img.Kind + ".png"
	img.DownloadURL = img.URL + "?download=1"
	return img
}

func imageJobTestWait(t *testing.T, manager *imageJobManager, id string) ImageGenerationJob {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	job, err := manager.Wait(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return job
}

func TestImageJobsDeduplicateExposePartialResultsAndRecoverLookup(t *testing.T) {
	dir := t.TempDir()
	var calls atomic.Int32
	architectureStarted := make(chan struct{})
	releaseArchitecture := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseArchitecture) }) }
	defer release()
	manager, err := newImageJobManagerWithDependencies(dir, imageJobTestProvider, func(ctx context.Context, provider imageProvider, copilot DeliveryCopilot, img GeneratedPRDImage, prompt string) (GeneratedPRDImage, error) {
		calls.Add(1)
		if img.Kind == "architecture" {
			close(architectureStarted)
			select {
			case <-releaseArchitecture:
			case <-ctx.Done():
				return img, ctx.Err()
			}
		}
		return imageJobTestReady(img), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	copilot := imageJobTestCopilot("dedup")
	job, err := manager.Start(copilot, nil, false, "initial-request-0001")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-architectureStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("second image did not start")
	}
	var group sync.WaitGroup
	for i := 0; i < 12; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			duplicate, startErr := manager.Start(copilot, nil, false, "initial-request-0001")
			if startErr != nil || duplicate.ID != job.ID {
				t.Errorf("duplicate = %s, %v", duplicate.ID, startErr)
			}
		}()
	}
	group.Wait()
	partial, _ := manager.Get(job.ID)
	if partial.Status != "running" || partial.Images[0].Status != "ready" || partial.Images[1].Status != "generating" {
		t.Fatalf("partial result = %+v", partial)
	}
	partial.Images[0].Status = "changed-by-client"
	if original, _ := manager.Get(job.ID); original.Images[0].Status != "ready" {
		t.Fatal("Get returned shared mutable state")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := manager.Wait(ctx, job.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("wait cancellation = %v", err)
	}
	if running, _ := manager.Get(job.ID); running.Status != "running" {
		t.Fatal("disconnect cancelled the background task")
	}
	release()
	completed := imageJobTestWait(t, manager, job.ID)
	if completed.Status != "completed" || completed.Images[1].Status != "ready" || calls.Load() != 2 {
		t.Fatalf("completed = %+v; calls=%d", completed, calls.Load())
	}
	copilot.GeneratedAt = time.Now()
	copilot.PRD.Info.LastUpdated = "tomorrow"
	copilot.GeneratedImages = completed.Images
	duplicate, err := manager.Start(copilot, nil, false, "second-request-0002")
	if err != nil || duplicate.ID != job.ID || calls.Load() != 2 {
		t.Fatalf("metadata changed generation identity: %s, %v, calls=%d", duplicate.ID, err, calls.Load())
	}
	providerCalled := false
	reloaded, err := newImageJobManagerWithDependencies(dir, func() (imageProvider, error) { providerCalled = true; return imageJobTestProvider() }, manager.generate)
	if err != nil {
		t.Fatal(err)
	}
	lookedUp, ok := reloaded.Lookup("second-request-0002")
	if !ok || lookedUp.ID != job.ID || lookedUp.Images[0].Status != "ready" || providerCalled {
		t.Fatal("persisted lookup did not recover without a provider call")
	}
	data, err := os.ReadFile(filepath.Join(dir, job.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private-test-image-key") || strings.Contains(string(data), `"copilot"`) {
		t.Fatal("snapshot persisted a secret or entire PRD")
	}
}

func TestImageJobsEnforceConcurrencyLimit(t *testing.T) {
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	manager, err := newImageJobManagerWithDependencies(t.TempDir(), imageJobTestProvider, func(ctx context.Context, _ imageProvider, _ DeliveryCopilot, img GeneratedPRDImage, _ string) (GeneratedPRDImage, error) {
		select {
		case <-release:
			return imageJobTestReady(img), nil
		case <-ctx.Done():
			return img, ctx.Err()
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := manager.Start(imageJobTestCopilot("one"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.Start(imageJobTestCopilot("two"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Start(imageJobTestCopilot("three"), nil, false); !errors.Is(err, errImageJobBusy) {
		t.Fatalf("third job error = %v", err)
	}
	unblock()
	imageJobTestWait(t, manager, first.ID)
	imageJobTestWait(t, manager, second.ID)
}

func TestImageJobsRestartUnknownRequiresAcknowledgementAndKeepsReadyImage(t *testing.T) {
	dir := t.TempDir()
	var calls atomic.Int32
	generator := func(ctx context.Context, _ imageProvider, _ DeliveryCopilot, img GeneratedPRDImage, _ string) (GeneratedPRDImage, error) {
		calls.Add(1)
		return imageJobTestReady(img), nil
	}
	manager, err := newImageJobManagerWithDependencies(dir, imageJobTestProvider, generator)
	if err != nil {
		t.Fatal(err)
	}
	copilot := imageJobTestCopilot("restart")
	job := ImageGenerationJob{ID: imageJobKey(copilot), Status: "running", Images: queuedPRDImages(), RequestIDs: []string{"before-restart-0001"}, TimeoutSeconds: 600}
	job.Images[0].Status = "ready"
	job.Images[0].URL = "/api/generated-images/keep-original.png"
	job.Images[1].Status = "generating"
	if err := manager.persist(job); err != nil {
		t.Fatal(err)
	}
	recovered, err := newImageJobManagerWithDependencies(dir, imageJobTestProvider, generator)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, ok := recovered.Get(job.ID)
	if !ok || snapshot.Status != "completed" || snapshot.Images[0].Status != "ready" || snapshot.Images[1].Status != "unknown" {
		t.Fatalf("recovery = %+v", snapshot)
	}
	if _, err := recovered.Start(copilot, nil, false); err != nil || calls.Load() != 0 {
		t.Fatalf("recovery auto-generated: %v calls=%d", err, calls.Load())
	}
	if _, err := recovered.Start(copilot, []string{"architecture"}, false, "retry-no-ack-0001"); !errors.Is(err, errImageJobUnknown) {
		t.Fatalf("unknown without acknowledgement = %v", err)
	}
	retried, err := recovered.Start(copilot, []string{"architecture"}, true, "retry-with-ack-0001")
	if err != nil {
		t.Fatal(err)
	}
	finished := imageJobTestWait(t, recovered, retried.ID)
	if calls.Load() != 1 || finished.Images[0].URL != "/api/generated-images/keep-original.png" || finished.Images[1].Status != "ready" {
		t.Fatalf("retry damaged ready image: %+v calls=%d", finished, calls.Load())
	}
	if _, err := recovered.Start(copilot, []string{"architecture"}, true, "retry-with-ack-0001"); err != nil || calls.Load() != 1 {
		t.Fatalf("same retry request was reissued: %v, calls=%d", err, calls.Load())
	}
	if _, err := recovered.Start(copilot, []string{"prototype"}, true, "cannot-retry-ready"); !errors.Is(err, errImageJobInvalid) {
		t.Fatalf("ready retry accepted: %v", err)
	}
}

func TestImageJobsQueuedAndCorruptLedgersStayUnknown(t *testing.T) {
	dir := t.TempDir()
	manager, err := newImageJobManagerWithDependencies(dir, imageJobTestProvider, func(context.Context, imageProvider, DeliveryCopilot, GeneratedPRDImage, string) (GeneratedPRDImage, error) {
		t.Error("must not generate during recovery")
		return GeneratedPRDImage{}, errors.New("not expected")
	})
	if err != nil {
		t.Fatal(err)
	}
	queued := ImageGenerationJob{ID: imageJobKey(imageJobTestCopilot("queued")), Status: "running", Images: queuedPRDImages()}
	if err := manager.persist(queued); err != nil {
		t.Fatal(err)
	}
	corruptID := imageJobKey(imageJobTestCopilot("corrupt"))
	if err := os.WriteFile(filepath.Join(dir, corruptID+".json"), []byte(`{"interrupted":`), 0600); err != nil {
		t.Fatal(err)
	}
	recovered, err := newImageJobManagerWithDependencies(dir, imageJobTestProvider, manager.generate)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{queued.ID, corruptID} {
		job, ok := recovered.Get(id)
		if !ok || job.Status != "completed" || len(job.Images) != 2 {
			t.Fatalf("invalid recovered job: %+v", job)
		}
		for _, img := range job.Images {
			if img.Status != "unknown" {
				t.Fatalf("unsafe recovered image status: %+v", img)
			}
		}
	}
}

func TestImageJobHTTPRoutesAndPathSafety(t *testing.T) {
	var calls atomic.Int32
	manager, err := newImageJobManagerWithDependencies(t.TempDir(), imageJobTestProvider, func(_ context.Context, _ imageProvider, _ DeliveryCopilot, img GeneratedPRDImage, _ string) (GeneratedPRDImage, error) {
		calls.Add(1)
		return imageJobTestReady(img), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerImageJobRoutes(mux, manager)
	request := map[string]any{"copilot": imageJobTestCopilot("http"), "requestId": "http-test-request-001"}
	body, _ := json.Marshal(request)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/delivery-copilot/image-jobs", bytes.NewReader(body)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var job ImageGenerationJob
	if err := json.Unmarshal(recorder.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	imageJobTestWait(t, manager, job.ID)
	for _, path := range []string{"/api/delivery-copilot/image-jobs/" + job.ID, "/api/delivery-copilot/image-jobs/lookup?requestId=http-test-request-001"} {
		result := httptest.NewRecorder()
		mux.ServeHTTP(result, httptest.NewRequest(http.MethodGet, path, nil))
		if result.Code != 200 || result.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("GET %s = %d", path, result.Code)
		}
	}
	if _, ok := manager.Get("../config.yaml"); ok {
		t.Fatal("path traversal ID accepted")
	}
	if _, ok := manager.Lookup("../../config.yaml"); ok {
		t.Fatal("invalid request ID accepted")
	}
	bad := httptest.NewRecorder()
	mux.ServeHTTP(bad, httptest.NewRequest(http.MethodPost, "/api/delivery-copilot/image-jobs", strings.NewReader(string(body)+`{}`)))
	if bad.Code != 400 {
		t.Fatalf("trailing JSON accepted: %d", bad.Code)
	}
	missing := httptest.NewRecorder()
	mux.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/delivery-copilot/image-jobs/lookup?requestId=missing-request-0001", nil))
	if missing.Code != 404 || calls.Load() != 2 {
		t.Fatalf("lookup sent generation: status=%d calls=%d", missing.Code, calls.Load())
	}
}

func TestImageJobsMigrateLegacyPartialResultsWithoutGenerating(t *testing.T) {
	var calls atomic.Int32
	var providerCalls atomic.Int32
	manager, err := newImageJobManagerWithDependencies(t.TempDir(), func() (imageProvider, error) { providerCalls.Add(1); return imageJobTestProvider() }, func(_ context.Context, _ imageProvider, _ DeliveryCopilot, img GeneratedPRDImage, _ string) (GeneratedPRDImage, error) {
		calls.Add(1)
		return imageJobTestReady(img), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	copilot := imageJobTestCopilot("legacy")
	copilot.GeneratedImages = []GeneratedPRDImage{
		{ID: "prototype", Kind: "prototype", Status: "ready", URL: "https://images.example.test/previous.png"},
		{ID: "architecture", Kind: "architecture", Status: "failed", Error: "context deadline exceeded (Client.Timeout exceeded while awaiting headers)"},
	}
	job, err := manager.Start(copilot, nil, false, "migrate-legacy-0001")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "completed" || job.Images[0].Status != "ready" || job.Images[1].Status != "unknown" || providerCalls.Load() != 0 || calls.Load() != 0 {
		t.Fatalf("unsafe migration: %+v, provider=%d calls=%d", job, providerCalls.Load(), calls.Load())
	}
	if _, err := manager.Start(copilot, []string{"architecture"}, false, "legacy-retry-no-ack"); !errors.Is(err, errImageJobUnknown) {
		t.Fatalf("legacy timeout retried without confirmation: %v", err)
	}
	retry, err := manager.Start(copilot, []string{"architecture"}, true, "legacy-retry-with-ack")
	if err != nil {
		t.Fatal(err)
	}
	finished := imageJobTestWait(t, manager, retry.ID)
	if finished.Images[0].URL != "https://images.example.test/previous.png" || calls.Load() != 1 {
		t.Fatalf("ready legacy image was replaced: %+v calls=%d", finished, calls.Load())
	}
	for _, unsafe := range []string{"javascript:alert(1)", "/api/generated-images/../../config.yaml", "/api/generated-images/missing-test-image.png", "https://user:secret@example.test/image.png"} {
		if validLegacyImageURL(unsafe) {
			t.Errorf("unsafe legacy URL accepted: %s", unsafe)
		}
	}
}

func TestImageJobsAmbiguousLegacyErrorsAndMissingAssetsDoNotGenerate(t *testing.T) {
	var calls atomic.Int32
	manager, err := newImageJobManagerWithDependencies(t.TempDir(), imageJobTestProvider, func(_ context.Context, _ imageProvider, _ DeliveryCopilot, img GeneratedPRDImage, _ string) (GeneratedPRDImage, error) {
		calls.Add(1)
		return imageJobTestReady(img), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ message, status string }{
		{"图片模型返回 HTTP 502：gateway", "unknown"},
		{"图片模型返回 HTTP 408：timeout", "unknown"},
		{"图片模型返回结果不完整", "unknown"},
		{"图片模型返回 HTTP 401：Unauthorized", "failed"},
		{"图片模型返回 HTTP 429：Rate limit", "failed"},
	} {
		copilot := imageJobTestCopilot(tc.message)
		copilot.GeneratedImages = []GeneratedPRDImage{{ID: "architecture", Kind: "architecture", Status: "failed", Error: tc.message}}
		job, err := manager.Start(copilot, nil, false)
		if err != nil || job.Images[1].Status != tc.status || calls.Load() != 0 {
			t.Fatalf("legacy %q = %+v, %v calls=%d", tc.message, job, err, calls.Load())
		}
	}
	copilot := imageJobTestCopilot("lost-ledger")
	copilot.ImageJobID = "img-00000000000000000000000000000000"
	job, err := manager.Start(copilot, nil, false, "lost-ledger-new-receipt")
	if err != nil || job.Images[0].Status != "unknown" || job.Images[1].Status != "unknown" || calls.Load() != 0 {
		t.Fatalf("lost ledger generated a new image: %+v %v calls=%d", job, err, calls.Load())
	}
}

func TestImageJobsFailedAndUnknownNeverAutomaticallyRetry(t *testing.T) {
	for _, status := range []string{"failed", "unknown"} {
		t.Run(status, func(t *testing.T) {
			var architectureCalls atomic.Int32
			manager, err := newImageJobManagerWithDependencies(t.TempDir(), imageJobTestProvider, func(_ context.Context, _ imageProvider, _ DeliveryCopilot, img GeneratedPRDImage, _ string) (GeneratedPRDImage, error) {
				if img.Kind == "architecture" && architectureCalls.Add(1) == 1 {
					if status == "unknown" {
						return img, context.DeadlineExceeded
					}
					return img, &imageRequestError{status: "failed", message: "HTTP 401 private-test-image-key"}
				}
				return imageJobTestReady(img), nil
			})
			if err != nil {
				t.Fatal(err)
			}
			copilot := imageJobTestCopilot(status)
			job, err := manager.Start(copilot, nil, false, "initial-idempotency-0001")
			if err != nil {
				t.Fatal(err)
			}
			completed := imageJobTestWait(t, manager, job.ID)
			if completed.Images[1].Status != status || strings.Contains(completed.Images[1].Error, "private-test-image-key") {
				t.Fatalf("wrong failure/redaction: %+v", completed)
			}
			if _, err := manager.Start(copilot, nil, false, "second-display-request"); err != nil || architectureCalls.Load() != 1 {
				t.Fatalf("display retried failed generation: %v calls=%d", err, architectureCalls.Load())
			}
			if _, err := manager.Start(copilot, []string{"architecture"}, true, "initial-idempotency-0001"); err != nil || architectureCalls.Load() != 1 {
				t.Fatalf("same request ID retried after unknown/failure: %v calls=%d", err, architectureCalls.Load())
			}
			if status == "unknown" {
				if _, err := manager.Start(copilot, []string{"architecture"}, false, "no-acknowledgement-001"); !errors.Is(err, errImageJobUnknown) {
					t.Fatalf("unknown did not require acknowledgement: %v", err)
				}
			}
			retry, err := manager.Start(copilot, []string{"architecture"}, status == "unknown", "explicit-new-attempt-001")
			if err != nil {
				t.Fatal(err)
			}
			finished := imageJobTestWait(t, manager, retry.ID)
			if architectureCalls.Load() != 2 || finished.Images[0].URL != completed.Images[0].URL || finished.Images[1].Status != "ready" {
				t.Fatalf("retry failed to preserve ready image: %+v calls=%d", finished, architectureCalls.Load())
			}
		})
	}
}
