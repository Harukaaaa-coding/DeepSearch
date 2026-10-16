package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func adaptiveTestPolicy() ResearchPolicy {
	p := defaultResearchPolicy()
	p.MaxRounds = 8
	p.TargetSources = 30
	p.TargetDomains = 20
	p.TargetEvidence = 80
	return p
}

func adaptiveTestSource(hit searchHit, index int) Source {
	return Source{
		ID: fmt.Sprintf("source-%03d", index), Title: hit.Title, URL: hit.URL,
		Status: "已阅读", Trust: 80,
		Content: fmt.Sprintf("AI agent customer workflow study %d documents actual use, observed adoption, and limits that require further verification before launch.", index),
	}
}

func TestAdaptiveEvidenceLoopEnforcesQueryAndReadBudgets(t *testing.T) {
	p := adaptiveTestPolicy()
	p.MaxQueries, p.MaxSources, p.ReadBatch = 8, 5, 2
	result := ResearchResult{Question: "AI agent workflow"}
	queries, reads, searchCalls := 0, 0, 0
	attempted := map[string]bool{}
	var events []ResearchCoverage
	tools := researchTools{
		search: func(_ context.Context, batch []string, _ func(ProgressEvent), _ string) []searchHit {
			queries += len(batch)
			searchCalls++
			hits := make([]searchHit, 12)
			for i := range hits {
				hits[i] = searchHit{Title: "AI agent workflow", URL: fmt.Sprintf("https://publisher-%d-%d.test/research", searchCalls, i)}
			}
			return hits
		},
		papers: func(context.Context, string) ([]searchHit, error) {
			t.Fatal("unexpected paper search")
			return nil, nil
		},
		read: func(_ context.Context, hits []searchHit, _ func(ProgressEvent)) []Source {
			if len(hits) > p.ReadBatch {
				t.Fatalf("read batch exceeds ceiling: %d", len(hits))
			}
			var sources []Source
			for _, hit := range hits {
				url := canonicalURL(hit.URL)
				if attempted[url] {
					t.Fatalf("read attempted twice: %s", url)
				}
				attempted[url] = true
				reads++
				sources = append(sources, adaptiveTestSource(hit, reads))
			}
			return sources
		},
	}
	runEvidenceLoop(context.Background(), &result, p, ResearchSearchPlan{Topic: "AI agent workflow"}, tools, func(event ProgressEvent) {
		if event.Coverage != nil {
			events = append(events, *event.Coverage)
		}
	})
	if queries != p.MaxQueries || result.Stats.Queries != queries {
		t.Fatalf("query budget mismatch: calls=%d stats=%d ceiling=%d", queries, result.Stats.Queries, p.MaxQueries)
	}
	if reads != p.MaxSources || result.Stats.Read != reads {
		t.Fatalf("read budget mismatch: reads=%d stats=%d ceiling=%d", reads, result.Stats.Read, p.MaxSources)
	}
	if result.Coverage.StopReason != "source_budget" {
		t.Fatalf("stop reason = %q", result.Coverage.StopReason)
	}
	if len(events) == 0 || events[len(events)-1].StopReason != result.Coverage.StopReason {
		t.Fatalf("final coverage not emitted: %#v", events)
	}
	if result.Coverage.Status != "insufficient" || len(result.Coverage.Gaps) == 0 {
		t.Fatal("budget exhaustion must retain unmet coverage gaps")
	}
}

func TestAdaptiveEvidenceLoopCountsPaperQueryInsideBudget(t *testing.T) {
	p := adaptiveTestPolicy()
	p.MaxQueries, p.MaxRounds = 7, 1
	result := ResearchResult{}
	queries, paperQueries := 0, 0
	tools := researchTools{
		search: func(_ context.Context, batch []string, _ func(ProgressEvent), _ string) []searchHit {
			queries += len(batch)
			return nil
		},
		papers: func(context.Context, string) ([]searchHit, error) { paperQueries++; return nil, nil },
		read: func(context.Context, []searchHit, func(ProgressEvent)) []Source {
			t.Fatal("unexpected reading without results")
			return nil
		},
	}
	runEvidenceLoop(context.Background(), &result, p, ResearchSearchPlan{Topic: "AI agent", PaperQuery: `all:"AI agent"`}, tools, func(ProgressEvent) {})
	if paperQueries != 1 || queries+paperQueries != p.MaxQueries || result.Stats.Queries != p.MaxQueries {
		t.Fatalf("paper request not included in ceiling: web=%d paper=%d stats=%d", queries, paperQueries, result.Stats.Queries)
	}
}

