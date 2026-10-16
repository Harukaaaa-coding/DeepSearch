package main

import (
	"reflect"
	"strings"
	"testing"
)

func scenarioResearchFixture() ResearchResult {
	return ResearchResult{
		RunID:    "scenario-content-audit",
		Question: "为中国自媒体团队设计品牌内容发布前审校 Agent",
		Evidence: []Evidence{
			{ID: "E1", SourceID: "S1", SourceTitle: "品牌审校工具使用文档", URL: "https://docs.example.test/brand-review", PublishedAt: "2026-09-12", Quote: "审核员上传字幕后，工具逐条标记品牌禁用词及其时间戳，并保留原始字幕以供人工复核。"},
			{ID: "E2", SourceID: "S2", SourceTitle: "视频协作工作流", URL: "https://workflow.example.test/video", Quote: "剪辑师在交付视频前提交字幕，审核员在协作平台记录意见并返回需要修改的时间段。"},
		},
		Report: Report{Analysis: "品牌内容审校可以作为待验证的首个场景，需要观察审核员的实际返工和交接过程。"},
	}
}

func detailedProductScenarioFixture() ProductScenario {
	return ProductScenario{
		ID: "SCN-001", Name: "品牌合作视频字幕发布前审校", Actor: "MCN 品牌审核员",
		Trigger:          "剪辑师提交品牌合作视频字幕，尚未进入发布排期",
		Context:          "审核员在品牌合作短视频交付前核对禁用词与修改意见；是否为团队真实痛点仍待访谈。",
		Inputs:           []string{"带时间戳的字幕文件", "经品牌负责人确认的禁用词表"},
		CurrentWorkflow:  []string{"接收字幕并打开品牌词表", "逐段查看字幕并记录修改时间戳"},
		CurrentGap:       "假设：修改意见与字幕版本分离，审核员需要反复核对已处理项；尚无真实返工率基线。",
		Impact:           "假设：未处理的禁用词会增加品牌复核返工；影响规模待实际观察。",
		ProposedWorkflow: []string{"上传字幕并锁定品牌词表版本", "按时间戳显示命中词与原文", "审核员逐项处理并签署复核结果"},
		Outcome:          "生成保留原字幕、命中位置、处理人和处理状态的审校清单，不自动发布视频。",
		Acceptance:       []string{"Given 字幕包含两个禁用词 When 审校完成 Then 清单显示两处原文和时间戳，未处理项不允许标记为复核通过"},
		Exceptions:       []string{"字幕缺少时间戳时提示重新上传，不生成伪造的定位信息"},
		Validation:       "邀请目标审核员用已授权历史字幕走查，记录漏检、误报和复核耗时后再确定试点门槛。",
		Basis:            "research_hypothesis",
		EvidenceLinks:    []ScenarioEvidenceLink{{ID: "E1", Field: "currentWorkflow", Quote: "工具逐条标记品牌禁用词及其时间戳", Explanation: "来源描述了字幕审校的输入与定位方式，仅支持现有工具能力，不证明客户的返工率。"}},
	}
}

func TestValidateResearchScenariosRequiresActionableDetails(t *testing.T) {
	mutations := map[string]func(*ProductScenario){
		"name":             func(s *ProductScenario) { s.Name = " " },
		"actor":            func(s *ProductScenario) { s.Actor = "" },
		"trigger":          func(s *ProductScenario) { s.Trigger = "" },
		"currentGap":       func(s *ProductScenario) { s.CurrentGap = "" },
		"outcome":          func(s *ProductScenario) { s.Outcome = "" },
		"context":          func(s *ProductScenario) { s.Context = "" },
		"impact":           func(s *ProductScenario) { s.Impact = "" },
		"validation":       func(s *ProductScenario) { s.Validation = "" },
		"inputs":           func(s *ProductScenario) { s.Inputs = nil },
		"currentWorkflow":  func(s *ProductScenario) { s.CurrentWorkflow = nil },
		"proposedWorkflow": func(s *ProductScenario) { s.ProposedWorkflow = []string{"只有一步"} },
		"acceptance":       func(s *ProductScenario) { s.Acceptance = nil },
		"exceptions":       func(s *ProductScenario) { s.Exceptions = nil },
	}
	for field, mutate := range mutations {
		t.Run(field, func(t *testing.T) {
			scenario := detailedProductScenarioFixture()
			mutate(&scenario)
			if _, err := validateResearchScenarios([]ProductScenario{scenario}, scenarioResearchFixture()); err == nil {
				t.Fatalf("missing %s must not pass as an actionable scenario", field)
			}
		})
	}
	if _, err := validateResearchScenarios([]ProductScenario{{}}, scenarioResearchFixture()); err == nil {
		t.Fatal("an empty model object must be rejected")
	}
}

