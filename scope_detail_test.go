package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"
)

func scopeDetailCopilotFixture() DeliveryCopilot {
	return DeliveryCopilot{
		PRD: EnterprisePRD{
			Info:           PRDDocumentInfo{Product: "仓库临期盘点助手"},
			Goals:          []string{"让仓库盘点员核对当日临期批次并形成差异确认单", "减少交班时找不到已确认版本的情况；基线待采集"},
			NonGoals:       []string{"不承诺消除所有库存损耗；试点只验证盘点和差异交接"},
			SuccessMetrics: []SuccessMetric{{Name: "差异闭环率", Definition: "已由班组长确认的差异单数/发现的差异单数", Baseline: "试点前记录七天基线", Target: "目标假设：减少未确认差异", Window: "两次交班周期", Owner: "仓库班组长"}},
			Users:          []PRDUserScenario{{Role: "夜班盘点员", Scenario: "临期食品交班盘点", Trigger: "收到临期批次清单", Goal: "形成可复核差异记录"}},
			Requirements:   []ExecutionRequirement{{ID: "REQ-W01", Title: "批次差异确认", UserStory: "夜班盘点员确认账实差异后交给班组长", Acceptance: []string{"Given 已导入批次清单 When 保存实盘数 Then 显示差异数并保留导入批次标识"}}},
		},
		Decision: ProductDecision{
			MVP:        []string{"批次清单导入、实盘数录入、差异确认单导出"},
			OutOfScope: []string{"首版不自动回写ERP库存；采用差异单导出供授权人员人工处理"},
		},
	}
}

func scopeDetailLists(copilot DeliveryCopilot) map[string][]string {
	return map[string][]string{
		"goals":               copilot.PRD.Goals,
		"nonGoals":            copilot.PRD.NonGoals,
		"decision.mvp":        copilot.Decision.MVP,
		"decision.outOfScope": copilot.Decision.OutOfScope,
	}
}

func scopeDetailSetList(copilot *DeliveryCopilot, target string, items []string) {
	switch target {
	case "goals":
		copilot.PRD.Goals = items
	case "nonGoals":
		copilot.PRD.NonGoals = items
	case "decision.mvp":
		copilot.Decision.MVP = items
	case "decision.outOfScope":
		copilot.Decision.OutOfScope = items
	}
}

func scopeDetailValidate(proposal PRDCoEditProposal, copilot DeliveryCopilot) PRDCoEditProposal {
	return validateCoEditListReplacements(sanitizePRDCoEditProposal(proposal, "产品负责人"), copilot)
}

func TestScopeDetailListReplacementJSONAndLongText(t *testing.T) {
	for target := range scopeDetailLists(scopeDetailCopilotFixture()) {
		t.Run(target, func(t *testing.T) {
			copilot := scopeDetailCopilotFixture()
			before := []string{"  " + strings.Repeat("原始范围条件", 150) + "；保留尾部验收边界  ", "第二条原始基线"}
			after := []string{strings.Repeat("具体角色输入操作输出验收", 100) + "；末尾禁止静默发布"}
			scopeDetailSetList(&copilot, target, before)
			original := PRDCoEditProposal{Changes: []PRDCoEditChange{{Target: target, Operation: "replace", BeforeItems: before, AfterItems: after, EvidenceStatus: "assumption"}}}
			encoded, err := json.Marshal(original)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(encoded, []byte(`"beforeItems"`)) || !bytes.Contains(encoded, []byte(`"afterItems"`)) {
				t.Fatal("list replacement is missing its JSON array contract")
			}
			var decoded PRDCoEditProposal
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			got := scopeDetailValidate(decoded, copilot)
			if len(got.Changes) != 1 {
				t.Fatalf("valid detailed replacement rejected: %#v", got)
			}
			change := got.Changes[0]
			if change.Operation != "replace" || !reflect.DeepEqual(change.BeforeItems, before) || !reflect.DeepEqual(change.AfterItems, after) {
				t.Fatalf("detailed arrays were trimmed, shortened, or changed: %#v", change)
			}
		})
	}
}

func TestScopeDetailListReplacementRejectsStaleBaseline(t *testing.T) {
	copilot := scopeDetailCopilotFixture()
	before := copilot.PRD.Goals
	for _, test := range []struct {
		name   string
		before []string
	}{
		{"different contents", []string{"已经被别人修改的目标", before[1]}},
		{"different order", []string{before[1], before[0]}},
		{"partial baseline", before[:1]},
		{"different whitespace", []string{before[0] + " ", before[1]}},
	} {
		t.Run(test.name, func(t *testing.T) {
			proposal := PRDCoEditProposal{Changes: []PRDCoEditChange{{Target: "goals", Operation: "replace", BeforeItems: test.before, AfterItems: []string{"完整详细的新目标"}}}}
			if got := scopeDetailValidate(proposal, copilot); len(got.Changes) != 0 {
				t.Fatalf("stale or incomplete baseline accepted: %#v", got.Changes)
			}
		})
	}
}

