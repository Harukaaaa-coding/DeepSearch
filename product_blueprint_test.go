package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDecodeProductBlueprintJSONRejectsTruncatedOutput(t *testing.T) {
	var core productBlueprintCore
	err := decodeProductBlueprintJSON(`{"productName":"中国 Agent 工作台","goals":["验证"]`, &core)
	if err == nil {
		t.Fatal("expected truncated JSON to be rejected")
	}
}

func TestValidateProductBlueprintRejectsPartialBlueprint(t *testing.T) {
	err := validateProductBlueprint(ProductBlueprint{ProductName: "中国互联网 Agent 工作台", OneSentence: "测试"})
	if err == nil {
		t.Fatal("expected an incomplete blueprint to be rejected")
	}
}

func TestProductManagerToolkitReviewSurfacesGaps(t *testing.T) {
	blueprint := ProductBlueprint{Requirements: []ExecutionRequirement{{ID: "REQ-001", Acceptance: []string{"Given x When y Then z"}, Tracking: []string{"task_item_create | 服务端成功 | task_id | 后端"}}}}
	applyProductManagerToolkitReview(&blueprint)
	if len(blueprint.OpenQuestions) == 0 {
		t.Fatal("expected missing exception coverage to be visible for review")
	}
}

func TestProductBlueprintInputIncludesLaterEvidenceCompetitionAndCoverage(t *testing.T) {
	result := ResearchResult{Question: "MCN 内容审核产品", Coverage: ResearchCoverage{Status: "partial", StopReason: "budget_exhausted", Gaps: []string{"需要客户切换成本证据"}}}
	for index := 1; index <= 12; index++ {
		result.Evidence = append(result.Evidence, Evidence{ID: fmt.Sprintf("E%d", index), SourceID: fmt.Sprintf("S%d", index), Quote: "对应来源的具体产品事实"})
	}
	result.Competition = &CompetitorAnalysis{Status: "insufficient_evidence", Competitors: []CompetitorProfile{{Name: "内容审校助手", Positioning: "品牌内容发布前审校", Evidence: []EvidenceRef{{ID: "E12"}}}}, OpenQuestions: []string{"核验团队版定价"}}
	input := productBlueprintInput(result)
	for _, want := range []string{"[E12]", "内容审校助手", "品牌内容发布前审校", "需要客户切换成本证据", "budget_exhausted", "核验团队版定价"} {
		if !strings.Contains(input, want) {
			t.Errorf("blueprint input missing %q", want)
		}
	}
}

func TestBlueprintInputEvidencePrioritizesCitationsAndSourceDiversity(t *testing.T) {
	result := ResearchResult{Evidence: []Evidence{
		{ID: "E1", SourceID: "S1"}, {ID: "E2", SourceID: "S1"}, {ID: "E3", SourceID: "S1"},
		{ID: "E4", SourceID: "S2"}, {ID: "E5", SourceID: "S3"},
	}, Competition: &CompetitorAnalysis{Competitors: []CompetitorProfile{{Evidence: []EvidenceRef{{ID: "E5"}, {ID: "E999"}}}}}}
	selected := blueprintInputEvidence(result, 3)
	if len(selected) != 3 || selected[0].ID != "E5" || selected[1].ID != "E1" || selected[2].ID != "E4" {
		t.Fatalf("expected citation then distinct sources, got %#v", selected)
	}
}

func TestProductBlueprintInputRespectsContextBudget(t *testing.T) {
	result := ResearchResult{Question: strings.Repeat("问题", 1000), Report: Report{Analysis: strings.Repeat("研究材料", 4000)}}
	for index := 1; index <= 100; index++ {
		result.Evidence = append(result.Evidence, Evidence{ID: fmt.Sprintf("E%d", index), SourceID: fmt.Sprintf("S%d", index), SourceTitle: strings.Repeat("长来源标题", 100), URL: "https://example.test/" + strings.Repeat("path", 100), Quote: strings.Repeat("有效证据文本", 500)})
	}
	result.Competition = &CompetitorAnalysis{Competitors: []CompetitorProfile{{Name: "后发现的重要竞品", Evidence: []EvidenceRef{{ID: "E90"}}}}}
	input := productBlueprintInput(result)
	if count := len([]rune(input)); count > 20000 {
		t.Fatalf("context budget exceeded: %d characters", count)
	}
	if !strings.Contains(input, "[E90]") {
		t.Fatal("late competitor evidence was lost from bounded context")
	}
}

