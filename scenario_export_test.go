package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestDeliveryScenarioDOCXPreservesDetailsAndEvidence(t *testing.T) {
	var scenario ProductScenario
	err := json.Unmarshal([]byte(`{
		"id":"SC-01","name":"短视频多平台改编 <审核>","actor":"自媒体内容编辑",
		"basis":"research_hypothesis","context":"中国三人内容团队；原始素材已授权",
		"trigger":"品牌确认选题后，编辑需在当天交付两个平台版本",
		"inputs":["品牌确认单","已授权横版素材"," "],
		"currentWorkflow":["下载经确认的素材","在剪辑工具中逐一调整画幅"],
		"currentGap":"确认单与视频版本分离，版本状态需人工核对",
		"impact":"返工耗时尚未实测，需记录逐次确认等待时长",
		"proposedWorkflow":["绑定确认单和素材版本","生成平台改编草稿后交编辑审核"],
		"outcome":"输出可追溯确认单的两个待发布版本，不自动发布",
		"acceptance":["Given 原片已确认，When 改编完成，Then 两个平台版本都保留原片版本标识"],
		"exceptions":["素材授权过期时停止改编并通知编辑"],
		"validation":"邀请两名编辑回放真实任务；先测确认耗时基线再确定目标",
		"evidenceLinks":[{"id":"E2","field":"currentWorkflow","quote":"编辑使用 <工具> & 人工调整画幅。","explanation":"仅支持存在画幅调整操作，不证明团队返工率。","sourceTitle":"编辑工作流研究","url":"https://example.test/docs?a=1&b=2","publishedAt":"2026-09-30"}]
	}`), &scenario)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewKnowledgeStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.SeedAIAgentDemo(); err != nil {
		t.Fatal(err)
	}
	copilot, err := store.BuildDeliveryCopilot()
	if err != nil {
		t.Fatal(err)
	}
	copilot.Discovery.Scenarios = []ProductScenario{scenario}
	data, err := buildDeliveryPRDDOCX(copilot)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document string
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
		document = string(body)
	}
	if document == "" {
		t.Fatal("missing document XML")
	}
	text := scenarioExportXMLText(t, document)
	for _, want := range []string{
		"详细业务场景", "SC-01 短视频多平台改编 <审核>", "自媒体内容编辑",
		"研究支持的场景假设：尚未完成真实用户验证", scenario.Context, scenario.Trigger,
		scenario.Inputs[0], scenario.Inputs[1], scenario.CurrentWorkflow[0], scenario.CurrentWorkflow[1],
		scenario.CurrentGap, scenario.Impact, scenario.ProposedWorkflow[0], scenario.ProposedWorkflow[1],
		scenario.Outcome, scenario.Acceptance[0], scenario.Exceptions[0], scenario.Validation,
		"拟议工作流（方案）", "预期结果（拟议）", "验收条件（拟议）", "支持字段：现有流程",
		"E2 编辑工作流研究", "编辑使用 <工具> & 人工调整画幅。", "仅支持存在画幅调整操作，不证明团队返工率。",
		"https://example.test/docs?a=1&b=2", "发布时间：2026-09-30",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("export omitted %q", want)
		}
	}
	if strings.Contains(document, "<审核>") || strings.Contains(document, "<工具>") {
		t.Fatal("unescaped user or evidence XML")
	}
	if !(strings.Index(text, "3 用户与场景") < strings.Index(text, "详细业务场景") && strings.Index(text, "详细业务场景") < strings.Index(text, "4 决策与范围")) {
		t.Fatal("detailed scenarios not placed in user/scenario section")
	}
}

func TestDeliveryScenarioDOCXLegacyAndEmptyCompatibility(t *testing.T) {
	if got := deliveryScenarioDocx(nil); got != "" {
		t.Fatalf("nil scenarios produced output: %q", got)
	}
	if got := deliveryScenarioDocx([]ProductScenario{{}}); got != "" {
		t.Fatalf("empty scenario produced empty headings: %q", got)
	}
	legacy := ProductScenario{
		Name: "旧场景", Actor: "编辑", Trigger: "收到修改意见", CurrentGap: "需人工比对", Outcome: "输出确认稿",
		Evidence: []EvidenceRef{{ID: "E1", Label: "旧来源"}},
	}
	fragment := deliveryScenarioDocx([]ProductScenario{legacy})
	text := scenarioExportXMLText(t, `<root xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`+fragment+`</root>`)
	for _, want := range []string{"旧场景", "业务角色：编辑", "具体触发：收到修改意见", "业务缺口（待验证）：需人工比对", "预期结果（拟议）：输出确认稿", "历史场景：证据关联与方案有效性待核验", "历史引用（尚未建立字段级支持关系）：旧来源 (E1)"} {
		if !strings.Contains(text, want) {
			t.Errorf("legacy export omitted %q", want)
		}
	}
	for _, absent := range []string{"输入与前置资料", "拟议工作流（方案）", "验收条件（拟议）", "验证动作：", "字段级证据与适用范围", "原始网址："} {
		if strings.Contains(text, absent) {
			t.Errorf("empty legacy field rendered: %q", absent)
		}
	}
	if strings.Contains(fragment, `<w:t xml:space="preserve"></w:t>`) {
		t.Fatal("legacy export contains empty paragraphs")
	}
}

func TestDeliveryScenarioDOCXBasesStayQualified(t *testing.T) {
	for _, test := range []struct{ basis, want string }{
		{"unverified_hypothesis", "未验证场景假设：不可作为既定需求"},
		{"needs_research", "场景证据不足：需继续调研"},
	} {
		var scenario ProductScenario
		if err := json.Unmarshal([]byte(`{"name":"待研究","basis":"`+test.basis+`"}`), &scenario); err != nil {
			t.Fatal(err)
		}
		if got := deliveryScenarioDocx([]ProductScenario{scenario}); !strings.Contains(got, test.want) {
			t.Errorf("%s missing qualification", test.basis)
		}
	}
}

func scenarioExportXMLText(t *testing.T, document string) string {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(document))
	var text strings.Builder
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("invalid Word XML: %v", err)
		}
		if chars, ok := token.(xml.CharData); ok {
			text.Write(chars)
		}
	}
	return text.String()
}