func TestValidateResearchScenariosRejectsProjectReviewTemplate(t *testing.T) {
	for _, name := range []string{"机会判断评审", "方案范围评审", "试点与发布评审"} {
		t.Run(name, func(t *testing.T) {
			scenario := detailedProductScenarioFixture()
			scenario.Name = name
			if _, err := validateResearchScenarios([]ProductScenario{scenario}, scenarioResearchFixture()); err == nil {
				t.Fatal("the old delivery-review card must not masquerade as the customer's product-use scenario")
			}
		})
	}
}

func TestValidateResearchScenariosKeepsOnlyVerifiedEvidenceLinks(t *testing.T) {
	result := scenarioResearchFixture()
	scenario := detailedProductScenarioFixture()
	scenario.Evidence = []EvidenceRef{{ID: "E999", Type: "public_research", Label: "模型编造的引用"}}
	scenario.EvidenceLinks[0].SourceTitle = "模型伪造的标题"
	scenario.EvidenceLinks[0].URL = "https://untrusted.example.test/fake"
	scenario.EvidenceLinks[0].PublishedAt = "2099-01-01"
	scenario.EvidenceLinks = append(scenario.EvidenceLinks,
		ScenarioEvidenceLink{ID: "E999", Field: "trigger", Quote: "工具逐条标记品牌禁用词及其时间戳", Explanation: "不存在的证据编号"},
		ScenarioEvidenceLink{ID: "E2", Field: "currentGap", Quote: "审核员每次返工都损失三千元收入", Explanation: "真实编号但引文不在来源内"},
		ScenarioEvidenceLink{ID: "E1", Field: "outcome", Quote: "工具逐条标记品牌禁用词及其时间戳", Explanation: "现有资料不能证明新方案必定实现结果"},
		ScenarioEvidenceLink{ID: "E2", Field: "currentWorkflow", Quote: "审核员在协作平台记录意见并返回需要修改的时间段", Explanation: ""},
	)
	items, err := validateResearchScenarios([]ProductScenario{scenario}, result)
	if err != nil || len(items) != 1 {
		t.Fatalf("one well-formed scenario should survive citation cleanup: count=%d err=%v", len(items), err)
	}
	got := items[0]
	if len(got.EvidenceLinks) != 1 || got.EvidenceLinks[0].ID != "E1" {
		t.Fatalf("only the exact, attributed quote should remain: %#v", got.EvidenceLinks)
	}
	link := got.EvidenceLinks[0]
	if link.SourceTitle != result.Evidence[0].SourceTitle || link.URL != result.Evidence[0].URL || link.PublishedAt != result.Evidence[0].PublishedAt {
		t.Fatalf("source metadata must come from the server evidence store, not model output: %#v", link)
	}
	if got.Basis != "research_hypothesis" {
		t.Fatalf("public evidence must support a hypothesis, not mark it as validated customer research: %s", got.Basis)
	}
	if len(got.Evidence) != 1 || got.Evidence[0].ID != "E1" {
		t.Fatalf("display citations must be derived solely from the surviving links: %#v", got.Evidence)
	}
}

func TestValidateResearchScenariosNormalizesWhitespaceButDoesNotInventSupport(t *testing.T) {
	result := scenarioResearchFixture()
	result.Evidence[0].Quote = "审核员上传字幕后，工具逐条标记品牌禁用词及其时间戳，\n并保留原始字幕以供人工复核。"
	scenario := detailedProductScenarioFixture()
	scenario.EvidenceLinks[0].Quote = "工具逐条标记品牌禁用词及其时间戳， 并保留原始字幕以供人工复核。"
	items, err := validateResearchScenarios([]ProductScenario{scenario}, result)
	if err != nil || len(items) != 1 || len(items[0].EvidenceLinks) != 1 {
		t.Fatalf("layout whitespace in a real quote should be tolerated: %#v, %v", items, err)
	}
	scenario.EvidenceLinks = nil
	scenario.Evidence = []EvidenceRef{{ID: "E1", Type: "public_research"}}
	scenario.Basis = "verified_customer_fact"
	items, err = validateResearchScenarios([]ProductScenario{scenario}, result)
	if err != nil || len(items) != 1 {
		t.Fatalf("unsupported but complete scenario should remain a visible hypothesis: %#v, %v", items, err)
	}
	if items[0].Basis != "unverified_hypothesis" {
		t.Fatalf("plain evidence IDs are not evidence links: %q", items[0].Basis)
	}
	for _, ref := range items[0].Evidence {
		if ref.ID != "" {
			t.Fatalf("an unsupported scenario must not inherit arbitrary evidence IDs: %#v", items[0].Evidence)
		}
	}
}

