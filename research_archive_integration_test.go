package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func archiveIntegrationRunner(_ context.Context, question string, _ func(ProgressEvent)) (ResearchResult, error) {
	return ResearchResult{
		Question: question, CompletedAt: time.Now().UTC(),
		Report:   Report{Title: "离线研究存档", Analysis: "原始报告正文与证据 [E1]，不重新合成。", ModelUsed: "offline-test"},
		Stats:    Stats{Queries: 2, Read: 1, EvidenceCount: 1},
		Sources:  []Source{{ID: "S1", Title: "原始来源", URL: "https://example.com/original", Status: "已阅读"}},
		Evidence: []Evidence{{ID: "E1", SourceID: "S1", Quote: "当时提取的原始证据", URL: "https://example.com/original"}},
	}, nil
}

func TestResearchStreamArchivesFullResultBeforeBrowserCompletion(t *testing.T) {
	t.Chdir(t.TempDir())
	store := newDiagnosticTestStore(t)
	w := httptest.NewRecorder()
	streamResearchWithRunner(w, httptest.NewRequest(http.MethodPost, "/api/research", nil), archiveIntegrationRunner, store, "保存这个历史课题，不重复消耗模型额度")
	events := decodeDiagnosticSSE(t, w.Body.String())
	runID := diagnosticStartedRunID(t, events)
	complete := diagnosticOnlyTerminal(t, events, "complete")
	if complete["archiveStatus"] != "saved" {
		t.Fatalf("archive status = %v", complete["archiveStatus"])
	}
	mux := http.NewServeMux()
	registerResearchHistoryRoutes(mux, store)
	for i := 0; i < 3; i++ {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/research/history/"+runID, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("open history: %d %s", response.Code, response.Body.String())
		}
		var detail struct {
			Result *ResearchResult `json:"result"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil {
			t.Fatal(err)
		}
		if detail.Result == nil || detail.Result.Report.Analysis != "原始报告正文与证据 [E1]，不重新合成。" || len(detail.Result.Evidence) != 1 || detail.Result.Evidence[0].Quote != "当时提取的原始证据" || detail.Result.Sources[0].URL != "https://example.com/original" {
			t.Fatalf("history did not return exact report/evidence/source: %#v", detail.Result)
		}
	}
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM research_archives`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("reopening created another run: count=%d err=%v", count, err)
	}
}

func TestResearchArchiveSurvivesTerminalFrameDisconnect(t *testing.T) {
	store := newDiagnosticTestStore(t)
	w := &diagnosticCompleteFailureWriter{header: make(http.Header), failDuring: "write"}
	streamResearchWithRunner(w, httptest.NewRequest(http.MethodPost, "/api/research", nil), archiveIntegrationRunner, store, "报告生成后浏览器掉线仍能查找历史")
	runID := diagnosticStartedRunID(t, decodeDiagnosticSSE(t, w.body.String()))
	if !w.failed {
		t.Fatal("terminal write was not exercised")
	}
	detail, err := store.GetResearchArchive(runID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Result == nil || !strings.Contains(detail.Result.Report.Analysis, "原始报告正文") {
		t.Fatal("server-produced report was discarded on delivery failure")
	}
	diagnostic, err := store.ResearchDiagnostic(runID)
	if err != nil || diagnostic.Status != "cancelled" || diagnostic.ErrorCode != "stream_disconnected" {
		t.Fatalf("delivery failure lost its separate diagnostic: %#v %v", diagnostic, err)
	}
	assertNoSuccessfulResearchMemory(t, store, runID)
}

func TestResearchArchiveFailureIsVisibleWithoutDiscardingReport(t *testing.T) {
	t.Chdir(t.TempDir())
	store := newDiagnosticTestStore(t)
	if _, err := store.db.Exec(`DROP TABLE research_archives`); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	streamResearchWithRunner(w, httptest.NewRequest(http.MethodPost, "/api/research", nil), archiveIntegrationRunner, store, "存储故障需要提示用户手动下载报告")
	events := decodeDiagnosticSSE(t, w.Body.String())
	complete := diagnosticOnlyTerminal(t, events, "complete")
	if complete["archiveStatus"] != "failed" || complete["status"] != "completed_with_warnings" || !strings.Contains(w.Body.String(), "历史归档保存失败") {
		t.Fatalf("archive failure was silent: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "原始报告正文") {
		t.Fatal("usable report was lost")
	}
	diagnostic, err := store.ResearchDiagnostic(diagnosticStartedRunID(t, events))
	if err != nil || !strings.Contains(strings.Join(diagnostic.Warnings, ","), "history_save_failed") {
		t.Fatalf("missing save diagnostic: %#v %v", diagnostic, err)
	}
}
