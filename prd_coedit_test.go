package main

import (
	"strings"
	"testing"
)

func TestResearchAnalysisKeepsPublicResearchOutOfCustomerFeedback(t *testing.T) {
	result := ResearchResult{
		RunID:    "run-1",
		Question: "中国 AI Agent 工作台应该先服务哪个业务场景",
		Report:   Report{Analysis: "公开研究摘要：企业希望先验证可靠性。", Conflicts: []string{"不同来源的市场规模口径不一致"}},
		Evidence: []Evidence{{ID: "E1", SourceTitle: "公开报告"}},
	}
	analysis, err := productAnalysisFromResearch(result)
	if err != nil {
		t.Fatalf("productAnalysisFromResearch() error = %v", err)
	}
	if len(analysis.UserFeedback) != 0 {
		t.Fatalf("public research must not be labelled as customer feedback: %#v", analysis.UserFeedback)
	}
	copilot := tailorDeliveryForResearch(DeliveryCopilot{}, result)
	if strings.Contains(copilot.PRD.Background.CurrentSituation, result.Report.Analysis) {
		t.Fatalf("PRD current situation must not repeat research summary: %q", copilot.PRD.Background.CurrentSituation)
	}
}

func TestLocalCoEditFallbackKeepsHumanInputDistinct(t *testing.T) {
	response := localCoEditFallback(PRDCoEditRequest{Actor: "客户", Message: "审核前必须看见证据来源，否则不会使用。"}, "model unavailable")
	if len(response.Proposal.Changes) != 1 {
		t.Fatalf("changes = %d, want 1", len(response.Proposal.Changes))
	}
	change := response.Proposal.Changes[0]
	if change.Target != "discovery.userFeedback" || change.EvidenceStatus != "human_input" {
		t.Fatalf("human input tagging = %#v", change)
	}
	if change.Feedback == nil || len(change.Feedback.Evidence) != 1 || change.Feedback.Evidence[0].Type != "human_input" {
		t.Fatalf("feedback evidence = %#v", change.Feedback)
	}
}

func TestSanitizePRDCoEditProposalDropsUnsupportedChanges(t *testing.T) {
	proposal := sanitizePRDCoEditProposal(PRDCoEditProposal{Changes: []PRDCoEditChange{
		{Target: "prd.everything", Operation: "replace", After: "unsafe"},
		{Target: "background.userProblem", Operation: "replace", After: "客户在审核场景中无法判断结论可信度", EvidenceStatus: "unknown"},
	}}, "产品负责人")
	if len(proposal.Changes) != 1 {
		t.Fatalf("sanitized changes = %#v", proposal.Changes)
	}
	if proposal.Changes[0].EvidenceStatus != "human_input" {
		t.Fatalf("unexpected evidence status: %q", proposal.Changes[0].EvidenceStatus)
	}
}