func TestProductBlueprintDeliveryInputPreservesCoreDesign(t *testing.T) {
	acceptance := "Given 审核员导入包含三个品牌禁用词的字幕文件且没有发布权限 When 发起审校并尝试直接发布 Then 系统逐条显示命中的时间戳、禁用词与建议修改，保留未修改的原文，并拒绝发布操作；只有指定复核员确认全部问题处理完成后才允许进入待发布状态。"
	core := productBlueprintCore{ProductName: "内容审校助手", MVP: []string{"人工复核后发布"}, Requirements: []ExecutionRequirement{{ID: "REQ-007", Title: "品牌词审核", MainFlow: []string{"上传视频字幕", "标记品牌禁用词"}, Acceptance: []string{acceptance}}}}
	input := productBlueprintDeliveryInput("原始证据包", core)
	start := strings.Index(input, "{\"productName\"")
	if start < 0 {
		t.Fatal("delivery input has no core design JSON")
	}
	var decoded productBlueprintCore
	if err := json.Unmarshal([]byte(input[start:]), &decoded); err != nil {
		t.Fatalf("core design was truncated: %v", err)
	}
	if decoded.Requirements[0].ID != "REQ-007" || decoded.MVP[0] != "人工复核后发布" {
		t.Fatalf("delivery input changed the product baseline: %#v", decoded)
	}
	if decoded.Requirements[0].Acceptance[0] != acceptance {
		t.Fatal("detailed input, permissions, and expected outcomes were shortened in stage handoff")
	}
}

func TestBlueprintTraceabilityReviewSurfacesUnsupportedP0AndDisconnectedSchema(t *testing.T) {
	result := ResearchResult{Evidence: []Evidence{{ID: "E1"}}, Coverage: ResearchCoverage{Gaps: []string{"目标客户访谈不足"}}}
	blueprint := ProductBlueprint{
		Requirements:   []ExecutionRequirement{{ID: "REQ-001", Priority: "P0", Evidence: []EvidenceRef{{ID: "E999"}}}},
		DataModel:      []DataEntity{{Name: "review_tasks", PrimaryKey: "task_id"}},
		DataDictionary: []DataDictionaryItem{{Entity: "unrelated_accounts", Field: "id", Type: "string", Required: "yes"}},
		TestPlan:       TestPlan{Cases: []TestCase{{Title: "[REQ-0010] 不同需求的测试"}}},
		SuccessMetrics: []SuccessMetric{{Name: "审核时长"}},
	}
	normalizeProductBlueprint(&blueprint, result.Evidence)
	applyProductBlueprintTraceabilityReview(&blueprint, result)
	questions := strings.Join(blueprint.OpenQuestions, "\n")
	for _, want := range []string{"目标客户访谈不足", "P0 证据缺口：REQ-001", "测试追溯：REQ-001", "指向未定义的实体", "主键缺少字典定义", "指标口径：审核时长"} {
		if !strings.Contains(questions, want) {
			t.Errorf("missing actionable gap %q in %s", want, questions)
		}
	}
}

func TestBlueprintTraceabilityReviewAcceptsLinkedDesign(t *testing.T) {
	blueprint := ProductBlueprint{
		Requirements:   []ExecutionRequirement{{ID: "REQ-001", Priority: "P0", UserStory: "审核员检查品牌词", MainFlow: []string{"提交审校"}, BusinessRules: []string{"复核通过后才允许发布"}, Permissions: []string{"审核员"}, Evidence: []EvidenceRef{{ID: "E1", Type: "public_research"}}}},
		DataModel:      []DataEntity{{Name: "review_tasks", PrimaryKey: "task_id"}},
		DataDictionary: []DataDictionaryItem{{Entity: "review_tasks", Field: "task_id", Type: "string", Required: "yes"}},
		TestPlan:       TestPlan{Cases: []TestCase{{Title: "[REQ-001] 禁用词阻止发布"}}},
		SuccessMetrics: []SuccessMetric{{Name: "审核时长", Definition: "提交至复核通过的时长中位数", Baseline: "试点前采集一周", Target: "目标假设：缩短20%", Window: "试点两周", Owner: "内容运营"}},
	}
	applyProductBlueprintTraceabilityReview(&blueprint, ResearchResult{})
	if len(blueprint.OpenQuestions) != 0 {
		t.Fatalf("linked design incorrectly flagged: %v", blueprint.OpenQuestions)
	}
}