func TestAdaptiveEvidenceLoopStopsAfterTwoRoundsWithoutEvidence(t *testing.T) {
	p := adaptiveTestPolicy()
	result := ResearchResult{}
	searchCalls, reads := 0, 0
	tools := researchTools{
		search: func(_ context.Context, _ []string, _ func(ProgressEvent), _ string) []searchHit {
			searchCalls++
			return []searchHit{{Title: "AI agent leads", URL: fmt.Sprintf("https://publisher-%d.test/report", searchCalls)}}
		},
		papers: func(context.Context, string) ([]searchHit, error) { return nil, nil },
		read: func(_ context.Context, hits []searchHit, _ func(ProgressEvent)) []Source {
			reads += len(hits)
			return []Source{{ID: fmt.Sprintf("unrelated-%d", reads), Title: "Geology", URL: hits[0].URL, Status: "已阅读", Content: "Volcanic rocks provide a record of mineral deposits and geological formations over long periods of time."}}
		},
	}
	runEvidenceLoop(context.Background(), &result, p, ResearchSearchPlan{Topic: "AI agent"}, tools, func(ProgressEvent) {})
	if result.Coverage.StopReason != "no_new_evidence" || searchCalls != 2 || result.Coverage.Round != 2 {
		t.Fatalf("stagnation failed to stop: calls=%d coverage=%#v", searchCalls, result.Coverage)
	}
	if len(result.Evidence) != 0 || result.Coverage.Status != "insufficient" {
		t.Fatal("unrelated pages must not count as supporting evidence")
	}
}

func TestAdaptiveEvidenceLoopFailedReadsConsumeAttemptBudget(t *testing.T) {
	p := adaptiveTestPolicy()
	p.MaxSources, p.ReadBatch = 3, 3
	result := ResearchResult{}
	readAttempts := 0
	tools := researchTools{
		search: func(context.Context, []string, func(ProgressEvent), string) []searchHit {
			var hits []searchHit
			for i := 0; i < 10; i++ {
				hits = append(hits, searchHit{Title: "AI agent", URL: fmt.Sprintf("https://blocked-%d.test/report", i)})
			}
			return hits
		},
		papers: func(context.Context, string) ([]searchHit, error) { return nil, nil },
		read: func(_ context.Context, hits []searchHit, _ func(ProgressEvent)) []Source {
			var sources []Source
			for _, hit := range hits {
				readAttempts++
				sources = append(sources, Source{ID: fmt.Sprintf("failed-%d", readAttempts), URL: hit.URL, Status: "读取失败", Error: "timeout"})
			}
			return sources
		},
	}
	runEvidenceLoop(context.Background(), &result, p, ResearchSearchPlan{Topic: "AI agent"}, tools, func(ProgressEvent) {})
	if readAttempts != 3 || result.Stats.Read != 0 || result.Coverage.StopReason != "source_budget" {
		t.Fatalf("failed reads were not budgeted correctly: attempts=%d stats=%#v coverage=%#v", readAttempts, result.Stats, result.Coverage)
	}
}

func TestAdaptiveEvidenceLoopCancelledBeforeSearch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := ResearchResult{}
	tools := researchTools{
		search: func(context.Context, []string, func(ProgressEvent), string) []searchHit {
			t.Fatal("search called after cancellation")
			return nil
		},
		papers: func(context.Context, string) ([]searchHit, error) {
			t.Fatal("paper search called after cancellation")
			return nil, nil
		},
		read: func(context.Context, []searchHit, func(ProgressEvent)) []Source {
			t.Fatal("read called after cancellation")
			return nil
		},
	}
	runEvidenceLoop(ctx, &result, adaptiveTestPolicy(), ResearchSearchPlan{Topic: "AI agent"}, tools, func(ProgressEvent) {})
	if result.Coverage.StopReason != "time_budget" || result.Stats.Queries != 0 || result.Coverage.Status != "insufficient" {
		t.Fatalf("unexpected cancellation result: %#v", result.Coverage)
	}
}