func TestProductScenariosFallbackUsesOnlyThisBlueprintUsers(t *testing.T) {
	result := scenarioResearchFixture()
	result.ProductBlueprint = &ProductBlueprint{Users: []PRDUserScenario{
		{Role: "品牌审核员", Scenario: "品牌合作短视频字幕复核", Trigger: "收到新版字幕", Goal: "输出待修改的品牌词清单", Evidence: []EvidenceRef{{ID: "E1"}}},
		{Role: "视频剪辑师", Scenario: "按时间戳处理品牌复核意见", Trigger: "收到复核退回单", Goal: "提交可对照的字幕修订版本", Evidence: []EvidenceRef{{ID: "E2"}}},
	}}
	items := productScenariosFromResearch(result)
	if len(items) != 2 {
		t.Fatalf("legacy blueprint should yield two explicit user-scenario candidates, got %#v", items)
	}
	for index, item := range items {
		user := result.ProductBlueprint.Users[index]
		if item.Actor != user.Role || item.Name != user.Scenario || item.Trigger != user.Trigger || item.Outcome != user.Goal {
			t.Fatalf("fallback must preserve the concrete role, trigger and outcome supplied by the blueprint: %#v", item)
		}
		if item.Basis != "needs_research" || len(item.EvidenceLinks) != 0 {
			t.Fatalf("legacy users lack quote-level observations and need research, got %#v", item)
		}
		for _, ref := range item.Evidence {
			if ref.ID != "" {
				t.Fatalf("fallback must not attach an old or round-robin citation: %#v", item.Evidence)
			}
		}
	}
	result.ProductBlueprint = nil
	if got := productScenariosFromResearch(result); len(got) != 0 {
		t.Fatalf("no research scenarios and no blueprint must not recreate the generic three-card template: %#v", got)
	}
}

func TestResearchScenariosRemainBoundToQuestionThroughBlueprintAndDelivery(t *testing.T) {
	result := scenarioResearchFixture()
	result.Scenarios = []ProductScenario{detailedProductScenarioFixture()}
	result.ScenarioStatus = "ready"
	result.ProductBlueprint = &ProductBlueprint{ProductName: "品牌审校助手", Users: []PRDUserScenario{{Role: "旧版产品负责人", Scenario: "容易覆盖场景的旧版简略描述"}}}
	want, err := validateResearchScenarios(result.Scenarios, result)
	if err != nil {
		t.Fatal(err)
	}
	input := productBlueprintInput(result)
	for _, detail := range []string{result.Scenarios[0].ID, result.Scenarios[0].Name, result.Scenarios[0].Trigger, result.Scenarios[0].CurrentGap, result.Scenarios[0].Inputs[0], result.Scenarios[0].ProposedWorkflow[1], result.Scenarios[0].Acceptance[0], result.Scenarios[0].Validation} {
		if !strings.Contains(input, detail) {
			t.Errorf("blueprint stage lost concrete scenario detail %q", detail)
		}
	}
	analysis, err := productAnalysisFromResearch(result)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(analysis.Scenarios, want) {
		t.Fatalf("product discovery did not preserve research scenarios: %#v", analysis.Scenarios)
	}
	store := &KnowledgeStore{}
	delivery, err := store.BuildDeliveryCopilotForResearch(result)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(delivery.Discovery.Scenarios, want) {
		t.Fatalf("applyProductBlueprint replaced detailed research scenarios with abbreviated users: %#v", delivery.Discovery.Scenarios)
	}
	other := ResearchResult{RunID: "scenario-inventory", Question: "为仓库班组设计临期食品库存盘点工具", ProductBlueprint: &ProductBlueprint{Users: []PRDUserScenario{{Role: "仓库盘点员", Scenario: "临期食品批次核对", Trigger: "收到当日临期批次清单", Goal: "记录实物数量与账面差异"}}}}
	otherScenarios := productScenariosFromResearch(other)
	if len(otherScenarios) != 1 || otherScenarios[0].Actor != "仓库盘点员" {
		t.Fatalf("another product must derive its own users and scenes: %#v", otherScenarios)
	}
	otherInput := productBlueprintInput(other)
	if strings.Contains(otherInput, "品牌审校") || strings.Contains(otherInput, result.Scenarios[0].CurrentGap) {
		t.Fatal("scenario content leaked between separate research questions")
	}
}
