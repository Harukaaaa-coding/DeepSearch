package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestKnowledgeStoreIngestsRawFileChunksAndSearches(t *testing.T) {
	store, err := NewKnowledgeStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	document, err := store.Ingest("refund-interview.md", []byte("# Refund interview\n\n用户不知道退款什么时候到账，并且三次联系客服查询退款进度。"), KnowledgeMetadata{Title: "退款用户访谈", DocumentType: "user_interview", Department: "product", Version: "2", Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	status := store.Status()
	if document.ChunkCount == 0 || status.Documents != 1 || status.Chunks != document.ChunkCount {
		t.Fatalf("unexpected stored document/status: %#v / %#v", document, status)
	}
	hits := store.Search("退款进度不清楚", "active", 5)
	if len(hits) == 0 || hits[0].DocumentID != document.ID {
		t.Fatalf("expected matching internal evidence, got %#v", hits)
	}
	for _, table := range []string{"documents", "document_chunks", "document_chunks_fts", "support_tickets", "interviews", "insights", "product_requirements", "research_notes", "structured_datasets", "product_initiatives", "prd_versions", "prd_requirements", "evidence_links", "prd_reviews", "prd_approvals", "agent_runs", "agent_events", "agent_memory_summaries"} {
		var name string
		if err := store.db.QueryRow(`SELECT name FROM sqlite_master WHERE name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("expected local table/index %s: %v", table, err)
		}
	}
}

func TestKnowledgeStoreSeedsAIAgentDemoIdempotently(t *testing.T) {
	store, err := NewKnowledgeStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	result, err := store.SeedAIAgentDemo()
	if err != nil {
		t.Fatal(err)
	}
	if !result.Created || result.Status.Documents != 7 || result.Status.Interviews != 3 || result.Status.SupportTickets != 8 || result.Status.Insights != 4 || result.Status.ProductRequirements != 1 || result.Status.ResearchNotes != 1 || result.Status.StructuredDatasets != 1 {
		t.Fatalf("unexpected demo seed result: %#v", result)
	}
	hits := store.Search("证据引用", "active", 8)
	if len(hits) == 0 {
		t.Fatal("expected seeded evidence to be searchable")
	}
	again, err := store.SeedAIAgentDemo()
	if err != nil {
		t.Fatal(err)
	}
	if again.Created || again.Status != result.Status {
		t.Fatalf("demo seed should be idempotent: %#v", again)
	}
}

func TestHistorySearchAndProductAnalysisUseStructuredTables(t *testing.T) {
	store, err := NewKnowledgeStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.SeedAIAgentDemo(); err != nil {
		t.Fatal(err)
	}
	history, err := store.SearchHistory("权限", "all", 20)
	if err != nil {
		t.Fatal(err)
	}
	if history.Total == 0 {
		t.Fatal("expected historical records across local tables")
	}
	analysis, err := store.BuildProductAnalysis()
	if err != nil {
		t.Fatal(err)
	}
	if len(analysis.UserFeedback) != 3 || len(analysis.DemandResearch) != 3 || len(analysis.Scenarios) != 3 || len(analysis.PainPoints) != 4 || len(analysis.FeaturePlan) != 4 || len(analysis.PRD.FunctionalRequirements) == 0 {
		t.Fatalf("unexpected product analysis: %#v", analysis)
	}
	if len(analysis.UserFeedback[0].Evidence) == 0 || analysis.UserFeedback[0].Evidence[0].ID == "" {
		t.Fatalf("product analysis is missing traceable historical evidence: %#v", analysis.UserFeedback[0])
	}
}

func TestDeliveryCopilotContainsEnterprisePRDAndReview(t *testing.T) {
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
	if len(copilot.Workflow) != 7 || len(copilot.PRD.Requirements) != 4 || len(copilot.PRD.Permissions) == 0 || len(copilot.PRD.Prototype.Screens) != 3 || len(copilot.PRD.SuccessMetrics) != 4 || len(copilot.PRD.DeliveryPlan.Phases) != 6 || len(copilot.Solution.APIs) == 0 || len(copilot.Solution.DataModel) != 5 || len(copilot.Solution.DataDictionary) != 8 || len(copilot.Solution.APIContracts) != 4 || len(copilot.Solution.Dependencies) != 3 || len(copilot.Visuals.Flow) != 7 || len(copilot.Visuals.ProductFlow) != 6 || len(copilot.Visuals.SolutionFlow) != 5 || len(copilot.Visuals.SOP) != 6 || len(copilot.Visuals.TestPlan.Roles) != 5 || len(copilot.Visuals.TestPlan.Cases) != 5 || len(copilot.Visuals.TestPlan.DataSets) != 4 || len(copilot.Readiness.Checks) == 0 || copilot.Readiness.Score <= 0 {
		t.Fatalf("incomplete delivery copilot: %#v", copilot)
	}
	encoded, err := json.Marshal(copilot)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"userFlow"`, `"businessRules"`, `"acceptance"`, `"prototype"`, `"successMetrics"`, `"deliveryPlan"`, `"dataModel"`, `"dataDictionary"`, `"apiContracts"`, `"dependencies"`, `"flow"`, `"productFlow"`, `"solutionFlow"`, `"sop"`, `"testPlan"`, `"dataSets"`, `"nextActions"`} {
		if !strings.Contains(string(encoded), key) {
			t.Fatalf("missing JSON key %s in %s", key, encoded)
		}
	}
}

func TestResearchBuildsIndependentEvidenceBoundPRD(t *testing.T) {
	store, err := NewKnowledgeStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	result := ResearchResult{
		RunID:    "research-run-001",
		Question: "企业级 AI Agent 的客服质检创业机会应该先验证什么？",
		Stats:    Stats{Discovered: 7, EvidenceCount: 3, HighConfidence: 2, ConflictCount: 1},
		Report: Report{
			Analysis:    "公开资料表明客服质检的可追溯性与人工复核是一个待验证机会。",
			Conflicts:   []string{"不同市场规模统计口径不一致。"},
			Limitations: []string{"尚未完成目标客户访谈。"},
		},
		Evidence: []Evidence{
			{ID: "E1", SourceTitle: "行业研究", URL: "https://example.com/industry", Quote: "质检需要可追溯证据。", Score: 84, Confidence: "高"},
			{ID: "E2", SourceTitle: "合规说明", URL: "https://example.com/compliance", Quote: "人工复核不可省略。", Score: 81, Confidence: "高"},
			{ID: "E3", SourceTitle: "案例资料", URL: "https://example.com/case", Quote: "试点必须明确验收条件。", Score: 68, Confidence: "中"},
		},
	}

	copilot, err := store.BuildDeliveryCopilotForResearch(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(copilot.PRD.Info.PRDID, "RPRD-") || !strings.Contains(copilot.PRD.Info.Feature, "客服质检") || len(copilot.PRD.Requirements) != 4 || len(copilot.PRD.SuccessMetrics) != 4 || len(copilot.PRD.DeliveryPlan.Phases) != 6 {
		t.Fatalf("unexpected research-specific PRD: %#v", copilot.PRD)
	}
	if strings.Contains(copilot.PRD.Background.CurrentSituation, "本地历史资料") || copilot.PRD.Requirements[0].ID != "REQ-001" || copilot.PRD.Requirements[0].Evidence[0].ID != "E1" {
		t.Fatalf("research PRD should remain bound to the supplied run: %#v", copilot.PRD)
	}
	state, err := store.PersistDeliveryCopilot(copilot)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Persisted || state.InitiativeKey == deliveryInitiativeKey || state.Requirements != 4 || state.EvidenceLinks == 0 {
		t.Fatalf("research PRD should persist as an independent initiative: %#v", state)
	}
}
func TestModelProductBlueprintFillsResearchPRDFields(t *testing.T) {
	store, err := NewKnowledgeStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	result := ResearchResult{
		RunID:    "creator-blueprint-run",
		Question: "为自媒体创作者、MCN 运营和品牌审核方设计互联网 AI Agent 工作台",
		Report:   Report{Analysis: "内容生产流程需要研究、协作与审核闭环。"},
		Evidence: []Evidence{{ID: "E1", SourceTitle: "内容行业研究", URL: "https://example.com/content", Quote: "创作团队需要协作和审核。", Confidence: "中"}},
		ProductBlueprint: &ProductBlueprint{
			ProductName:    "CreatorOps AI 工作台",
			OneSentence:    "让创作者、运营和审核方在一个可追溯工作台协作完成选题、生产和复盘。",
			Goals:          []string{"缩短选题到发布协作链路", "保留审核和来源记录"},
			NonGoals:       []string{"不自动替代人工发布"},
			SuccessMetrics: []SuccessMetric{{Name: "选题流转完成率", Definition: "创建后完成审核的选题比例", Baseline: "待试点", Target: "待 Owner 确认", Window: "每周", Owner: "运营负责人"}},
			Users:          []PRDUserScenario{{Role: "自媒体创作者", Buyer: "个人/MCN", Scenario: "建立选题任务", Trigger: "发现内容机会", Goal: "完成内容 Brief", Evidence: []EvidenceRef{{ID: "E1"}}}, {Role: "MCN 运营", Buyer: "MCN", Scenario: "分派与审核", Trigger: "任务待审核", Goal: "控制产能", Evidence: []EvidenceRef{{ID: "E1"}}}},
			Options:        []DecisionOption{{Name: "通用聊天", Benefit: "快", Tradeoff: "不可协作", Verdict: "不推荐"}, {Name: "CreatorOps 工作台", Benefit: "可协作", Tradeoff: "需试点", Verdict: "推荐"}},
			Decision:       "先做带审核门的 CreatorOps MVP。",
			Reasoning:      "研究显示协作与审核需要进入同一工作流。",
			MVP:            []string{"选题任务", "内容 Brief", "审核队列"},
			OutOfScope:     []string{"自动发布到所有平台"},
			UserFlow:       []string{"创建选题", "补充 Brief", "提交审核", "发布复盘"},
			SystemFlow:     []string{"保存任务", "检索资料", "生成草案", "记录审核"},
			SOP:            []SOPStep{{Order: 1, Title: "确认选题", Owner: "运营", Action: "核验范围", Input: "候选选题", Output: "已批准任务", QualityGate: "来源与责任人完整"}},
			Permissions:    []PermissionRule{{Capability: "发布审核", User: "提交", CSAgent: "—", Admin: "批准"}},
			Requirements:   []ExecutionRequirement{{ID: "REQ-001", Title: "选题任务", Priority: "P0", UserStory: "作为创作者，我要创建选题任务。", Preconditions: []string{"已登录"}, MainFlow: []string{"创建", "分派"}, BusinessRules: []string{"必须指定负责人"}, Exceptions: []string{"资料不足时标记 Gap"}, Permissions: []string{"创作者编辑"}, Acceptance: []string{"Given 已创建 When 保存 Then 生成任务"}, Tracking: []string{"topic_created"}, Evidence: []EvidenceRef{{ID: "E1"}}}, {ID: "REQ-002", Title: "审核队列", Priority: "P0", UserStory: "作为运营，我要审核内容。", Preconditions: []string{"任务已提交"}, MainFlow: []string{"查看", "批准"}, BusinessRules: []string{"需记录审核意见"}, Exceptions: []string{"审核人缺席时升级"}, Permissions: []string{"运营审核"}, Acceptance: []string{"Given 待审核 When 批准 Then 记录意见"}, Tracking: []string{"reviewed"}, Evidence: []EvidenceRef{{ID: "E1"}}}, {ID: "REQ-003", Title: "内容复盘", Priority: "P1", UserStory: "作为运营，我要复盘内容。", Preconditions: []string{"内容已发布"}, MainFlow: []string{"导入", "标记"}, BusinessRules: []string{"数据来源可追溯"}, Exceptions: []string{"指标缺失时提示"}, Permissions: []string{"运营查看"}, Acceptance: []string{"Given 数据可用 When 复盘 Then 输出结论"}, Tracking: []string{"review_complete"}, Evidence: []EvidenceRef{{ID: "E1"}}}},
			Prototype:      ProductPrototype{Screens: []PrototypeScreen{{ID: "SCR-01", Name: "选题看板", Route: "/topics", Purpose: "管理选题", PrimaryAction: "新建选题", Regions: []PrototypeRegion{{Name: "任务列", Content: "状态", Behavior: "拖动"}}, States: []string{"空态", "待审核"}}}},
			DataModel:      []DataEntity{{Name: "Topic", Purpose: "选题", PrimaryKey: "topic_id", Relationships: []string{"Task"}, Retention: "一年", Access: "团队"}},
			DataDictionary: []DataDictionaryItem{{Entity: "Topic", Field: "title", Type: "string", Required: "是", Description: "选题标题", Example: "AI 工作台", Classification: "内部"}},
			APIContracts:   []APIContract{{Name: "创建选题", Method: "POST", Path: "/topics", Auth: "成员", Request: []string{"title"}, Response: []string{"topic_id"}, Failures: []string{"400"}}},
			Dependencies:   []DeliveryDependency{{Name: "内容平台数据", Owner: "运营", NeedBy: "试点前", Risk: "授权延迟", Fallback: "使用脱敏样例"}},
			NonFunctional:  []NonFunctionalRequirement{{Area: "权限", Requirement: "审核记录可审计", Verification: "权限测试", Status: "必须"}},
			TestPlan:       TestPlan{Strategy: "主流程与权限测试", Roles: []TestRole{{Role: "创作者", Owner: "试点用户", HowToTest: "创建任务", Exit: "成功保存"}}, Cases: []TestCase{{ID: "TC-001", Title: "创建选题", Level: "P0", Steps: []string{"创建"}, Expected: "保存成功"}}, DataSets: []TestDataSet{{Name: "脱敏选题", Purpose: "测试", Example: "10 条", Notice: "无个人信息"}}, ExitRules: []string{"P0 通过"}},
			DeliveryPlan:   DeliveryPlan{Estimate: "待试点评审", Assumptions: []string{"团队可用"}, Phases: []DeliveryPhase{{Window: "第 1 周", Stage: "设计", Owner: "产品", Outcome: "原型", EntryGate: "范围确认", ExitGate: "评审通过", DelayRisk: "用户不可用"}}},
			Tracking:       []string{"topic_created"}, TestFocus: []string{"审核权限"}, LaunchChecklist: []string{"Owner 确认"}, Monitoring: []string{"任务失败率"}, Rollback: []string{"关闭试点"}, OpenQuestions: []string{"首批试点是谁？"},
		},
	}
	copilot, err := store.BuildDeliveryCopilotForResearch(result)
	if err != nil {
		t.Fatal(err)
	}
	if copilot.PRD.Info.Product != "CreatorOps AI 工作台" || copilot.PRD.Requirements[0].Title != "选题任务" || copilot.PRD.Prototype.Screens[0].Name != "选题看板" || copilot.Solution.DataModel[0].Name != "Topic" || copilot.Visuals.TestPlan.Cases[0].ID != "TC-001" || copilot.PRD.DeliveryPlan.Phases[0].Stage != "设计" {
		t.Fatalf("model blueprint was not applied to PRD: %#v", copilot)
	}
}
func TestPersistDeliveryDraftStoresVersionedRequirementsEvidenceReviewsAndApprovals(t *testing.T) {
	store, err := NewKnowledgeStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.SeedAIAgentDemo(); err != nil {
		t.Fatal(err)
	}
	state, err := store.PersistDeliveryDraft()
	if err != nil {
		t.Fatal(err)
	}
	if !state.Persisted || state.Version != "0.1" || state.Status != "draft" || state.Requirements != 4 || state.EvidenceLinks == 0 || state.ReviewCount != 1 || len(state.Approvals) != 5 || state.PendingApprovals != 5 {
		t.Fatalf("unexpected persisted workflow state: %#v", state)
	}
	status := store.Status()
	if status.Initiatives != 1 || status.PRDVersions != 1 || status.PRDRequirements != 4 || status.EvidenceLinks == 0 || status.PRDReviews != 1 || status.PRDApprovals != 5 {
		t.Fatalf("unexpected delivery persistence counters: %#v", status)
	}
	if _, err := store.SubmitDeliveryForReview(); err == nil {
		t.Fatal("expected current demo draft to be blocked by the readiness gate")
	}
	updated, err := store.PersistDeliveryDraft()
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != "0.1" || updated.Requirements != 4 || updated.ReviewCount != 2 || updated.EvidenceLinks != state.EvidenceLinks {
		t.Fatalf("draft save should preserve versioned requirements/evidence and retain review history: %#v", updated)
	}
}

func TestDeliveryApprovalRequiresReviewAndCompletesVersion(t *testing.T) {
	store, err := NewKnowledgeStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.SeedAIAgentDemo(); err != nil {
		t.Fatal(err)
	}
	state, err := store.PersistDeliveryDraft()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.DecideDeliveryApproval("product_owner", "Demo Product Owner", "approved", "too early"); err == nil {
		t.Fatal("approval must not be possible before formal review")
	}
	if _, err := store.db.Exec(`UPDATE prd_reviews SET readiness_score=85, verdict='Ready for review' WHERE prd_version_id=?`, state.VersionID); err != nil {
		t.Fatal(err)
	}
	state, err = store.SubmitDeliveryForReview()
	if err != nil || state.Status != "in_review" {
		t.Fatalf("expected formal review: %#v / %v", state, err)
	}
	for _, role := range []string{"product_owner", "design_lead", "engineering_lead", "qa_lead", "security_lead"} {
		state, err = store.DecideDeliveryApproval(role, "Local reviewer", "approved", "approved in test")
		if err != nil {
			t.Fatalf("approval failed for %s: %v", role, err)
		}
	}
	if state.Status != "approved" || state.PendingApprovals != 0 {
		t.Fatalf("expected all approvals to complete the version: %#v", state)
	}
}

func TestAgentMemoryRetainsToolTraceAndCompressesOlderContext(t *testing.T) {
	store, err := NewKnowledgeStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for i := 0; i < 7; i++ {
		now := time.Now().UTC().Add(time.Duration(i) * time.Minute)
		longTitle := strings.Repeat("可追溯市场证据 ", 85)
		result := ResearchResult{
			RunID:       fmt.Sprintf("memory-run-%d", i),
			Question:    "AI Agent 市场证据与权限风险",
			StartedAt:   now,
			CompletedAt: now,
			Plan:        []PlanStep{{ID: "scope", Title: "问题定义", Status: "done"}, {ID: "read", Title: "阅读与提取", Status: "done"}},
			Stats:       Stats{Discovered: 5, Read: 3, EvidenceCount: 3, ConflictCount: i % 2},
			Sources: []Source{
				{Title: longTitle + "A", Query: "AI Agent 市场规模", Status: "已阅读"},
				{Title: longTitle + "B", Query: "AI Agent 风险监管", Status: "已阅读"},
				{Title: longTitle + "C", Query: "AI Agent 竞争产品", Status: "已阅读"},
			},
			Evidence: []Evidence{{ID: "E1", SourceTitle: longTitle + "A"}, {ID: "E2", SourceTitle: longTitle + "B"}, {ID: "E3", SourceTitle: longTitle + "C"}},
			Report:   Report{Title: "证据约束研究报告", Method: "仅使用当前证据包", ModelUsed: "test"},
		}
		if err := store.RecordResearchRun(result); err != nil {
			t.Fatal(err)
		}
	}
	policy := AgentMemoryPolicy{ContextWindowTokens: 2000, CompressAtPercent: 55, RetainRecentEvents: 4, SummaryTokenBudget: 300}
	context, err := store.buildAgentContext("AI Agent 市场证据与权限风险", policy)
	if err != nil {
		t.Fatal(err)
	}
	if len(context.RecentEvents) < 4 || !context.CompressionTriggered || !strings.Contains(context.CompressedToolSummary, "conflict_checker") {
		t.Fatalf("expected recent tool trace plus compressed older context: %#v", context)
	}
	overview, err := store.AgentMemoryOverview("AI Agent 市场证据")
	if err != nil {
		t.Fatal(err)
	}
	if overview.Runs != 7 || overview.Events != 42 || overview.Summaries == 0 {
		t.Fatalf("unexpected durable memory overview: %#v", overview)
	}
}
