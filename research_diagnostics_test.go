package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// These tests inject a local runner; no ResearchEngine or provider is invoked.
func TestResearchDiagnosticStoreSurvivesReopen(t *testing.T) {
	root := t.TempDir()
	store, err := NewKnowledgeStore(root)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now().UTC().Add(-2 * time.Second).Truncate(time.Millisecond)
	if err := store.BeginResearchDiagnostic("diagnostic-persisted", start); err != nil {
		t.Fatal(err)
	}
	running, err := store.ResearchDiagnostic("diagnostic-persisted")
	if err != nil {
		t.Fatal(err)
	}
	if running.Status != "running" || !running.StartedAt.Equal(start) || !running.EndedAt.IsZero() {
		t.Fatalf("unexpected running diagnostic: %#v", running)
	}
	end := start.Add(1500 * time.Millisecond)
	want := ResearchRunDiagnostic{
		RunID: "diagnostic-persisted", Status: "failed", StartedAt: start, EndedAt: end,
		DurationMS: 1500, LastStep: "read", EventCount: 4, WarningCount: 1,
		ErrorCode: "internal_error", ErrorMessage: "Research failed.",
		Stats: Stats{Queries: 2, Discovered: 8, Read: 4, EvidenceCount: 5},
	}
	if err := store.FinishResearchDiagnostic(want); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = NewKnowledgeStore(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	got, err := store.ResearchDiagnostic(want.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RunID != want.RunID || got.Status != want.Status || !got.StartedAt.Equal(start) || !got.EndedAt.Equal(end) ||
		got.DurationMS != want.DurationMS || got.LastStep != want.LastStep || got.EventCount != want.EventCount ||
		got.WarningCount != want.WarningCount || got.ErrorCode != want.ErrorCode || got.Stats != want.Stats {
		t.Fatalf("diagnostic did not survive reopening: got %#v; want %#v", got, want)
	}
	assertNoSuccessfulResearchMemory(t, store, want.RunID)
}

func TestStreamResearchRecordsFailureWithoutSensitivePayload(t *testing.T) {
	store := newDiagnosticTestStore(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/research", nil)
	question := "private-question-marker: confidential customer acquisition plan"
	secret := "Bearer sk-private-test-marker"
	run := func(ctx context.Context, gotQuestion string, emit func(ProgressEvent)) (ResearchResult, error) {
		if gotQuestion != question {
			t.Fatalf("runner received wrong question: %q", gotQuestion)
		}
		identity := researchRunIdentityFromContext(ctx)
		startedID := diagnosticStartedRunID(t, decodeDiagnosticSSE(t, w.Body.String()))
		if identity.ID != startedID {
			t.Fatal("runner context does not share the server-issued stream identity")
		}
		running, err := store.ResearchDiagnostic(startedID)
		if err != nil || running.Status != "running" {
			t.Fatalf("runner began before durable running diagnostic: %#v err=%v", running, err)
		}
		emit(ProgressEvent{Kind: "read", Step: "read", Message: "sensitive-message-marker", Discovered: 4, ReadCount: 2, EvidenceCount: 3, At: time.Now()})
		return ResearchResult{Stats: Stats{Discovered: 4, Read: 2, EvidenceCount: 3}}, errors.New("upstream body contained " + secret)
	}
	streamResearchWithRunner(w, r, run, store, question)
	events := decodeDiagnosticSSE(t, w.Body.String())
	runID := diagnosticStartedRunID(t, events)
	failure := diagnosticOnlyTerminal(t, events, "failure")
	if failure["runId"] != runID || failure["status"] != "failed" {
		t.Fatalf("failure lost server run identity or status: %#v", failure)
	}
	got, err := store.ResearchDiagnostic(runID)
	if err != nil {
		t.Fatal(err)
	}
	assertDiagnosticEnded(t, got, "failed")
	if got.ErrorCode == "" || got.ErrorMessage == "" || got.LastStep != "read" || got.EventCount < 2 {
		t.Fatalf("failure diagnostic lacks operational context: %#v", got)
	}
	if got.Stats.Discovered != 4 || got.Stats.Read != 2 || got.Stats.EvidenceCount != 3 {
		t.Fatalf("failure diagnostic lost collected counters: %#v", got.Stats)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, sensitive := range []string{secret, question, "sensitive-message-marker", "upstream body contained"} {
		if strings.Contains(string(encoded), sensitive) {
			t.Fatalf("diagnostic stored private payload %q", sensitive)
		}
	}
	if strings.Contains(w.Body.String(), secret) {
		t.Fatal("failure event leaked original upstream credentials")
	}
	assertNoSuccessfulResearchMemory(t, store, runID)
}

func TestResearchDiagnosticStoreSanitizesFieldsAndKeepsTerminalState(t *testing.T) {
	store := newDiagnosticTestStore(t)
	start := time.Now().UTC().Add(-time.Second)
	if err := store.BeginResearchDiagnostic("diagnostic-private", start); err != nil {
		t.Fatal(err)
	}
	record := ResearchRunDiagnostic{
		RunID: "diagnostic-private", Status: "failed", StartedAt: start, EndedAt: time.Now().UTC(),
		ErrorCode: "internal_error", ErrorMessage: "Bearer sk-private-test-marker",
		LastStep: "private customer title", Warnings: []string{"unexpected_eof", "secret raw provider message"},
	}
	if err := store.FinishResearchDiagnostic(record); err != nil {
		t.Fatal(err)
	}
	got, err := store.ResearchDiagnostic(record.RunID)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"sk-private-test-marker", "private customer title", "secret raw provider message"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("diagnostic persistence accepted untrusted text: %q", secret)
		}
	}
	if got.ErrorCode != "internal_error" || got.ErrorMessage == "" || len(got.Warnings) != 1 || got.Warnings[0] != "unexpected_eof" {
		t.Fatalf("sanitization also removed useful safe diagnostic categories: %#v", got)
	}
	// A delayed or duplicate completion must never turn an already failed run
	// into a successful one.
	record.Status, record.ErrorCode = "completed", ""
	if err := store.FinishResearchDiagnostic(record); err != nil {
		t.Fatal(err)
	}
	after, err := store.ResearchDiagnostic(record.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != "failed" || after.ErrorCode != got.ErrorCode {
		t.Fatalf("later completion overwrote terminal diagnostic: %#v", after)
	}
}

func TestResearchDiagnosticEndpointIsReadOnlyAndAddressableByRunID(t *testing.T) {
	store := newDiagnosticTestStore(t)
	start := time.Now().UTC().Add(-time.Second)
	if err := store.BeginResearchDiagnostic("diagnostic-http", start); err != nil {
		t.Fatal(err)
	}
	if err := store.FinishResearchDiagnostic(ResearchRunDiagnostic{RunID: "diagnostic-http", Status: "cancelled", StartedAt: start, EndedAt: time.Now().UTC(), ErrorCode: "cancelled"}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerResearchDiagnosticRoutes(mux, store)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/research/runs/diagnostic-http", nil))
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("diagnostic endpoint did not return a private read: status=%d header=%v", w.Code, w.Header())
	}
	var got ResearchRunDiagnostic
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got.RunID != "diagnostic-http" || got.Status != "cancelled" {
		t.Fatalf("diagnostic was not addressable by run ID: got=%#v err=%v", got, err)
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/api/research/runs/missing-run", http.StatusNotFound},
		{http.MethodGet, "/api/research/runs/invalid%20identifier", http.StatusNotFound},
		{http.MethodPost, "/api/research/runs/diagnostic-http", http.StatusMethodNotAllowed},
	} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.status {
			t.Fatalf("%s %s status=%d want=%d", tc.method, tc.path, w.Code, tc.status)
		}
	}
}

func TestStreamResearchCancellationAndDeadlineAreTerminal(t *testing.T) {
	for _, tc := range []struct {
		name, status, code string
		cause              error
	}{
		{"cancelled", "cancelled", "cancelled", context.Canceled},
		{"deadline", "failed", "deadline_exceeded", context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newDiagnosticTestStore(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/research", nil).WithContext(ctx)
			run := func(ctx context.Context, _ string, emit func(ProgressEvent)) (ResearchResult, error) {
				emit(ProgressEvent{Kind: "search", Step: "search", At: time.Now()})
				if tc.cause == context.Canceled {
					cancel()
					if !errors.Is(ctx.Err(), context.Canceled) {
						t.Fatal("runner did not inherit request cancellation")
					}
				}
				return ResearchResult{}, tc.cause
			}
			streamResearchWithRunner(w, r, run, store, "offline diagnostic cancellation test")
			events := decodeDiagnosticSSE(t, w.Body.String())
			runID := diagnosticStartedRunID(t, events)
			failure := diagnosticOnlyTerminal(t, events, "failure")
			if failure["runId"] != runID || failure["status"] != tc.status {
				t.Fatalf("incorrect terminal status: %#v", failure)
			}
			got, err := store.ResearchDiagnostic(runID)
			if err != nil {
				t.Fatal(err)
			}
			assertDiagnosticEnded(t, got, tc.status)
			if got.ErrorCode != tc.code {
				t.Fatalf("diagnostic code = %q, want %q", got.ErrorCode, tc.code)
			}
			assertNoSuccessfulResearchMemory(t, store, runID)
		})
	}
}

func TestStreamResearchWriteFailureCancelsRunnerAndRecordsDiagnostic(t *testing.T) {
	store := newDiagnosticTestStore(t)
	w := &diagnosticBrokenWriter{header: make(http.Header), failAt: 2}
	r := httptest.NewRequest(http.MethodGet, "/api/research", nil)
	runnerWasCancelled := false
	run := func(ctx context.Context, _ string, emit func(ProgressEvent)) (ResearchResult, error) {
		emit(ProgressEvent{Kind: "read", Step: "read", At: time.Now()})
		runnerWasCancelled = errors.Is(ctx.Err(), context.Canceled)
		return ResearchResult{}, ctx.Err()
	}
	streamResearchWithRunner(w, r, run, store, "offline diagnostic disconnected stream test")
	if !runnerWasCancelled {
		t.Fatal("failed SSE write did not immediately cancel the research runner")
	}
	events := decodeDiagnosticSSE(t, w.body.String())
	runID := diagnosticStartedRunID(t, events)
	got, err := store.ResearchDiagnostic(runID)
	if err != nil {
		t.Fatal(err)
	}
	assertDiagnosticEnded(t, got, "cancelled")
	if got.ErrorCode != "stream_disconnected" {
		t.Fatalf("disconnect reason was lost: %#v", got)
	}
	for _, event := range events {
		if event.kind == "complete" {
			t.Fatal("disconnected stream emitted a successful completion")
		}
	}
	assertNoSuccessfulResearchMemory(t, store, runID)
}

func TestStreamResearchSuccessfulRunHasOneCompleteAndPreservesMemory(t *testing.T) {
	for _, warning := range []bool{false, true} {
		name := "completed"
		if warning {
			name = "completed_with_warnings"
		}
		t.Run(name, func(t *testing.T) {
			// RecordResearchRun reads agent.yaml; isolate even that local policy input.
			t.Chdir(t.TempDir())
			store := newDiagnosticTestStore(t)
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/research", nil)
			run := func(_ context.Context, question string, emit func(ProgressEvent)) (ResearchResult, error) {
				if warning {
					emit(ProgressEvent{Kind: "warning", Step: "synthesize", Message: "offline warning", At: time.Now()})
				}
				return ResearchResult{
					RunID: "runner-controlled-id-must-not-replace-server-id", Question: question,
					StartedAt: time.Now().Add(-time.Second), CompletedAt: time.Now(),
					Stats:  Stats{Queries: 1, Discovered: 2, Read: 1, EvidenceCount: 2},
					Report: Report{Title: "Offline completed report", Analysis: "Bounded fixture result."},
				}, nil
			}
			streamResearchWithRunner(w, r, run, store, "offline diagnostic successful research test")
			events := decodeDiagnosticSSE(t, w.Body.String())
			runID := diagnosticStartedRunID(t, events)
			complete := diagnosticOnlyTerminal(t, events, "complete")
			if complete["runId"] != runID || runID == "runner-controlled-id-must-not-replace-server-id" {
				t.Fatalf("complete event did not preserve the server run ID: %#v", complete)
			}
			got, err := store.ResearchDiagnostic(runID)
			if err != nil {
				t.Fatal(err)
			}
			assertDiagnosticEnded(t, got, name)
			if got.ErrorCode != "" || got.Stats.EvidenceCount != 2 || (warning && got.WarningCount < 1) {
				t.Fatalf("unexpected successful diagnostic: %#v", got)
			}
			var count int
			if err := store.db.QueryRow(`SELECT COUNT(*) FROM agent_runs WHERE id=? AND status=?`, runID, name).Scan(&count); err != nil || count != 1 {
				t.Fatalf("successful run was not recorded exactly once: count=%d err=%v", count, err)
			}
			if err := store.db.QueryRow(`SELECT COUNT(*) FROM agent_events WHERE run_id=?`, runID).Scan(&count); err != nil || count == 0 {
				t.Fatalf("successful run lost its operational memory: count=%d err=%v", count, err)
			}
		})
	}
}

func TestStreamResearchCompleteDeliveryFailureDoesNotCreateSuccessfulMemory(t *testing.T) {
	for _, failDuring := range []string{"write", "flush"} {
		t.Run(failDuring, func(t *testing.T) {
			store := newDiagnosticTestStore(t)
			w := &diagnosticCompleteFailureWriter{header: make(http.Header), failDuring: failDuring}
			r := httptest.NewRequest(http.MethodGet, "/api/research", nil)
			run := func(_ context.Context, question string, _ func(ProgressEvent)) (ResearchResult, error) {
				return ResearchResult{Question: question, CompletedAt: time.Now(), Report: Report{Title: "Offline report", Analysis: "Complete locally; delivery must still be checked."}}, nil
			}
			streamResearchWithRunner(w, r, run, store, "offline complete delivery failure test")
			if !w.failed {
				t.Fatalf("test never reached complete %s", failDuring)
			}
			runID := diagnosticStartedRunID(t, decodeDiagnosticSSE(t, w.body.String()))
			got, err := store.ResearchDiagnostic(runID)
			if err != nil {
				t.Fatal(err)
			}
			assertDiagnosticEnded(t, got, "cancelled")
			if got.ErrorCode != "stream_disconnected" {
				t.Fatalf("complete delivery failure lost its cause: %#v", got)
			}
			assertNoSuccessfulResearchMemory(t, store, runID)
		})
	}
}

func newDiagnosticTestStore(t *testing.T) *KnowledgeStore {
	t.Helper()
	store, err := NewKnowledgeStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func assertNoSuccessfulResearchMemory(t *testing.T, store *KnowledgeStore, runID string) {
	t.Helper()
	for _, query := range []string{
		`SELECT COUNT(*) FROM agent_runs WHERE id=? AND status IN ('completed','completed_with_warnings')`,
		`SELECT COUNT(*) FROM agent_events WHERE run_id=?`,
	} {
		var count int
		if err := store.db.QueryRow(query, runID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("non-successful diagnostic created successful memory: count=%d query=%s", count, query)
		}
	}
}

func assertDiagnosticEnded(t *testing.T, got ResearchRunDiagnostic, status string) {
	t.Helper()
	if got.RunID == "" || got.Status != status || got.StartedAt.IsZero() || got.EndedAt.IsZero() ||
		got.EndedAt.Before(got.StartedAt) || got.DurationMS < 0 {
		t.Fatalf("invalid terminal diagnostic (want %s): %#v", status, got)
	}
}

type diagnosticSSEEvent struct {
	kind    string
	payload map[string]any
}

func decodeDiagnosticSSE(t *testing.T, body string) []diagnosticSSEEvent {
	t.Helper()
	var events []diagnosticSSEEvent
	for _, frame := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n\n") {
		var kind string
		var data []string
		for _, line := range strings.Split(frame, "\n") {
			if strings.HasPrefix(line, "event:") {
				kind = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			}
			if strings.HasPrefix(line, "data:") {
				data = append(data, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		if kind == "" || len(data) == 0 {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(strings.Join(data, "\n")), &payload); err != nil {
			t.Fatalf("invalid SSE JSON: %v; frame=%s", err, frame)
		}
		events = append(events, diagnosticSSEEvent{kind: kind, payload: payload})
	}
	return events
}

func diagnosticStartedRunID(t *testing.T, events []diagnosticSSEEvent) string {
	t.Helper()
	for _, event := range events {
		if event.kind == "progress" && event.payload["kind"] == "started" {
			if runID, ok := event.payload["runId"].(string); ok && runID != "" {
				return runID
			}
		}
	}
	t.Fatalf("started event did not contain a runId: %#v", events)
	return ""
}

func diagnosticOnlyTerminal(t *testing.T, events []diagnosticSSEEvent, kind string) map[string]any {
	t.Helper()
	var terminal map[string]any
	count := 0
	for _, event := range events {
		if event.kind != "complete" && event.kind != "failure" {
			continue
		}
		count++
		if event.kind != kind {
			t.Fatalf("unexpected terminal event %s; wanted %s", event.kind, kind)
		}
		terminal = event.payload
	}
	if count != 1 {
		t.Fatalf("expected exactly one %s event, got %d terminal events", kind, count)
	}
	return terminal
}

type diagnosticBrokenWriter struct {
	header http.Header
	body   bytes.Buffer
	writes int
	failAt int
}

func (w *diagnosticBrokenWriter) Header() http.Header { return w.header }
func (w *diagnosticBrokenWriter) WriteHeader(int)     {}
func (w *diagnosticBrokenWriter) Flush()              {}
func (w *diagnosticBrokenWriter) Write(body []byte) (int, error) {
	w.writes++
	if w.writes >= w.failAt {
		return 0, io.ErrClosedPipe
	}
	return w.body.Write(body)
}

type diagnosticCompleteFailureWriter struct {
	header       http.Header
	body         bytes.Buffer
	failDuring   string
	lastComplete bool
	failed       bool
}

func (w *diagnosticCompleteFailureWriter) Header() http.Header { return w.header }
func (w *diagnosticCompleteFailureWriter) WriteHeader(int)     {}
func (w *diagnosticCompleteFailureWriter) Flush()              { _ = w.FlushError() }
func (w *diagnosticCompleteFailureWriter) FlushError() error {
	if w.lastComplete && w.failDuring == "flush" {
		w.failed = true
		return io.ErrClosedPipe
	}
	return nil
}
func (w *diagnosticCompleteFailureWriter) Write(body []byte) (int, error) {
	w.lastComplete = bytes.HasPrefix(body, []byte("event: complete\n"))
	if w.lastComplete && w.failDuring == "write" {
		w.failed = true
		return 0, io.ErrClosedPipe
	}
	return w.body.Write(body)
}