func TestProductManagerReviewRejectsPlaceholderAcceptanceAndTracking(t *testing.T) {
	blueprint := ProductBlueprint{Requirements: []ExecutionRequirement{{ID: "REQ-001", Priority: "P0", Acceptance: []string{"通过验收"}, Tracking: []string{"event_name"}}}}
	applyProductManagerToolkitReview(&blueprint)
	questions := strings.Join(blueprint.OpenQuestions, "\n")
	if !strings.Contains(questions, "Given/When/Then") || !strings.Contains(questions, "事件、触发时机、属性或上报端") {
		t.Fatalf("placeholder checks were accepted: %v", blueprint.OpenQuestions)
	}
}

func TestBlueprintMetricNumbersRequireEvidenceOrQualification(t *testing.T) {
	evidence := []Evidence{{ID: "E1"}}
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"unsupported baseline", "当前人工审核耗时42分钟", true},
		{"unsupported target", "首月付费转化率达到20%", true},
		{"invented citation", "人工审核耗时42分钟 [E999]", true},
		{"unknown baseline with collection plan", "未知；上线前采集7天审核耗时", false},
		{"testable target hypothesis", "目标假设：中位审核耗时降低20%；试点对照人工流程验证", false},
		{"cited observation", "公开案例记录审核耗时42分钟 [E1]", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := blueprintUnqualifiedMetricNumber(test.value, evidence); got != test.want {
				t.Fatalf("unqualified=%v, want %v for %q", got, test.want, test.value)
			}
		})
	}
}

func blueprintCoreStageFixture() productBlueprintCore {
	return productBlueprintCore{
		ProductName: "内容审校助手", OneSentence: "发布前复核品牌禁用词",
		Goals: []string{"目标一", "目标二"}, NonGoals: []string{"非目标一", "非目标二"},
		SuccessMetrics: make([]SuccessMetric, 3), Users: make([]PRDUserScenario, 1), Options: make([]DecisionOption, 2), MVP: []string{"品牌词审校"},
		UserFlow: []string{"上传字幕", "人工复核"}, SystemFlow: []string{"待审校→处理中", "处理中→待复核"},
		SOP: make([]SOPStep, 4), Permissions: make([]PermissionRule, 3), Requirements: make([]ExecutionRequirement, 4),
		Prototype: ProductPrototype{Screens: make([]PrototypeScreen, 3)},
	}
}

func blueprintDeliveryStageFixture() productBlueprintDelivery {
	return productBlueprintDelivery{
		DataModel: make([]DataEntity, 4), DataDictionary: make([]DataDictionaryItem, 6), APIContracts: make([]APIContract, 3),
		Dependencies: make([]DeliveryDependency, 3), NonFunctional: make([]NonFunctionalRequirement, 4),
		TestPlan:     TestPlan{Cases: make([]TestCase, 4), DataSets: make([]TestDataSet, 2)},
		DeliveryPlan: DeliveryPlan{Phases: make([]DeliveryPhase, 5)},
		Monitoring:   []string{"监控一", "监控二", "监控三"}, Rollback: []string{"触发一", "触发二"},
	}
}

func TestBlueprintStageValidationKeepsStagesIndependent(t *testing.T) {
	coreJSON, _ := json.Marshal(blueprintCoreStageFixture())
	var core productBlueprintCore
	if err := decodeValidatedProductBlueprintPart(string(coreJSON), &core); err != nil {
		t.Fatalf("valid core should not require delivery fields: %v", err)
	}
	deliveryJSON, _ := json.Marshal(blueprintDeliveryStageFixture())
	var delivery productBlueprintDelivery
	if err := decodeValidatedProductBlueprintPart(string(deliveryJSON), &delivery); err != nil {
		t.Fatalf("valid delivery should not require core fields: %v", err)
	}
	var blueprint ProductBlueprint
	core.applyTo(&blueprint)
	delivery.applyTo(&blueprint)
	if err := validateProductBlueprint(blueprint); err != nil {
		t.Fatalf("stage requirements drifted from final validation: %v", err)
	}
}

