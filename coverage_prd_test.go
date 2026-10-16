package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestCoveragePRDUsesProposedProductBackground(t *testing.T) {
	result := ResearchResult{Question: "中国自媒体内容审核 Agent 如何验证 MVP", ProductBlueprint: &ProductBlueprint{
		ProductName: "创作者审核助手", OneSentence: "发布前核对品牌素材与审核意见", Reasoning: "先验证人工审核意见能否回到具体素材",
		Users: []PRDUserScenario{{Role: "MCN 审核员", Scenario: "短视频发布前审稿", Goal: "确认素材修改后再批准发布"}},
		MVP:   []string{"素材意见关联"}, UserFlow: []string{"提交素材", "标注审核意见", "人工确认"},
	}}
	prd := tailorDeliveryForResearch(DeliveryCopilot{}, result).PRD
	for _, item := range []struct{ text, want string }{{prd.Background.BusinessContext, "创作者审核助手"}, {prd.Background.BusinessContext, "先验证人工审核意见"}, {prd.Background.UserProblem, "MCN 审核员"}, {prd.Background.UserProblem, "短视频发布前审稿"}, {prd.Background.CurrentSituation, "拟验证范围：素材意见关联"}, {prd.Background.CurrentSituation, "提交素材 → 标注审核意见 → 人工确认"}, {prd.Background.CurrentSituation, "不代表客户现行业务事实"}} {
		if !strings.Contains(item.text, item.want) {
			t.Errorf("background missing %q: %s", item.want, item.text)
		}
	}
}

func TestCoveragePRDGapsSurviveBlueprintAndDOCX(t *testing.T) {
	for _, withBlueprint := range []bool{false, true} {
		result := ResearchResult{Question: "研究中国内容工作台市场与产品机会", Coverage: ResearchCoverage{Status: "insufficient", StopReason: "time_budget", Gaps: []string{"缺少竞品官方定价", "缺少竞品官方定价"}}}
		if withBlueprint {
			result.ProductBlueprint = &ProductBlueprint{ProductName: "内容工作台", OpenQuestions: []string{"业务负责人待确认", "研究补证：缺少竞品官方定价"}}
		}
		copilot := tailorDeliveryForResearch(DeliveryCopilot{}, result)
		questions := strings.Join(copilot.PRD.OpenQuestions, "\n")
		for _, want := range []string{"证据覆盖不足", "达到研究时间预算", "研究补证：缺少竞品官方定价"} {
			if !strings.Contains(questions, want) {
				t.Errorf("withBlueprint=%v: questions missing %q", withBlueprint, want)
			}
			if !strings.Contains(deliveryPRDDocxDocument(copilot), want) {
				t.Errorf("withBlueprint=%v: DOCX missing %q", withBlueprint, want)
			}
		}
		if strings.Count(questions, "研究补证：缺少竞品官方定价") != 1 {
			t.Errorf("duplicate coverage gap: %s", questions)
		}
	}
}

func TestCoveragePRDOldTasksRemainCompatible(t *testing.T) {
	original := DeliveryCopilot{PRD: EnterprisePRD{OpenQuestions: []string{"已有评审问题"}}, Solution: SolutionDesign{TechnicalConstraints: []string{"现有约束"}}}
	if got := applyResearchCoverageToPRD(original, ResearchCoverage{}); !reflect.DeepEqual(got, original) {
		t.Fatal("empty coverage changed legacy PRD")
	}
	got := applyResearchCoverageToPRD(original, ResearchCoverage{Status: "sufficient", StopReason: "coverage_target"})
	text := strings.Join(got.PRD.OpenQuestions, "\n")
	if strings.Contains(text, "证据覆盖不足") || !strings.Contains(text, "仍需核验具体结论") {
		t.Fatalf("incorrect sufficient-coverage boundary: %s", text)
	}
}