func TestAdaptiveQueriesStayFreshAndRotateCompetitorCandidates(t *testing.T) {
	plan := ResearchSearchPlan{Topic: "中国 自媒体 AI agent 工作台", Candidates: []string{"AlphaProduct", "BetaProduct", "GammaProduct", "DeltaProduct"}}
	seen := map[string]bool{}
	seenCandidates := map[string]bool{}
	for round := 1; round <= 5; round++ {
		queries := adaptiveQueries(plan, ResearchCoverage{}, round)
		if len(queries) == 0 || len(queries) > 8 {
			t.Fatalf("round %d invalid query batch length %d", round, len(queries))
		}
		for _, query := range queries {
			if seen[query] {
				t.Fatalf("round %d repeats already used query %q", round, query)
			}
			seen[query] = true
			for _, candidate := range plan.Candidates {
				if strings.Contains(query, candidate) {
					seenCandidates[candidate] = true
				}
			}
		}
	}
	for _, candidate := range plan.Candidates {
		if !seenCandidates[candidate] {
			t.Errorf("candidate never searched: %s", candidate)
		}
	}
}

func TestAdaptiveBatchFiltersIrrelevantPapersAndCanonicalDuplicates(t *testing.T) {
	hits := []searchHit{
		{Title: "Mineral composition", URL: "https://arxiv.org/abs/1000.1", Abstract: "A study of basalt, olivine and geological deposits."},
		{Title: "Agent workflow evaluation", URL: "https://arxiv.org/abs/1000.2", Abstract: "AI agent workflow reliability and bounded tool use."},
		{Title: "AI agent documentation", URL: "https://docs.vendor.test/page?utm_source=search"},
		{Title: "AI agent documentation", URL: "https://docs.vendor.test/page#section"},
		{Title: "AI agent prior article", URL: "https://prior.test/article"},
	}
	batch := selectResearchBatch("AI agent workflow", hits, map[string]bool{"https://prior.test/article": true}, nil, 8)
	if len(batch) != 2 {
		t.Fatalf("expected relevant paper plus one canonical page, got %#v", batch)
	}
	for _, hit := range batch {
		if hit.URL == "https://arxiv.org/abs/1000.1" || hit.URL == "https://prior.test/article" {
			t.Errorf("irrelevant or already attempted hit selected: %#v", hit)
		}
	}
}

func TestAdaptiveBatchBroadensDomainsBeforeRepeatingPublisher(t *testing.T) {
	hits := []searchHit{
		{Title: "AI agent", URL: "https://docs.vendor.com/a"}, {Title: "AI agent", URL: "https://docs.vendor.com/b"},
		{Title: "AI agent", URL: "https://help.vendor.com/c"}, {Title: "AI agent", URL: "https://www.vendor.com/d"},
		{Title: "AI agent", URL: "https://independent.org/report"}, {Title: "AI agent", URL: "https://university.edu/report"},
	}
	batch := selectResearchBatch("AI agent", hits, map[string]bool{}, nil, 8)
	if len(batch) != 5 {
		t.Fatalf("publisher ceiling should permit 3 vendor + 2 other results: %#v", batch)
	}
	firstDomains := map[string]bool{}
	for _, hit := range batch[:3] {
		firstDomains[sourceDomain(hit.URL)] = true
	}
	if len(firstDomains) != 3 {
		t.Fatalf("first pass did not broaden publisher domains: %#v", batch[:3])
	}
}

func TestAdaptiveEvidenceRetainsLongParagraphsAndMultipleTraceablePassages(t *testing.T) {
	paragraphs := []string{}
	for _, topic := range []string{"customer workflow", "pricing and billing", "security evaluation"} {
		paragraphs = append(paragraphs, "AI agent "+topic+" study: "+strings.Repeat("Agent operators record measured outcomes and verify each observed result against primary material. ", 5))
	}
	stamp := time.Date(2026, 10, 5, 1, 0, 0, 0, time.UTC)
	source := Source{ID: "S1", Title: "AI agent study", URL: "https://original.test/report", Status: "已阅读", Trust: 80, PublishedAt: "2026-09-12", FetchedAt: stamp, Content: strings.Join(paragraphs, "\n")}
	copySource := source
	copySource.ID, copySource.URL = "S2", "https://syndicated.test/report"
	evidence := extractResearchEvidence("AI agent workflow", []Source{copySource, source}, 20)
	if len(evidence) != 3 {
		t.Fatalf("expected three unique passages, not one per page or syndicated copies: %d %#v", len(evidence), evidence)
	}
	for i, item := range evidence {
		if len([]rune(item.Quote)) <= 420 {
			t.Errorf("long English passage discarded or shortened excessively: %d", len([]rune(item.Quote)))
		}
		if item.ID != fmt.Sprintf("E%d", i+1) || item.SourceID != "S1" || item.URL != source.URL || item.PublishedAt != source.PublishedAt || !item.FetchedAt.Equal(stamp) {
			t.Errorf("provenance not retained: %#v", item)
		}
	}
	if limited := extractResearchEvidence("AI agent workflow", []Source{source}, 2); len(limited) != 2 {
		t.Fatalf("evidence ceiling ignored: %d", len(limited))
	}
}

