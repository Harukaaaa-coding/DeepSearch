package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestCheckedResponseOutputTextRejectsIncompleteOrInvalidResults(t *testing.T) {
	tests := []struct {
		name string
		body string
		want error
	}{
		{"original truncated report", `{"choices":[{"finish_reason":"length","message":{"content":"## 研究结论\n第一条发现是，接下来我们"}}]}`, errIncompleteModelOutput},
		{"chat filtered", `{"choices":[{"finish_reason":"content_filter","message":{"content":"partial"}}]}`, errIncompleteModelOutput},
		{"chat tools pending", `{"choices":[{"finish_reason":"tool_calls","message":{"content":"partial"}}]}`, errIncompleteModelOutput},
		{"chat function pending", `{"choices":[{"finish_reason":"function_call","message":{"content":"partial"}}]}`, errIncompleteModelOutput},
		{"chat finish null", `{"choices":[{"finish_reason":null,"message":{"content":"partial"}}]}`, errIncompleteModelOutput},
		{"chat finish empty", `{"choices":[{"finish_reason":"","message":{"content":"partial"}}]}`, errIncompleteModelOutput},
		{"chat finish unknown", `{"choices":[{"finish_reason":"unexpected-secret","message":{"content":"partial"}}]}`, errIncompleteModelOutput},
		{"chat tool payload despite stop", `{"choices":[{"finish_reason":"stop","message":{"content":"partial","tool_calls":[{"type":"function"}]}}]}`, errIncompleteModelOutput},
		{"chat function payload despite stop", `{"choices":[{"finish_reason":"stop","message":{"content":"partial","function_call":{"name":"lookup"}}}]}`, errIncompleteModelOutput},
		{"chat refusal with text", `{"choices":[{"finish_reason":"stop","message":{"content":"partial","refusal":"private refusal"}}]}`, errInvalidModelOutput},
		{"chat content refusal", `{"choices":[{"finish_reason":"stop","message":{"content":[{"type":"text","text":"partial"},{"type":"refusal","refusal":"private refusal"}]}}]}`, errInvalidModelOutput},
		{"responses incomplete shortcut", `{"status":"incomplete","output_text":"partial"}`, errIncompleteModelOutput},
		{"responses max tokens", `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output_text":"partial"}`, errIncompleteModelOutput},
		{"responses details overrides completed", `{"status":"completed","incomplete_details":{"reason":"content_filter"},"output_text":"partial"}`, errIncompleteModelOutput},
		{"responses failed", `{"status":"failed","output_text":"partial"}`, errInvalidModelOutput},
		{"responses canceled", `{"status":"cancelled","output_text":"partial"}`, errIncompleteModelOutput},
		{"responses queued", `{"status":"queued","output_text":"partial"}`, errIncompleteModelOutput},
		{"responses in progress", `{"status":"in_progress","output_text":"partial"}`, errIncompleteModelOutput},
		{"responses null status", `{"status":null,"output_text":"partial"}`, errIncompleteModelOutput},
		{"responses error hides shortcut", `{"status":"completed","error":{"message":"api-key-private-secret","code":"private-code"},"output_text":"partial"}`, errInvalidModelOutput},
		{"responses incomplete message hides shortcut", `{"status":"completed","output_text":"partial","output":[{"type":"message","status":"incomplete","content":[{"type":"output_text","text":"partial"}]}]}`, errIncompleteModelOutput},
		{"responses refusal hides shortcut", `{"status":"completed","output_text":"partial","output":[{"type":"message","content":[{"type":"refusal","refusal":"private refusal"}]}]}`, errInvalidModelOutput},
		{"responses tool pending hides shortcut", `{"status":"completed","output_text":"partial","output":[{"type":"function_call","name":"lookup"}]}`, errIncompleteModelOutput},
		{"responses custom tool pending", `{"status":"completed","output_text":"partial","output":[{"type":"custom_tool_call"}]}`, errIncompleteModelOutput},
		{"responses top refusal", `{"status":"completed","output_text":"partial","refusal":"private refusal"}`, errInvalidModelOutput},
		{"invalid JSON", `{"choices":[`, errInvalidModelOutput},
		{"empty", `{}`, errInvalidModelOutput},
		{"null", `null`, errInvalidModelOutput},
		{"array", `[]`, errInvalidModelOutput},
		{"empty choice text", `{"choices":[{"finish_reason":"stop","message":{"content":"  "}}]}`, errInvalidModelOutput},
		{"empty choices cannot fallback", `{"choices":[],"output_text":"partial"}`, errInvalidModelOutput},
		{"choices null cannot fallback", `{"choices":null,"output_text":"partial"}`, errInvalidModelOutput},
		{"reasoning only", `{"status":"completed","output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"not final"}]}]}`, errInvalidModelOutput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := checkedResponseOutputText([]byte(tt.body))
			if text != "" || !errors.Is(err, tt.want) {
				t.Fatalf("text=%q err=%v; expected empty text and %v", text, err, tt.want)
			}
			if got := responseOutputText([]byte(tt.body)); got != "" {
				t.Fatalf("compatibility wrapper leaked rejected text: %q", got)
			}
			if strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "partial") {
				t.Fatalf("diagnostic exposed provider response content: %v", err)
			}
		})
	}
}