func TestScopeDetailListReplacementRejectsEmptyAndOversizedArrays(t *testing.T) {
	for _, test := range []struct {
		name   string
		before []string
		after  []string
	}{
		{"nil before", nil, []string{"详细目标"}},
		{"nil after", []string{"原目标"}, nil},
		{"empty before", []string{}, []string{"详细目标"}},
		{"empty after", []string{"原目标"}, []string{}},
		{"blank before item", []string{"原目标", " "}, []string{"详细目标"}},
		{"blank after item", []string{"原目标"}, []string{"详细目标", "\t"}},
		{"too many before", strings.Split(strings.Repeat("原目标,", 12)+"末条", ","), []string{"详细目标"}},
		{"too many after", []string{"原目标"}, strings.Split(strings.Repeat("详细目标,", 12)+"末条", ",")},
		{"oversized before", []string{strings.Repeat("原", 1601)}, []string{"详细目标"}},
		{"oversized after", []string{"原目标"}, []string{strings.Repeat("新", 1601)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			copilot := scopeDetailCopilotFixture()
			copilot.PRD.Goals = test.before
			proposal := PRDCoEditProposal{Changes: []PRDCoEditChange{{Target: "goals", Operation: "replace", BeforeItems: test.before, AfterItems: test.after, After: "不应以此字符串绕过无效数组"}}}
			if got := scopeDetailValidate(proposal, copilot); len(got.Changes) != 0 {
				t.Fatalf("unsafe replace accepted: %#v", got.Changes)
			}
		})
	}
	var nullItems PRDCoEditProposal
	if err := json.Unmarshal([]byte(`{"changes":[{"target":"goals","operation":"replace","beforeItems":["原目标"],"afterItems":[null]}]}`), &nullItems); err != nil {
		t.Fatal(err)
	}
	copilot := scopeDetailCopilotFixture()
	copilot.PRD.Goals = []string{"原目标"}
	if got := scopeDetailValidate(nullItems, copilot); len(got.Changes) != 0 {
		t.Fatal("JSON null array member must not delete a list silently")
	}
}

func TestScopeDetailDuplicateReplaceAndLegacyChanges(t *testing.T) {
	copilot := scopeDetailCopilotFixture()
	proposal := PRDCoEditProposal{Changes: []PRDCoEditChange{
		{Target: "goals", Operation: "replace", BeforeItems: copilot.PRD.Goals, AfterItems: []string{"首个详细目标"}},
		{Target: "goals", Operation: "replace", BeforeItems: copilot.PRD.Goals, AfterItems: []string{"冲突的第二个目标"}},
		{Target: "nonGoals", Operation: "append", After: "不以系统生成的假设替代用户确认"},
		{Target: "background.currentSituation", Operation: "replace", Before: "旧现状", After: "盘点员通过纸单交班，系统覆盖情况仍需访谈核验"},
	}}
	got := scopeDetailValidate(proposal, copilot)
	if len(got.Changes) != 3 {
		t.Fatalf("expected one replacement and both legacy changes: %#v", got.Changes)
	}
	if got.Changes[0].AfterItems[0] != "首个详细目标" || got.Changes[1].Operation != "append" || got.Changes[1].After == "" || got.Changes[2].Target != "background.currentSituation" {
		t.Fatalf("valid legacy changes were altered or duplicate won: %#v", got.Changes)
	}
	for target := range scopeDetailLists(copilot) {
		one := PRDCoEditProposal{Changes: []PRDCoEditChange{{Target: target, Operation: "append", After: "保留兼容的追加条目"}}}
		if result := scopeDetailValidate(one, copilot); len(result.Changes) != 1 || result.Changes[0].After != "保留兼容的追加条目" {
			t.Errorf("legacy append broken for %s: %#v", target, result)
		}
	}
}

func TestScopeDetailListReplacementAllowsExactSizeBoundary(t *testing.T) {
	copilot := scopeDetailCopilotFixture()
	before, after := make([]string, 12), make([]string, 12)
	for i := range before {
		before[i], after[i] = strings.Repeat("原", 1600), strings.Repeat("新", 1600)
	}
	copilot.PRD.Goals = before
	proposal := PRDCoEditProposal{Changes: []PRDCoEditChange{{Target: "goals", Operation: "replace", BeforeItems: before, AfterItems: after}}}
	got := scopeDetailValidate(proposal, copilot)
	if len(got.Changes) != 1 || !reflect.DeepEqual(got.Changes[0].BeforeItems, before) || !reflect.DeepEqual(got.Changes[0].AfterItems, after) {
		t.Fatal("exact 12-item / 1600-rune valid contract should not be shortened")
	}
}