func TestAdaptiveEvidenceRequiresRelevantBodyNotJustTitle(t *testing.T) {
	sources := []Source{
		{ID: "S1", Title: "AI agent market", URL: "https://unrelated.test/report", Status: "已阅读", Content: "Volcanic rocks record mineral deposits and geological formations from many regions over long historical periods."},
		{ID: "S2", Title: "AI agent market", URL: "https://failed.test/report", Status: "读取失败", Content: "AI agent market studies describe adoption and outcomes but this page was never successfully read or verified."},
	}
	if got := extractResearchEvidence("AI agent", sources, 20); len(got) != 0 {
		t.Fatalf("irrelevant or unread body became evidence: %#v", got)
	}
}

func TestAdaptiveCoverageDoesNotTreatSubdomainsAsIndependentPublishers(t *testing.T) {
	p := defaultResearchPolicy()
	p.TargetSources, p.TargetDomains, p.TargetEvidence = 2, 2, 2
	quote := "AI agent market survey shows customer workflow platform competition, pricing billing documentation evaluation and security risk limitations."
	result := ResearchResult{Stats: Stats{Read: 2}, Evidence: []Evidence{
		{ID: "E1", SourceID: "S1", URL: "https://docs.vendor.com/report", Quote: quote},
		{ID: "E2", SourceID: "S2", URL: "https://help.vendor.com/report", Quote: quote + " The second page remains the same publisher."},
	}}
	coverage := assessResearchCoverage(result, p, 2)
	if coverage.Domains != 1 || coverage.Status != "insufficient" || len(coverage.Gaps) == 0 {
		t.Fatalf("subdomains incorrectly establish cross-publisher coverage: %#v", coverage)
	}
	for _, dimension := range coverage.Dimensions {
		if dimension.Domains != 1 || dimension.Status != "gap" {
			t.Errorf("same-domain dimension marked covered: %#v", dimension)
		}
	}
	result.Evidence[1].URL = "https://independent.org/report"
	coverage = assessResearchCoverage(result, p, 2)
	if coverage.Domains != 2 || coverage.Status != "sufficient" || len(coverage.Gaps) != 0 {
		t.Fatalf("all bounded coverage criteria should now be met: %#v", coverage)
	}
}

func TestAdaptiveCoverageCountsEvidenceProducingSourcesNotAllReads(t *testing.T) {
	p := defaultResearchPolicy()
	p.TargetSources, p.TargetDomains, p.TargetEvidence = 2, 1, 1
	result := ResearchResult{Stats: Stats{Read: 12}, Evidence: []Evidence{{ID: "E1", SourceID: "S1", URL: "https://primary.test/report", Quote: "AI agent market survey"}}}
	coverage := assessResearchCoverage(result, p, 1)
	if coverage.SourcesRead != 12 || coverage.Status != "insufficient" {
		t.Fatalf("successful reads are not equivalent to useful sources: %#v", coverage)
	}
	if !strings.Contains(strings.Join(coverage.Gaps, " "), "1/2") {
		t.Fatalf("missing useful-source coverage deficit: %#v", coverage.Gaps)
	}
}

func TestAdaptiveFallbackSearchPlanPreservesProductScenario(t *testing.T) {
	plan := fallbackSearchPlan("现在呢如果我要做一个中国自媒体团队使用的 AI Agent 工作台，我应该怎么做，请研究竞品和定价")
	for _, term := range []string{"中国", "自媒体", "工作台"} {
		if !strings.Contains(plan.Topic, term) {
			t.Errorf("fallback topic lost scenario %q: %q", term, plan.Topic)
		}
	}
	if !strings.Contains(strings.ToLower(plan.Topic), "agent") || plan.PaperQuery == "" {
		t.Fatalf("agent-specific search planning missing: %#v", plan)
	}
}