func TestCheckedResponseOutputTextAcceptsCompleteAndLegacyShapes(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"chat stop", `{"choices":[{"finish_reason":"stop","message":{"content":"完整报告","refusal":null,"tool_calls":[],"function_call":null}}]}`, "完整报告"},
		{"chat legacy missing metadata", `{"choices":[{"message":{"content":"旧兼容文本"}}]}`, "旧兼容文本"},
		{"chat typed text", `{"choices":[{"finish_reason":"stop","message":{"content":[{"type":"text","text":"第一段"},{"type":"output_text","text":"第二段"}]}}]}`, "第一段\n第二段"},
		{"responses shortcut", `{"status":"completed","output_text":"完整报告","error":null,"incomplete_details":null}`, "完整报告"},
		{"responses typed output", `{"status":"completed","output":[{"type":"reasoning"},{"type":"message","status":"completed","content":[{"type":"output_text","text":"第一段"},{"type":"output_text","text":"第二段"}]}]}`, "第一段\n第二段"},
		{"responses executed server tool", `{"status":"completed","output":[{"type":"web_search_call","status":"completed"},{"type":"message","status":"completed","content":[{"type":"output_text","text":"完整报告"}]}]}`, "完整报告"},
		{"responses legacy shortcut", `{"output_text":"旧兼容文本"}`, "旧兼容文本"},
		{"responses legacy typed output", `{"output":[{"content":[{"type":"text","text":"旧兼容文本"}]}]}`, "旧兼容文本"},
		{"response shortcut not doubled", `{"status":"completed","output_text":"完整报告","output":[{"type":"message","content":[{"type":"output_text","text":"完整报告"}]}]}`, "完整报告"},
		{"one selected choice only", `{"choices":[{"index":0,"finish_reason":"stop","message":{"content":[{"type":"text","text":"候选零"}]}},{"index":1,"finish_reason":"stop","message":{"content":[{"type":"text","text":"候选一"}]}}]}`, "候选零"},
		{"index zero stable selection", `{"choices":[{"index":1,"finish_reason":"length","message":{"content":"其他半截"}},{"index":0,"finish_reason":"stop","message":{"content":"候选零"}}]}`, "候选零"},
		{"missing indices first only", `{"choices":[{"message":{"content":[{"type":"text","text":"第一条"}]}},{"message":{"content":[{"type":"text","text":"第二条"}]}}]}`, "第一条"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := checkedResponseOutputText([]byte(tt.body))
			if err != nil || text != tt.want {
				t.Fatalf("text=%q err=%v, want %q", text, err, tt.want)
			}
		})
	}
}