func TestScopeDetailUnavailableDoesNotInventCustomerFeedback(t *testing.T) {
	request := PRDCoEditRequest{Mode: "scope_detail", Actor: "产品负责人", Message: "细化范围与目标", Copilot: scopeDetailCopilotFixture()}
	got := coEditUnavailable(request, "模型返回不完整")
	if len(got.Proposal.Changes) != 0 || got.Warning != "模型返回不完整" {
		t.Fatalf("scope failure must leave the original draft unchanged: %#v", got)
	}
	request.Mode = ""
	legacy := coEditUnavailable(request, "模型不可用")
	if len(legacy.Proposal.Changes) != 1 || legacy.Proposal.Changes[0].Target != "discovery.userFeedback" || legacy.Proposal.Changes[0].EvidenceStatus != "human_input" {
		t.Fatal("legacy general coedit must retain its explicitly qualified human-input fallback")
	}
}

func TestScopeDetailPromptUsedByGenerationAndCoEdit(t *testing.T) {
	if strings.TrimSpace(productScopeDetailPrompt) == "" {
		t.Fatal("scope detail guidance is empty")
	}
	for name, prompt := range map[string]string{"blueprint core": productBlueprintCorePrompt, "coedit": prdCoEditPrompt} {
		if !strings.Contains(prompt, productScopeDetailPrompt) {
			t.Errorf("%s is not using the shared detailed scope contract", name)
		}
	}
}

func TestScopeDetailCoEditInputKeepsFullBaselineAndProductContext(t *testing.T) {
	copilot := scopeDetailCopilotFixture()
	for target := range scopeDetailLists(copilot) {
		scopeDetailSetList(&copilot, target, []string{target + "：" + strings.Repeat("输入输出边界条件", 95) + "；独特末尾" + target})
	}
	input := prdCoEditInput(PRDCoEditRequest{Copilot: copilot, Actor: "产品负责人", Message: "请具体展开当前MVP及目标"})
	for target, items := range scopeDetailLists(copilot) {
		if !strings.Contains(input, items[0]) {
			t.Errorf("%s baseline was truncated; model cannot produce safe beforeItems", target)
		}
	}
	for _, want := range []string{"差异闭环率", "已由班组长确认的差异单数/发现的差异单数", "试点前记录七天基线", "目标假设：减少未确认差异", "两次交班周期", "仓库班组长", "夜班盘点员", "临期食品交班盘点", "收到临期批次清单", "REQ-W01", copilot.PRD.Requirements[0].Acceptance[0]} {
		if !strings.Contains(input, want) {
			t.Errorf("coedit input lost contextual detail %q", want)
		}
	}
}

func TestScopeDetailBlueprintRoundTripDeliveryAndDOCX(t *testing.T) {
	blueprint := ProductBlueprint{ProductName: "仓库临期盘点助手", OneSentence: "用批次清单交接账实差异"}
	details := []string{
		"目标：" + strings.Repeat("盘点员录入实盘数后能够识别批次差异，班组长逐项确认并保留处理说明；", 24) + "目标唯一尾部",
		"非目标：不自动推断采购决策；" + strings.Repeat("不能将待核验批次视为实际损耗；", 24) + "非目标唯一尾部",
		"MVP：导入批次清单，录入实盘数并导出差异单；" + strings.Repeat("输入包含批次号、库位和数量；错误行提示并保留原文件；", 24) + "MVP唯一尾部",
		"非范围：不自动回写ERP；" + strings.Repeat("接口权限未确认时由授权人员下载差异单人工处理；", 24) + "非范围唯一尾部",
	}
	blueprint.Goals, blueprint.NonGoals, blueprint.MVP, blueprint.OutOfScope = []string{details[0]}, []string{details[1]}, []string{details[2]}, []string{details[3]}
	encoded, err := json.Marshal(blueprint)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeProductBlueprint(string(encoded))
	if err != nil {
		t.Fatal(err)
	}
	normalizeProductBlueprint(&decoded, nil)
	result := ResearchResult{RunID: "scope-detail-test", Question: "为中国仓库的临期食品设计盘点交班工具", ProductBlueprint: &decoded}
	copilot, err := (&KnowledgeStore{}).BuildDeliveryCopilotForResearch(result)
	if err != nil {
		t.Fatal(err)
	}
	for target, got := range scopeDetailLists(copilot) {
		want := map[string][]string{"goals": blueprint.Goals, "nonGoals": blueprint.NonGoals, "decision.mvp": blueprint.MVP, "decision.outOfScope": blueprint.OutOfScope}[target]
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s changed across blueprint/PRD handoff", target)
		}
	}
	docx, err := buildDeliveryPRDDOCX(copilot)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range archive.File {
		if file.Name != "word/document.xml" {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(stream)
		stream.Close()
		if err != nil {
			t.Fatal(err)
		}
		text := scenarioExportXMLText(t, string(body))
		for _, detail := range details {
			if !strings.Contains(text, detail) {
				t.Errorf("DOCX shortened a scope/goal paragraph ending in %q", []rune(detail)[len([]rune(detail))-12:])
			}
		}
		return
	}
	t.Fatal("DOCX missing word/document.xml")
}
