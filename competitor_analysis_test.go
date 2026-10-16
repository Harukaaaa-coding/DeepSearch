package main

import "testing"

func TestSanitizeCompetitorAnalysisRequiresKnownEvidence(t *testing.T) {
	result := ResearchResult{Evidence: []Evidence{{ID: "E1", SourceTitle: "有证据产品 官方产品页"}}}
	analysis := sanitizeCompetitorAnalysis(CompetitorAnalysis{
		Competitors: []CompetitorProfile{
			{Name: "有证据产品", Category: "工作台", Evidence: []EvidenceRef{{ID: "E1"}, {ID: "unknown"}}},
			{Name: "无证据产品", Evidence: []EvidenceRef{{ID: "unknown"}}},
		},
		Opportunities: []CompetitorOpportunity{
			{Hypothesis: "有来源的待验证机会", Validation: "访谈 5 位目标用户", Evidence: []EvidenceRef{{ID: "E1"}}},
			{Hypothesis: "无来源机会", Validation: "访谈", Evidence: []EvidenceRef{{ID: "unknown"}}},
		},
	}, result)
	if len(analysis.Competitors) != 1 || analysis.Competitors[0].Name != "有证据产品" {
		t.Fatalf("unexpected competitors: %#v", analysis.Competitors)
	}
	if analysis.Competitors[0].Pricing != "公开资料未核验到定价" {
		t.Fatalf("unexpected pricing default: %q", analysis.Competitors[0].Pricing)
	}
	if len(analysis.Opportunities) != 1 || len(analysis.Opportunities[0].Evidence) != 1 {
		t.Fatalf("unexpected opportunities: %#v", analysis.Opportunities)
	}
}

func TestCompetitorNamesAndPricesNeedMatchingEvidence(t *testing.T) {
	result := ResearchResult{Evidence: []Evidence{{ID: "E1", SourceTitle: "Alpha 官方文档", Quote: "Alpha provides agent workflows; pricing starts at $20 per seat monthly."}}}
	got := sanitizeCompetitorAnalysis(CompetitorAnalysis{Competitors: []CompetitorProfile{
		{Name: "Beta", Evidence: []EvidenceRef{{ID: "E1"}}},
		{Name: "Alpha", Pricing: "99 元/月", Evidence: []EvidenceRef{{ID: "E1"}}},
	}}, result)
	if len(got.Competitors) != 1 || got.Competitors[0].Name != "Alpha" {
		t.Fatalf("unrelated citation accepted: %#v", got)
	}
	if got.Competitors[0].Pricing != "公开资料未核验到定价" {
		t.Fatal("unsupported numeric price accepted")
	}
}

func TestWantsCompetitorAnalysisForMarketAndProductQuestions(t *testing.T) {
	if !wantsCompetitorAnalysis("中国 AI Agent 市场的竞品与创业机会") {
		t.Fatal("market competition question should enable competition analysis")
	}
	if !wantsCompetitorAnalysis("设计一个企业 AI 工作台的产品需求") {
		t.Fatal("product discovery question should enable competition analysis")
	}
	if wantsCompetitorAnalysis("今天北京会下雨吗") {
		t.Fatal("unrelated question should not enable competition analysis")
	}
}
