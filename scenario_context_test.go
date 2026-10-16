package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestScenarioQueriesContinueBeyondGenericUserCoverage(t *testing.T) {
	plan := ResearchSearchPlan{Topic: "中国自媒体工作台", ScenarioQueries: []string{"自媒体运营 选题表 素材交接", "视频团队 审稿 返工 版本", "多平台运营 改稿 用户反馈"}}
	coverage := ResearchCoverage{Dimensions: []CoverageDimension{{ID: "users", Status: "covered"}, {ID: "pricing", Status: "gap"}}}
	for round := 1; round <= 3; round++ {
		queries := adaptiveQueries(plan, coverage, round)
		if !strings.Contains(strings.Join(queries, "\n"), plan.ScenarioQueries[round-1]) {
			t.Fatalf("round %d discarded specific business query: %v", round, queries)
		}
		if len(queries) > 8 {
			t.Fatal("scenario queries exceeded batch budget")
		}
	}
}

func TestScenarioBlueprintContextLeavesRoomForEvidence(t *testing.T) {
	result := ResearchResult{Question: "设计自媒体内容工作台", Report: Report{Analysis: strings.Repeat("研究限制。", 800)}}
	for i := 0; i < 4; i++ {
		item := detailedProductScenarioFixture()
		item.ID = fmt.Sprintf("SCN-%03d", i+1)
		item.CurrentGap = strings.Repeat("具体障碍。", 500)
		item.ProposedWorkflow = []string{strings.Repeat("输入材料处理。", 500), strings.Repeat("输出审核版本。", 500)}
		item.Acceptance = []string{strings.Repeat("可验证验收。", 500)}
		result.Scenarios = append(result.Scenarios, item)
	}
	for i := 1; i <= 36; i++ {
		result.Evidence = append(result.Evidence, Evidence{ID: fmt.Sprintf("E%d", i), SourceID: fmt.Sprint(i), SourceTitle: "业务任务资料", URL: "https://example.com/field-study", Quote: strings.Repeat("业务流程观察及限制。", 100)})
	}
	input := productBlueprintInput(result)
	if len([]rune(input)) > 20500 {
		t.Fatalf("PRD context grew past budget: %d", len([]rune(input)))
	}
	if !strings.Contains(input, "SCN-004") || !strings.Contains(input, "[E1]") {
		t.Fatal("scenario summary crowded out scenarios/evidence")
	}
	if len([]rune(scenarioBlueprintContext(result.Scenarios))) > 6500 {
		t.Fatal("scenario contract exceeded its sub-budget")
	}
}
