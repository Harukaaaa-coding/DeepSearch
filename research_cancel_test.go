package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

type researchCancelRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f researchCancelRoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestResearchRunReturnsCancellationDuringReportModel(t *testing.T) {
	for _, responseArrives := range []bool{false, true} {
		t.Run(fmt.Sprintf("response_arrives_%t", responseArrives), func(t *testing.T) {
			// Isolate config and prompts: never load the user's credentials.
			t.Chdir(t.TempDir())
			if err := os.WriteFile("config.yaml", []byte("report_model:\n  provider: bai\n  api_key: offline-unit-test-key\n  model: gpt-5.2\n"), 0600); err != nil {
				t.Fatal(err)
			}
			policy := "max_rounds: 1\nmax_queries: 1\nmax_sources: 1\nread_batch: 1\ntarget_sources: 1\ntarget_domains: 1\ntarget_evidence: 1\nmax_evidence: 3\nmax_seconds: 30\n"
			if err := os.WriteFile("research_policy.yaml", []byte(policy), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var modelCalls atomic.Int32
			var finished atomic.Bool
			var synthesisStarted atomic.Bool
			response := func(req *http.Request, contentType, body string) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
			}
			modelResponse := func(req *http.Request, content string) (*http.Response, error) {
				body, err := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": content}}}})
				if err != nil {
					return nil, err
				}
				return response(req, "application/json", string(body))
			}
			originalTransport := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = originalTransport })
			// Every request is intercepted, including planner, search and page reads.
			// Unknown endpoints fail locally instead of delegating to real networking.
			http.DefaultTransport = researchCancelRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				switch req.URL.Hostname() {
				case "api.b.ai":
					if req.Header.Get("Authorization") != "Bearer offline-unit-test-key" {
						return nil, errors.New("test did not use isolated synthetic credentials")
					}
					switch modelCalls.Add(1) {
					case 1:
						return modelResponse(req, `{"topic":"AI agent workflow","paperQuery":"","candidates":[]}`)
					case 2:
						cancel()
						if responseArrives {
							return modelResponse(req, "A bounded research finding supported by [E1].")
						}
						return nil, context.Canceled
					default:
						return nil, errors.New("unexpected model call after cancellation")
					}
				case "html.duckduckgo.com":
					return response(req, "text/html; charset=utf-8", `<a class="result__a" href="https://offline-publisher.test/research">AI agent workflow study</a><div class="result__snippet">AI agent customer workflow research and measured adoption.</div>`)
				case "offline-publisher.test":
					paragraph := strings.Repeat("AI agent customer workflow studies measure adoption and document observed limitations before any product release. ", 4)
					return response(req, "text/html; charset=utf-8", "<html><title>AI agent workflow study</title><article><p>"+paragraph+"</p></article></html>")
				default:
					return nil, fmt.Errorf("unexpected offline test endpoint: %s", req.URL.Hostname())
				}
			})
			result, err := NewResearchEngine().Run(ctx, "AI agent workflow research", func(event ProgressEvent) {
				if event.Step == "synthesize" {
					synthesisStarted.Store(true)
				}
				if event.Kind == "finished" {
					finished.Store(true)
				}
			})
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Run must preserve cancellation, got error %v", err)
			}
			if modelCalls.Load() != 2 || !synthesisStarted.Load() || len(result.Evidence) == 0 {
				t.Fatalf("test did not reach report synthesis with real extracted fixture evidence: calls=%d synthesis=%t evidence=%d", modelCalls.Load(), synthesisStarted.Load(), len(result.Evidence))
			}
			if finished.Load() || !result.CompletedAt.IsZero() {
				t.Fatalf("cancelled research was marked complete: finished=%t completedAt=%v", finished.Load(), result.CompletedAt)
			}
		})
	}
}