func TestCheckedResponseOutputTextDoesNotSkipTruncatedPrimaryChoice(t *testing.T) {
	body := []byte(`{"choices":[{"index":0,"finish_reason":"length","message":{"content":"第一条半截"}},{"index":1,"finish_reason":"stop","message":{"content":"另一候选"}}]}`)
	text, err := checkedResponseOutputText(body)
	if text != "" || !errors.Is(err, errIncompleteModelOutput) {
		t.Fatalf("selected truncated choice was silently replaced: %q, %v", text, err)
	}
	if !strings.Contains(err.Error(), "达到长度上限") || !strings.Contains(err.Error(), "未作为成功结果") {
		t.Fatalf("truncation diagnostic is not actionable: %v", err)
	}
}

func TestRequestModelTextRejectsHTTP200TruncatedReport(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"finish_reason":"length","message":{"content":"报告第一部分；关键结论还没"}}]}`)
	}))
	defer server.Close()
	provider := reportProvider{Endpoint: server.URL, APIKey: "local-test", Model: "local-test", Protocol: "chat_completions"}
	text, err := (&ResearchEngine{}).requestModelText(context.Background(), provider, "测试", "本地资料", 123)
	if text != "" || !errors.Is(err, errIncompleteModelOutput) {
		t.Fatalf("truncated report accepted through real request path: %q, %v", text, err)
	}
	if requests.Load() != 1 {
		t.Fatalf("semantic truncation should not be retried as transport failure: %d calls", requests.Load())
	}
}

func TestRequestModelTextPreservesResponsesOutputBudget(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode mock request: %v", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if payload["max_output_tokens"] != float64(4321) || payload["store"] != false {
			t.Errorf("responses request must preserve caller budget and privacy control: %#v", payload)
		}
		if _, exists := payload["max_tokens"]; exists {
			t.Error("responses payload received chat-only max_tokens")
		}
		_, _ = io.WriteString(w, `{"status":"completed","output":[{"type":"message","status":"completed","content":[{"type":"output_text","text":"完整的本地模拟报告"}]}]}`)
	}))
	defer server.Close()
	provider := reportProvider{Endpoint: server.URL, APIKey: "local-test", Model: "local-test", SupportsStore: true}
	text, err := (&ResearchEngine{}).requestModelText(context.Background(), provider, "测试", "本地资料", 4321)
	if text != "完整的本地模拟报告" || err != nil || requests.Load() != 1 {
		t.Fatalf("text=%q err=%v calls=%d", text, err, requests.Load())
	}
}

type modelIntegrityRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f modelIntegrityRoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func installModelIntegrityOfflineFixture(t *testing.T, transport modelIntegrityRoundTripperFunc) {
	t.Helper()
	// The real user's config, key, prompts and database are never read by these tests.
	t.Chdir(t.TempDir())
	if err := os.WriteFile("config.yaml", []byte("report_model:\n  provider: bai\n  api_key: model-integrity-offline-key\n  model: offline-model\n"), 0600); err != nil {
		t.Fatal(err)
	}
	policy := "max_rounds: 1\nmax_queries: 1\nmax_sources: 1\nread_batch: 1\ntarget_sources: 1\ntarget_domains: 1\ntarget_evidence: 1\nmax_evidence: 3\nmax_seconds: 30\n"
	if err := os.WriteFile("research_policy.yaml", []byte(policy), 0600); err != nil {
		t.Fatal(err)
	}
	original := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original })
}

func modelIntegrityOfflineResponse(req *http.Request, contentType, body string) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
}

func TestBuildReportRejectsTruncatedOutputWithoutReturningPartialReport(t *testing.T) {
	var calls atomic.Int32
	installModelIntegrityOfflineFixture(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Hostname() != "api.b.ai" || req.Header.Get("Authorization") != "Bearer model-integrity-offline-key" {
			return nil, errors.New("unexpected request blocked by offline fixture")
		}
		calls.Add(1)
		return modelIntegrityOfflineResponse(req, "application/json", `{"choices":[{"finish_reason":"length","message":{"content":"PRIVATE_PARTIAL_SHOULD_NEVER_ESCAPE"}}]}`)
	})
	report, err := (&ResearchEngine{}).buildReport(context.Background(), "AI agent workflow research", []Evidence{{ID: "E1", SourceTitle: "Offline evidence", Quote: "Offline bounded evidence quote for report synthesis."}}, Stats{Read: 1, EvidenceCount: 1}, nil)
	if !errors.Is(err, errIncompleteModelOutput) || !reflect.DeepEqual(report, Report{}) {
		t.Fatalf("buildReport must return zero report on truncation: report=%#v err=%v", report, err)
	}
	if calls.Load() != 1 || strings.Contains(err.Error(), "PRIVATE_PARTIAL") {
		t.Fatalf("unexpected repeated call or partial text exposure: calls=%d err=%v", calls.Load(), err)
	}
}

func TestResearchRunTruncationProducesWarnedFallbackAndPersistsDiagnostic(t *testing.T) {
	var modelCalls atomic.Int32
	installModelIntegrityOfflineFixture(t, func(req *http.Request) (*http.Response, error) {
		switch req.URL.Hostname() {
		case "api.b.ai":
			if req.Header.Get("Authorization") != "Bearer model-integrity-offline-key" {
				return nil, errors.New("unexpected credentials blocked by offline fixture")
			}
			switch modelCalls.Add(1) {
			case 1:
				return modelIntegrityOfflineResponse(req, "application/json", `{"choices":[{"finish_reason":"stop","message":{"content":"{\"topic\":\"AI agent workflow\",\"paperQuery\":\"\",\"candidates\":[]}"}}]}`)
			case 2:
				return modelIntegrityOfflineResponse(req, "application/json", `{"choices":[{"finish_reason":"length","message":{"content":"PRIVATE_PARTIAL_SHOULD_NEVER_ESCAPE"}}]}`)
			default:
				return nil, errors.New("unexpected model call after rejected report")
			}
		case "html.duckduckgo.com":
			return modelIntegrityOfflineResponse(req, "text/html; charset=utf-8", `<a class="result__a" href="https://integrity-publisher.test/research">AI agent workflow study</a><div class="result__snippet">AI agent customer workflow research and measured adoption.</div>`)
		case "integrity-publisher.test":
			paragraph := strings.Repeat("AI agent customer workflow studies measure adoption and document observed limitations before any product release. ", 4)
			return modelIntegrityOfflineResponse(req, "text/html; charset=utf-8", "<html><title>AI agent workflow study</title><article><p>"+paragraph+"</p></article></html>")
		default:
			return nil, errors.New("unexpected endpoint blocked by offline fixture")
		}
	})
	store, err := NewKnowledgeStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	writer := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/research", nil)
	streamResearchWithRunner(writer, request, NewResearchEngine().Run, store, "AI agent workflow research")
	if modelCalls.Load() != 2 {
		t.Fatalf("expected planner then report, not extra model calls: %d", modelCalls.Load())
	}
	if strings.Contains(writer.Body.String(), "PRIVATE_PARTIAL") {
		t.Fatal("truncated model text escaped into SSE or fallback report")
	}
	var completed ResearchResult
	for _, frame := range strings.Split(writer.Body.String(), "\n\n") {
		if strings.HasPrefix(frame, "event: complete\n") {
			if err := json.Unmarshal([]byte(strings.TrimPrefix(frame, "event: complete\ndata: ")), &completed); err != nil {
				t.Fatalf("decode completion frame: %v", err)
			}
		}
	}
	if completed.Status != "completed_with_warnings" || completed.Report.ModelUsed != "保守规则合成（模型合成未完成）" || len(completed.Evidence) == 0 {
		t.Fatalf("truncation was not clearly demoted to a warned evidence fallback: %#v", completed)
	}
	if !strings.Contains(writer.Body.String(), "输出不完整") || !strings.Contains(writer.Body.String(), "未作为成功结果") {
		t.Fatal("SSE did not clearly explain that the model report was rejected")
	}
	diagnostic, err := store.ResearchDiagnostic(completed.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if diagnostic.Status != "completed_with_warnings" || diagnostic.WarningCount < 1 || diagnostic.LastStep != "synthesize" {
		t.Fatalf("fallback diagnostic state incorrect: %#v", diagnostic)
	}
	if !strings.Contains(strings.Join(diagnostic.Warnings, ","), "model_output_incomplete") {
		t.Fatalf("truncation warning was not persisted with its specific code: %#v", diagnostic)
	}
}
