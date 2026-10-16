package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func historyTestResult(id, question string, minute int) ResearchResult {
	start := time.Date(2026, 10, 5, 8, minute, 0, 0, time.UTC)
	return ResearchResult{
		RunID: id, Question: question, Status: "completed_with_warnings", StartedAt: start, CompletedAt: start.Add(time.Minute),
		Plan:     []PlanStep{{ID: "read", Title: "阅读官方文档", Status: "done"}},
		Stats:    Stats{Queries: 4, Discovered: 12, Read: 8, EvidenceCount: 2},
		Report:   Report{Title: "研究：" + question, Analysis: "## 原始结论\n任务 " + id + " 的完整正文，有明确证据 [E1]。\n限制与反证：样本不足。", Method: "多轮检索", ModelUsed: "offline-test", Limitations: []string{"需继续验证"}, Conflicts: []string{"样本口径不同"}},
		Sources:  []Source{{ID: "S1", Title: "官方参考", URL: "https://example.com/reference", Domain: "example.com", Kind: "official", FetchedAt: start, Trust: 92, Status: "已阅读"}},
		Evidence: []Evidence{{ID: "E1", SourceID: "S1", SourceTitle: "官方参考", URL: "https://example.com/reference", Quote: "可追溯证据原文", Score: 92, Confidence: "high", Tags: []string{"市场"}, FetchedAt: start}},
	}
}