func TestBlueprintStageValidationReportsExactMissingCounts(t *testing.T) {
	core := blueprintCoreStageFixture()
	core.UserFlow = core.UserFlow[:1]
	core.SystemFlow = nil
	core.SOP = core.SOP[:3]
	core.Permissions = core.Permissions[:2]
	encoded, _ := json.Marshal(core)
	var output productBlueprintCore
	err := decodeValidatedProductBlueprintPart(string(encoded), &output)
	if err == nil {
		t.Fatal("incomplete core accepted")
	}
	for _, want := range []string{"userFlow 当前 1 项，至少需要 2 项", "systemFlow 当前 0 项，至少需要 2 项", "sop 当前 3 项，至少需要 4 项", "permissions 当前 2 项，至少需要 3 项"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing repair detail %q in %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "dataModel") || output.ProductName != "" {
		t.Fatalf("stage validation mixed stages or exposed invalid output: %v / %#v", err, output)
	}
	delivery := blueprintDeliveryStageFixture()
	delivery.DataDictionary = delivery.DataDictionary[:5]
	delivery.DeliveryPlan.Phases = delivery.DeliveryPlan.Phases[:4]
	encoded, _ = json.Marshal(delivery)
	var deliveryOutput productBlueprintDelivery
	err = decodeValidatedProductBlueprintPart(string(encoded), &deliveryOutput)
	if err == nil || !strings.Contains(err.Error(), "dataDictionary 当前 5 项，至少需要 6 项") || !strings.Contains(err.Error(), "deliveryPlan.phases 当前 4 项，至少需要 5 项") {
		t.Fatalf("delivery limits were not enforced precisely: %v", err)
	}
}

func TestBlueprintStageValidationDoesNotInheritMissingRetryFields(t *testing.T) {
	previous := blueprintCoreStageFixture()
	err := decodeValidatedProductBlueprintPart(`{"productName":"新回复","oneSentence":"省略其他字段"}`, &previous)
	if err == nil || !strings.Contains(err.Error(), "permissions 当前 0 项") {
		t.Fatalf("retry incorrectly inherited old fields: %v", err)
	}
	if previous.ProductName != "内容审校助手" {
		t.Fatal("failed candidate was committed to the output")
	}
}

func TestBlueprintStageRequestRepairsValidJSONWithMissingStructure(t *testing.T) {
	valid := blueprintCoreStageFixture()
	broken := valid
	broken.UserFlow = broken.UserFlow[:1]
	broken.Permissions = broken.Permissions[:2]
	var requests atomic.Int32
	instructions := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || len(payload.Messages) == 0 {
			http.Error(w, "invalid mock request", http.StatusBadRequest)
			return
		}
		instructions <- payload.Messages[0].Content
		candidate := valid
		if requests.Add(1) == 1 {
			candidate = broken
		}
		encoded, _ := json.Marshal(candidate)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": string(encoded)}}}})
	}))
	defer server.Close()
	provider := reportProvider{Endpoint: server.URL, APIKey: "offline-test", Model: "offline-test", Protocol: "chat_completions"}
	var output productBlueprintCore
	err := (&ResearchEngine{}).requestProductBlueprintPart(context.Background(), provider, "生成本阶段 JSON", "离线证据", "产品与需求", 8200, &output)
	if err != nil {
		t.Fatalf("bounded repair failed: %v", err)
	}
	if requests.Load() != 2 || len(output.UserFlow) != 2 || len(output.Permissions) != 3 {
		t.Fatalf("expected exactly one repair, requests=%d output=%#v", requests.Load(), output)
	}
	<-instructions
	repair := <-instructions
	if !strings.Contains(repair, "userFlow 当前 1 项，至少需要 2 项") || !strings.Contains(repair, "permissions 当前 2 项，至少需要 3 项") {
		t.Fatalf("retry prompt lacks exact repair instructions: %s", repair)
	}
}
