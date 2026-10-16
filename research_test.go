package main

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestUnwrapDuckDuckGoProtocolRelativeURL(t *testing.T) {
	raw := "//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fresearch"
	if got, want := unwrapDuckDuckGo(raw), "https://example.com/research"; got != want {
		t.Fatalf("unwrapDuckDuckGo() = %q, want %q", got, want)
	}
}

func TestParseDuckDuckGoReturnsSafeResult(t *testing.T) {
	page := `<a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Freport">Market report</a><div class="result__snippet">Evidence summary</div>`
	hits := parseDuckDuckGo(page, "market report")
	if len(hits) != 1 {
		t.Fatalf("parseDuckDuckGo() returned %d hits, want 1", len(hits))
	}
	if hits[0].URL != "https://example.com/report" {
		t.Fatalf("URL = %q", hits[0].URL)
	}
}

func TestReportProviderUsesDeepSeekResponsesAPI(t *testing.T) {
	provider, err := newReportProvider("deepseek", "test-key", "")
	if err != nil {
		t.Fatalf("newReportProvider() error = %v", err)
	}
	if provider.Endpoint != "https://api.deepseek.com/responses" || provider.Model != "deepseek-flash" {
		t.Fatalf("unexpected DeepSeek config: %#v", provider)
	}
}

func TestReportProviderUsesBaiChatCompletionsAPI(t *testing.T) {
	provider, err := newReportProvider("bai", "test-key", "gpt-5.2")
	if err != nil {
		t.Fatalf("newReportProvider() error = %v", err)
	}
	if provider.Endpoint != "https://api.b.ai/v1/chat/completions" || provider.Model != "gpt-5.2" || provider.Protocol != "chat_completions" {
		t.Fatalf("unexpected B.AI config: %#v", provider)
	}
	payload := modelRequestPayload(provider, "system instruction", "user evidence", 1234)
	if payload["max_tokens"] != 1234 || payload["temperature"] != 0.2 {
		t.Fatalf("unexpected B.AI payload controls: %#v", payload)
	}
	messages, ok := payload["messages"].([]map[string]string)
	if !ok || len(messages) != 2 || messages[0]["role"] != "system" || messages[1]["content"] != "user evidence" {
		t.Fatalf("B.AI payload must use chat messages: %#v", payload)
	}
}

func TestResponseOutputTextReadsChatCompletionsChoice(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"content":"模型合成的具体结论"}}]}`)
	if got := responseOutputText(body); got != "模型合成的具体结论" {
		t.Fatalf("responseOutputText() = %q", got)
	}
}
func TestReportProviderReadsDeepSeekYAML(t *testing.T) {
	provider, configured, err := reportProviderFromYAML([]byte("report_model:\n  provider: deepseek\n  api_key: test-key\n  model: deepseek-flash\n"))
	if err != nil || !configured {
		t.Fatalf("reportProviderFromYAML() = %#v, %t, %v", provider, configured, err)
	}
	if provider.Endpoint != "https://api.deepseek.com/responses" || provider.Model != "deepseek-flash" {
		t.Fatalf("unexpected YAML provider: %#v", provider)
	}
}

func TestDecodePageTextReadsGBK(t *testing.T) {
	page := `<html><head><meta charset="gbk"></head><body><p>中国企业级 AI Agent 市场正在增长。</p></body></html>`
	encoded, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(page))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodePageText(bytes.NewReader(encoded), "text/html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(decoded, "中国企业级 AI Agent") {
		t.Fatalf("GBK content was not decoded: %q", decoded)
	}
}

func TestParseArxivMarksPreprintAndPaperMetadata(t *testing.T) {
	feed := `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><entry><id>http://arxiv.org/abs/2501.12345</id><title>  Reliable Agent Systems </title><summary> This paper evaluates long-horizon agent reliability with a reproducible benchmark and documents failure modes across several tasks. </summary><published>2025-01-20T00:00:00Z</published><author><name>Ada Lovelace</name></author><author><name>Grace Hopper</name></author></entry></feed>`
	hits := parseArxiv(feed, "agent reliability")
	if len(hits) != 1 {
		t.Fatalf("parseArxiv() returned %d hits, want 1", len(hits))
	}
	hit := hits[0]
	if hit.Kind != "论文 / arXiv 预印本" || hit.URL != "https://arxiv.org/abs/2501.12345" || hit.PDFURL != "https://arxiv.org/pdf/2501.12345.pdf" {
		t.Fatalf("unexpected paper hit: %#v", hit)
	}
	if hit.Authors != "Ada Lovelace, Grace Hopper" || hit.PublishedAt != "2025-01-20" {
		t.Fatalf("paper metadata missing: %#v", hit)
	}
}

func TestPrioritizeHitsKeepsPapersInReadingSet(t *testing.T) {
	hits := prioritizeHits([]searchHit{{Title: "Web", URL: "https://example.com/a"}, {Title: "Paper", URL: "https://arxiv.org/abs/1", Kind: "论文 / arXiv 预印本"}})
	if hits[0].Title != "Paper" {
		t.Fatalf("expected paper to be prioritized, got %#v", hits)
	}
}