func historyTestStore(t *testing.T) *KnowledgeStore {
	t.Helper()
	store, err := NewKnowledgeStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func historyTestSave(t *testing.T, store *KnowledgeStore, result ResearchResult) {
	t.Helper()
	if err := store.SaveResearchArchive(result); err != nil {
		t.Fatal(err)
	}
}

func historyTestLegacy(t *testing.T, store *KnowledgeStore, result ResearchResult) {
	t.Helper()
	_, err := store.db.Exec(`INSERT INTO agent_runs(id,question,status,result_summary,started_at,completed_at,created_at) VALUES(?,?,?,?,?,?,?)`, result.RunID, result.Question, "completed", "旧摘要：只有标题和方法，没有报告正文", result.StartedAt.Format(time.RFC3339Nano), result.CompletedAt.Format(time.RFC3339Nano), result.CompletedAt.Format(time.RFC3339Nano))
	if err != nil {
		t.Fatal(err)
	}
}

func historyTestDelivery(result ResearchResult, goal string) DeliveryCopilot {
	return DeliveryCopilot{ResearchRunID: result.RunID, PRD: EnterprisePRD{Info: PRDDocumentInfo{PRDID: researchHistoryPRDID(result.RunID, result.Question), Feature: result.Question, Product: "历史测试产品", Version: "0.1"}, Goals: []string{goal}}}
}

func TestResearchHistoryExactSnapshotSurvivesReopen(t *testing.T) {
	root := t.TempDir()
	store, err := NewKnowledgeStore(root)
	if err != nil {
		t.Fatal(err)
	}
	want := historyTestResult("history-persisted", "自媒体工具市场", 1)
	historyTestSave(t, store, want)
	delivery := historyTestDelivery(want, "用户特定目标")
	if err := store.SaveResearchDelivery(want.RunID, delivery); err != nil {
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
	got, err := store.GetResearchArchive(want.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Result == nil || !reflect.DeepEqual(*got.Result, want) {
		t.Fatalf("full report/evidence snapshot changed: got %#v", got.Result)
	}
	if got.Delivery == nil || !reflect.DeepEqual(*got.Delivery, delivery) {
		t.Fatalf("delivery snapshot changed: %#v", got.Delivery)
	}
	if !got.Item.HasSnapshot || !got.Item.HasDelivery || got.Item.Status != "completed_with_warnings" || got.Item.RunID != want.RunID {
		t.Fatalf("wrong metadata: %#v", got.Item)
	}
	page, err := store.ListResearchHistory("自媒体", 20, 0)
	if err != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("reopened history: %#v, err=%v", page, err)
	}
}

func TestResearchHistoryKeywordPagingAndLegacyDedup(t *testing.T) {
	store := historyTestStore(t)
	questions := []string{"增长率 10% 的市场", "产品_工作台", "中文市场竞品", "中文市场需求"}
	for i, question := range questions {
		result := historyTestResult("history-page-"+string(rune('a'+i)), question, i)
		historyTestSave(t, store, result)
		if i == 0 {
			historyTestLegacy(t, store, result)
		}
	}
	historyTestLegacy(t, store, historyTestResult("history-old", "旧的中文市场问题", 10))
	for _, tc := range []struct {
		query string
		count int
	}{{"", 5}, {"%", 1}, {"_", 1}, {"中文市场", 3}, {"' OR 1=1 --", 0}, {`\`, 0}} {
		page, err := store.ListResearchHistory(tc.query, 100, 0)
		if err != nil || page.Total != tc.count || len(page.Items) != tc.count {
			t.Errorf("query %q: %#v err=%v", tc.query, page, err)
		}
	}
	first, err := store.ListResearchHistory("中文市场", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.ListResearchHistory("中文市场", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Total != 3 || first.Items[0].RunID != "history-old" || second.Items[0].RunID == first.Items[0].RunID || second.Offset != 1 || second.Limit != 1 {
		t.Fatalf("unstable paging: %#v %#v", first, second)
	}
	empty, err := store.ListResearchHistory("中文市场", 1, 99)
	if err != nil || empty.Items == nil || len(empty.Items) != 0 || empty.Total != 3 {
		t.Fatalf("empty page: %#v err=%v", empty, err)
	}
	bounded, err := store.ListResearchHistory("", 999, -1)
	if err != nil || bounded.Limit != 100 || bounded.Offset != 0 {
		t.Fatalf("paging bounds: %#v %v", bounded, err)
	}
}

func TestResearchHistoryLegacyIsExplicitAndSavedPRDMatchesExactly(t *testing.T) {
	store := historyTestStore(t)
	a := historyTestResult("history-legacy-a", "完全一样的问题", 1)
	b := historyTestResult("history-legacy-b", "完全一样的问题", 2)
	historyTestLegacy(t, store, a)
	historyTestLegacy(t, store, b)
	saved := historyTestDelivery(a, "A 的人工编辑目标")
	saved.ResearchRunID = "" // Legacy snapshots predate the new association field.
	if _, err := store.PersistDeliveryCopilot(saved); err != nil {
		t.Fatal(err)
	}
	detail, err := store.GetResearchArchive(a.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Result != nil || detail.Item.HasSnapshot || detail.Notice == "" || detail.Delivery == nil || !detail.Item.HasDelivery {
		t.Fatalf("legacy report was fabricated or saved PRD lost: %#v", detail)
	}
	if detail.Delivery.PRD.Goals[0] != "A 的人工编辑目标" || detail.Delivery.ResearchRunID != a.RunID {
		t.Fatal("wrong restored PRD")
	}
	other, err := store.GetResearchArchive(b.RunID)
	if err != nil || other.Delivery != nil || other.Item.HasDelivery {
		t.Fatalf("same-question runs were falsely associated: %#v err=%v", other, err)
	}
	page, err := store.ListResearchHistory("", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range page.Items {
		if item.HasDelivery != (item.RunID == a.RunID) {
			t.Fatalf("wrong legacy hasDelivery: %#v", item)
		}
	}
}

func TestResearchHistoryLatestSavedPRDWinsAndRunsDoNotOverwrite(t *testing.T) {
	store := historyTestStore(t)
	a := historyTestResult("history-independent-a", "相同问题", 1)
	b := historyTestResult("history-independent-b", "相同问题", 2)
	historyTestSave(t, store, a)
	historyTestSave(t, store, b)
	initial := historyTestDelivery(a, "最初生成的目标")
	if err := store.SaveResearchDelivery(a.RunID, initial); err != nil {
		t.Fatal(err)
	}
	edited := historyTestDelivery(a, "用户已编辑的目标")
	if _, err := store.PersistDeliveryCopilot(edited); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PersistDeliveryCopilot(historyTestDelivery(b, "另一个研究的目标")); err != nil {
		t.Fatal(err)
	}
	// Re-saving the report is idempotent and must not clear its delivery snapshot.
	historyTestSave(t, store, a)
	got, err := store.GetResearchArchive(a.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Result.Report.Analysis != a.Report.Analysis || got.Delivery == nil || got.Delivery.PRD.Goals[0] != "用户已编辑的目标" {
		t.Fatalf("manual edit replaced with old or global draft: %#v", got)
	}
	other, err := store.GetResearchArchive(b.RunID)
	if err != nil || other.Result.Report.Analysis != b.Report.Analysis || other.Delivery.PRD.Goals[0] != "另一个研究的目标" {
		t.Fatalf("other run overwritten: %#v %v", other, err)
	}
	if err := store.SaveResearchDelivery(b.RunID, initial); err == nil {
		t.Fatal("allowed cross-run PRD attachment")
	}
	initial.ResearchRunID = b.RunID
	if err := store.SaveResearchDelivery(a.RunID, initial); err == nil {
		t.Fatal("ignored explicit mismatching researchRunId")
	}
	wrongQuestion := a
	wrongQuestion.Question = "其他问题"
	if err := store.SaveResearchArchive(wrongQuestion); err == nil {
		t.Fatal("allowed run ID to overwrite another question")
	}
	if err := store.SaveResearchDelivery("missing", edited); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing association err=%v", err)
	}
}

func TestResearchHistoryRejectsIncompleteAndMismatchedStoredPRD(t *testing.T) {
	store := historyTestStore(t)
	valid := historyTestResult("history-valid", "合法问题", 1)
	for _, change := range []func(*ResearchResult){func(r *ResearchResult) { r.Report.Analysis = " " }, func(r *ResearchResult) { r.Question = "" }, func(r *ResearchResult) { r.RunID = "../bad" }, func(r *ResearchResult) { r.Status = "cancelled" }} {
		invalid := valid
		change(&invalid)
		if err := store.SaveResearchArchive(invalid); err == nil {
			t.Fatalf("invalid report accepted: %#v", invalid)
		}
	}
	historyTestSave(t, store, valid)
	copilot := historyTestDelivery(valid, "正确的归档 PRD")
	if err := store.SaveResearchDelivery(valid.RunID, copilot); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PersistDeliveryCopilot(copilot); err != nil {
		t.Fatal(err)
	}
	wrong := historyTestDelivery(historyTestResult("other-run", "另一个问题", 2), "不应出现")
	payload, _ := json.Marshal(wrong)
	if _, err := store.db.Exec(`UPDATE prd_versions SET snapshot_json=?`, string(payload)); err != nil {
		t.Fatal(err)
	}
	detail, err := store.GetResearchArchive(valid.RunID)
	if err != nil || detail.Delivery != nil || detail.Result == nil || detail.Notice == "" {
		t.Fatalf("mismatched saved snapshot leaked or hidden by stale fallback: %#v %v", detail, err)
	}
}

func TestResearchHistoryCorruptSavedPRDDoesNotHideReport(t *testing.T) {
	store := historyTestStore(t)
	result := historyTestResult("history-damaged-prd", "可恢复的报告", 1)
	historyTestSave(t, store, result)
	copilot := historyTestDelivery(result, "陈旧的自动生成目标")
	if err := store.SaveResearchDelivery(result.RunID, copilot); err != nil {
		t.Fatal(err)
	}
	copilot.PRD.Goals = []string{"需要保护的人工修改"}
	if _, err := store.PersistDeliveryCopilot(copilot); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE prd_versions SET snapshot_json='{'`); err != nil {
		t.Fatal(err)
	}
	detail, err := store.GetResearchArchive(result.RunID)
	if err != nil || detail.Result == nil || detail.Result.Report.Analysis != result.Report.Analysis || detail.Delivery != nil || detail.Item.HasDelivery || !strings.Contains(detail.Notice, "未自动覆盖或重新生成") {
		t.Fatalf("damaged saved PRD hid report or restored stale generation: %#v err=%v", detail, err)
	}
	page, err := store.ListResearchHistory("", 20, 0)
	if err != nil || page.Total != 1 || !page.Items[0].HasSnapshot || page.Items[0].HasDelivery {
		t.Fatalf("damaged PRD hid history list: %#v err=%v", page, err)
	}
	var raw string
	if err := store.db.QueryRow(`SELECT snapshot_json FROM prd_versions LIMIT 1`).Scan(&raw); err != nil || raw != "{" {
		t.Fatal("GET rewrote damaged manual data")
	}
}

func TestResearchHistoryRoutesAreReadOnlyAndDoNotRegenerate(t *testing.T) {
	store := historyTestStore(t)
	store.db.SetMaxOpenConns(1)
	result := historyTestResult("history-route", "无需再次研究的问题", 1)
	historyTestSave(t, store, result)
	historyTestLegacy(t, store, historyTestResult("history-route-old", "旧问题", 2))
	if _, err := store.db.Exec(`PRAGMA query_only=ON`); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerResearchHistoryRoutes(mux, store)
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/api/research/history", 200},
		{http.MethodGet, "/api/research/history?q=%E9%97%AE%E9%A2%98&limit=1&offset=0", 200},
		{http.MethodGet, "/api/research/history/history-route", 200},
		{http.MethodGet, "/api/research/history/history-route-old", 200},
		{http.MethodGet, "/api/research/history/no-such-run", 404},
		{http.MethodGet, "/api/research/history/", 404},
		{http.MethodPost, "/api/research/history", 405},
		{http.MethodDelete, "/api/research/history/history-route", 405},
		{http.MethodGet, "/api/research/history?limit=invalid", 400},
		{http.MethodGet, "/api/research/history?limit=0", 400},
		{http.MethodGet, "/api/research/history?offset=-2", 400},
	} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.status || w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s %s: %d %s", tc.method, tc.path, w.Code, w.Body.String())
		}
		if tc.status == 405 && w.Header().Get("Allow") != http.MethodGet {
			t.Error("missing Allow header")
		}
		if tc.path == "/api/research/history/history-route-old" {
			var payload map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload["result"] != nil || payload["delivery"] != nil || payload["notice"] == "" {
				t.Fatalf("legacy route concealed missing snapshot: %s", w.Body.String())
			}
		}
	}
}

func TestResearchHistoryRouteDoesNotExposeDatabaseErrors(t *testing.T) {
	store := historyTestStore(t)
	mux := http.NewServeMux()
	registerResearchHistoryRoutes(mux, store)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/research/history", "/api/research/history/history-route"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 500 || strings.Contains(w.Body.String(), "database is closed") || strings.Contains(w.Body.String(), store.root) {
			t.Fatalf("database internals exposed: %d %s", w.Code, w.Body.String())
		}
	}
}
