package main

import "time"

type DeliveryCopilot struct {
	ResearchRunID         string              `json:"researchRunId,omitempty"`
	HistoryNotice         string              `json:"historyNotice,omitempty"`
	Workflow              []WorkflowStage     `json:"workflow"`
	Discovery             ProductAnalysis     `json:"discovery"`
	Decision              ProductDecision     `json:"decision"`
	PRD                   EnterprisePRD       `json:"prd"`
	Solution              SolutionDesign      `json:"solution"`
	Evaluation            DeliveryEvaluation  `json:"evaluation"`
	Visuals               DeliveryVisuals     `json:"visuals"`
	Readiness             PRDReadiness        `json:"readiness"`
	Chains                []EvidenceChain     `json:"chains"`
	GeneratedImages       []GeneratedPRDImage `json:"generatedImages,omitempty"`
	ImageJobID            string              `json:"imageJobId,omitempty"`
	ImageJobStatus        string              `json:"imageJobStatus,omitempty"`
	ImageRequestID        string              `json:"imageRequestId,omitempty"`
	ImageInputFingerprint string              `json:"imageInputFingerprint,omitempty"`
	GeneratedAt           time.Time           `json:"generatedAt"`
}

type WorkflowStage struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type ProductDecision struct {
	Goal       string           `json:"goal"`
	Options    []DecisionOption `json:"options"`
	Decision   string           `json:"decision"`
	Reasoning  string           `json:"reasoning"`
	MVP        []string         `json:"mvp"`
	OutOfScope []string         `json:"outOfScope"`
	Evidence   []EvidenceRef    `json:"evidence"`
}

type DecisionOption struct {
	Name     string `json:"name"`
	Benefit  string `json:"benefit"`
	Tradeoff string `json:"tradeoff"`
	Verdict  string `json:"verdict"`
}

type EnterprisePRD struct {
	Info           PRDDocumentInfo            `json:"info"`
	Background     PRDBackground              `json:"background"`
	Goals          []string                   `json:"goals"`
	NonGoals       []string                   `json:"nonGoals"`
	SuccessMetrics []SuccessMetric            `json:"successMetrics"`
	Users          []PRDUserScenario          `json:"users"`
	Architecture   []ArchitectureModule       `json:"architecture"`
	UserFlow       []string                   `json:"userFlow"`
	SystemFlow     []string                   `json:"systemFlow"`
	Permissions    []PermissionRule           `json:"permissions"`
	Requirements   []ExecutionRequirement     `json:"requirements"`
	NonFunctional  []NonFunctionalRequirement `json:"nonFunctional"`
	OpenQuestions  []string                   `json:"openQuestions"`
	Prototype      ProductPrototype           `json:"prototype"`
	DeliveryPlan   DeliveryPlan               `json:"deliveryPlan"`
}

type PRDDocumentInfo struct {
	PRDID         string `json:"prdID"`
	Product       string `json:"product"`
	Feature       string `json:"feature"`
	Owner         string `json:"owner"`
	Reviewers     string `json:"reviewers"`
	Status        string `json:"status"`
	Version       string `json:"version"`
	CreatedDate   string `json:"createdDate"`
	TargetRelease string `json:"targetRelease"`
	LastUpdated   string `json:"lastUpdated"`
}

type PRDBackground struct {
	BusinessContext  string        `json:"businessContext"`
	UserProblem      string        `json:"userProblem"`
	CurrentSituation string        `json:"currentSituation"`
	BaselineMetrics  string        `json:"baselineMetrics"`
	Evidence         []EvidenceRef `json:"evidence"`
}

type SuccessMetric struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
	Baseline   string `json:"baseline"`
	Target     string `json:"target"`
	Window     string `json:"window"`
	Owner      string `json:"owner"`
}
type PRDUserScenario struct {
	Role     string        `json:"role"`
	Buyer    string        `json:"buyer"`
	Scenario string        `json:"scenario"`
	Trigger  string        `json:"trigger"`
	Goal     string        `json:"goal"`
	Evidence []EvidenceRef `json:"evidence"`
}

type ArchitectureModule struct {
	Name     string   `json:"name"`
	Children []string `json:"children"`
}
type PermissionRule struct {
	Capability string `json:"capability"`
	User       string `json:"user"`
	CSAgent    string `json:"csAgent"`
	Admin      string `json:"admin"`
}

type ExecutionRequirement struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Priority      string        `json:"priority"`
	UserStory     string        `json:"userStory"`
	Preconditions []string      `json:"preconditions"`
	MainFlow      []string      `json:"mainFlow"`
	BusinessRules []string      `json:"businessRules"`
	Exceptions    []string      `json:"exceptions"`
	Permissions   []string      `json:"permissions"`
	Acceptance    []string      `json:"acceptance"`
	Tracking      []string      `json:"tracking"`
	Evidence      []EvidenceRef `json:"evidence"`
}

type NonFunctionalRequirement struct {
	Area         string `json:"area"`
	Requirement  string `json:"requirement"`
	Verification string `json:"verification"`
	Status       string `json:"status"`
}
type SolutionDesign struct {
	Pages                []string             `json:"pages"`
	APIs                 []string             `json:"apis"`
	DataEntities         []string             `json:"dataEntities"`
	RAGConstraints       []string             `json:"ragConstraints"`
	TechnicalConstraints []string             `json:"technicalConstraints"`
	DataModel            []DataEntity         `json:"dataModel"`
	DataDictionary       []DataDictionaryItem `json:"dataDictionary"`
	APIContracts         []APIContract        `json:"apiContracts"`
	Dependencies         []DeliveryDependency `json:"dependencies"`
}
type DeliveryEvaluation struct {
	Tracking        []string `json:"tracking"`
	TestFocus       []string `json:"testFocus"`
	LaunchChecklist []string `json:"launchChecklist"`
	Monitoring      []string `json:"monitoring"`
	Rollback        []string `json:"rollback"`
}
type DeliveryVisuals struct {
	Title        string           `json:"title"`
	Flow         []VisualFlowNode `json:"flow"`
	ProductFlow  []VisualFlowNode `json:"productFlow"`
	SolutionFlow []VisualFlowNode `json:"solutionFlow"`
	SOP          []SOPStep        `json:"sop"`
	TestPlan     TestPlan         `json:"testPlan"`
	DataNotice   string           `json:"dataNotice"`
}

// DeliveryPlan is an estimate, not a release commitment. It exposes the people,
// dependencies and go/no-go gates that may extend the schedule.
type DeliveryPlan struct {
	Estimate    string          `json:"estimate"`
	Assumptions []string        `json:"assumptions"`
	Phases      []DeliveryPhase `json:"phases"`
}

type DeliveryPhase struct {
	Window    string `json:"window"`
	Stage     string `json:"stage"`
	Owner     string `json:"owner"`
	Outcome   string `json:"outcome"`
	EntryGate string `json:"entryGate"`
	ExitGate  string `json:"exitGate"`
	DelayRisk string `json:"delayRisk"`
}

// ProductPrototype is a low-fidelity, buildable screen specification. It makes
// intended content, behavior and states reviewable before visual polish begins.
type ProductPrototype struct {
	Screens []PrototypeScreen `json:"screens"`
}

type PrototypeScreen struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Route         string            `json:"route"`
	Purpose       string            `json:"purpose"`
	PrimaryAction string            `json:"primaryAction"`
	Regions       []PrototypeRegion `json:"regions"`
	States        []string          `json:"states"`
}

type PrototypeRegion struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
	Behavior string `json:"behavior"`
}

type DataEntity struct {
	Name          string   `json:"name"`
	Purpose       string   `json:"purpose"`
	PrimaryKey    string   `json:"primaryKey"`
	Relationships []string `json:"relationships"`
	Retention     string   `json:"retention"`
	Access        string   `json:"access"`
}

type DataDictionaryItem struct {
	Entity         string `json:"entity"`
	Field          string `json:"field"`
	Type           string `json:"type"`
	Required       string `json:"required"`
	Description    string `json:"description"`
	Example        string `json:"example"`
	Classification string `json:"classification"`
}

type APIContract struct {
	Name     string   `json:"name"`
	Method   string   `json:"method"`
	Path     string   `json:"path"`
	Auth     string   `json:"auth"`
	Request  []string `json:"request"`
	Response []string `json:"response"`
	Failures []string `json:"failures"`
}

type DeliveryDependency struct {
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	NeedBy   string `json:"needBy"`
	Risk     string `json:"risk"`
	Fallback string `json:"fallback"`
}

type VisualFlowNode struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Owner  string `json:"owner"`
	Output string `json:"output"`
}

type SOPStep struct {
	Order       int    `json:"order"`
	Title       string `json:"title"`
	Owner       string `json:"owner"`
	Action      string `json:"action"`
	Input       string `json:"input"`
	Output      string `json:"output"`
	QualityGate string `json:"qualityGate"`
}

type TestPlan struct {
	Strategy  string        `json:"strategy"`
	Roles     []TestRole    `json:"roles"`
	Cases     []TestCase    `json:"cases"`
	DataSets  []TestDataSet `json:"dataSets"`
	ExitRules []string      `json:"exitRules"`
}

type TestRole struct {
	Role      string `json:"role"`
	Owner     string `json:"owner"`
	HowToTest string `json:"howToTest"`
	Exit      string `json:"exit"`
}

type TestCase struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Level    string   `json:"level"`
	Steps    []string `json:"steps"`
	Expected string   `json:"expected"`
}

type TestDataSet struct {
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
	Example string `json:"example"`
	Notice  string `json:"notice"`
}
type ReadinessCheck struct {
	Area   string `json:"area"`
	Status string `json:"status"`
	Note   string `json:"note"`
}
type PRDReadiness struct {
	Score       int              `json:"score"`
	Verdict     string           `json:"verdict"`
	Checks      []ReadinessCheck `json:"checks"`
	NextActions []string         `json:"nextActions"`
}
type EvidenceChain struct {
	Source      string        `json:"source"`
	Insight     string        `json:"insight"`
	Requirement string        `json:"requirement"`
	Evidence    []EvidenceRef `json:"evidence"`
}

// BuildDeliveryCopilot turns local discovery evidence into decision, implementation PRD, solution and review artifacts.
func (s *KnowledgeStore) BuildDeliveryCopilot() (DeliveryCopilot, error) {
	discovery, err := s.BuildProductAnalysis()
	if err != nil {
		return DeliveryCopilot{}, err
	}
	return buildDeliveryCopilotFromAnalysis(discovery), nil
}

// BuildDeliveryCopilotForResearch creates a new in-memory PRD bound to one completed
// research run. It never depends on the demo knowledge set and does not overwrite another run.
func (s *KnowledgeStore) BuildDeliveryCopilotForResearch(result ResearchResult) (DeliveryCopilot, error) {
	discovery, err := productAnalysisFromResearch(result)
	if err != nil {
		return DeliveryCopilot{}, err
	}
	return tailorDeliveryForResearch(buildDeliveryCopilotFromAnalysis(discovery), result), nil
}

func buildDeliveryCopilotFromAnalysis(discovery ProductAnalysis) DeliveryCopilot {
	evidence := discovery.Summary.Evidence
	if len(evidence) == 0 {
		evidence = []EvidenceRef{{Label: "证据待补充", Type: "gap"}}
	}
	feedbackEvidence := evidence
	if len(discovery.UserFeedback) > 0 && len(discovery.UserFeedback[0].Evidence) > 0 {
		feedbackEvidence = discovery.UserFeedback[0].Evidence
	}
	feedbackAt := func(index int) []EvidenceRef {
		if index >= 0 && index < len(discovery.UserFeedback) && len(discovery.UserFeedback[index].Evidence) > 0 {
			return discovery.UserFeedback[index].Evidence
		}
		return feedbackEvidence
	}
	painAt := func(index int) []EvidenceRef {
		if index >= 0 && index < len(discovery.PainPoints) && len(discovery.PainPoints[index].Evidence) > 0 {
			return discovery.PainPoints[index].Evidence
		}
		return feedbackEvidence
	}
	evidenceAt := func(index int) EvidenceRef {
		if index >= 0 && index < len(evidence) {
			return evidence[index]
		}
		return EvidenceRef{Label: "证据待补充", Type: "gap"}
	}
	return DeliveryCopilot{
		Workflow: []WorkflowStage{
			{ID: "discover", Title: "Discover", Description: "核验问题、用户、证据、冲突与机会。", Status: "ready"},
			{ID: "decide", Title: "Decide", Description: "比较方案，锁定价值、优先级、MVP 与非范围。", Status: "ready"},
			{ID: "design", Title: "Design", Description: "审查原型、流程、数据、接口、权限与技术可行性。", Status: "draft"},
			{ID: "baseline", Title: "PRD Baseline", Description: "将需求、验收、埋点和证据冻结为可评审版本。", Status: "draft"},
			{ID: "build", Title: "Build & Test", Description: "开发、测试设计、质量验证与缺陷处理。", Status: "attention"},
			{ID: "release", Title: "UAT & Release", Description: "试点验收、发布评审、灰度与回滚准备。", Status: "attention"},
			{ID: "learn", Title: "Measure & Learn", Description: "监控指标、收集反馈，并把变更带回下一版本。", Status: "attention"},
		},
		Discovery: discovery,
		Decision: ProductDecision{
			Goal: "验证可审计研究工作台能否减少团队对 AI 结论的核验成本，并使内部资料可被安全复用。",
			Options: []DecisionOption{
				{Name: "泛化聊天 Agent", Benefit: "上线快、覆盖场景多", Tradeoff: "与现有助手同质化，结论无法被验证", Verdict: "不作为 MVP 主线"},
				{Name: "仅做文档/RAG 搜索", Benefit: "资料汇总明确、实施风险较低", Tradeoff: "不能形成从洞察到需求的决策闭环", Verdict: "作为基础能力"},
				{Name: "可审计产品交付 Copilot", Benefit: "把证据、洞察、需求、规则和验收串成可复核链路", Tradeoff: "需要先解决权限、资料质量和流程复杂度", Verdict: "推荐 MVP"},
			},
			Decision:   "MVP 聚焦产品/客服洞察团队：用可追溯资料生成产品发现与研发可评审 PRD，而不是通用 Agent 平台。",
			Reasoning:  "历史反馈反复指向引用、冲突、权限、审计和工单结构化；这些问题决定了企业是否信任并扩大使用。",
			MVP:        []string{"历史资料统一检索与来源标签", "Evidence ↔ Insight ↔ Requirement 链路", "可审计 PRD 的需求、规则、异常和验收标准", "Reviewer Readiness 缺口检查"},
			OutOfScope: []string{"自动替代产品负责人做最终商业决策", "无人工确认地发布 PRD 或修改生产资料", "一次性覆盖所有行业的 SOP 与系统集成"},
			Evidence:   evidence,
		},
		PRD: EnterprisePRD{
			Info:       PRDDocumentInfo{PRDID: "AIPDC-PRD-001", Product: "AI Product Delivery Copilot", Feature: "Evidence & Requirements Workspace", Owner: "待指定 Product Owner", Reviewers: "Product · Design · Engineering · QA · Security", Status: "Draft — review required", Version: "0.1", CreatedDate: time.Now().Format("2006-01-02"), TargetRelease: "待评审后确定", LastUpdated: time.Now().Format("2006-01-02 15:04")},
			Background: PRDBackground{BusinessContext: "产品、客服与安全团队需要把分散资料转为可审阅的决策和交付物；仅有摘要无法支持研发评审。", UserProblem: "用户无法快速追溯结论、发现冲突资料或确认内部资料的权限和时效性。", CurrentSituation: "当前已有研究报告和本地知识库，但产品发现、方案决策、企业 PRD 与交付评估尚未形成显式链路。", BaselineMetrics: "当前仅有演示样本（访谈/工单/洞察），不是生产基线；上线前必须补充真实工单量、核验耗时、人工接受率和支持成本。", Evidence: evidence},
			Goals:      []string{"让每条关键需求都可回到访谈、工单、旧 PRD 或研究证据", "让产品、设计、研发、测试对范围、流程、规则和验收达成一致", "让低置信度、冲突和资料缺口进入评审，而不是被隐藏"},
			NonGoals:   []string{"不自动修改业务状态或代表用户执行高风险动作", "不把模拟资料、单一访谈或单条工单当作市场事实", "不在本版本替代技术设计和安全评审"},
			SuccessMetrics: []SuccessMetric{
				{Name: "关键需求可追溯率", Definition: "含 Evidence ID 或明确 Gap 的 P0/P1 需求 ÷ 全部 P0/P1 需求", Baseline: "演示基线；生产需首周采集", Target: "上线后 ≥95%，无来源需求不得进入研发评审", Window: "每周", Owner: "Product Owner"},
				{Name: "PRD 可评审周期", Definition: "从问题创建到通过 Readiness 评审的工作时长", Baseline: "待首批真实项目采集", Target: "首个试点以较当前人工流程更短为目标；数值由业务 Owner 确认", Window: "每个试点", Owner: "Product + Engineering"},
				{Name: "证据核验效率", Definition: "评审者从结论定位到原始证据的成功率与耗时", Baseline: "待可用性测试采集", Target: "≥90% 测试参与者可独立完成核验；耗时目标待基线确认", Window: "UAT", Owner: "Design + QA"},
				{Name: "权限与审计合规", Definition: "未授权原文暴露、审计缺失和高风险操作绕过的数量", Baseline: "0 起（测试环境）", Target: "0 起高危问题；100% 受限资料读取可审计", Window: "每次发布", Owner: "Security"},
			}, Users: []PRDUserScenario{
				{Role: "产品负责人", Buyer: "产品/业务负责人", Scenario: "准备机会评审和路线图评审", Trigger: "收到一个模糊业务问题或一组用户反馈", Goal: "快速判断问题是否成立、先做什么、仍缺什么证据", Evidence: feedbackAt(1)},
				{Role: "客服运营负责人", Buyer: "客服/运营负责人", Scenario: "每周复盘高频工单", Trigger: "导入或同步一批客服工单", Goal: "获得可回到原始对话的痛点、机会和待验证需求", Evidence: feedbackEvidence},
				{Role: "IT 安全负责人", Buyer: "企业 IT/安全负责人", Scenario: "评估扩大内部资料访问范围", Trigger: "团队申请将 Agent 接入新的内部知识源", Goal: "确认权限、审计、数据保留和异常边界", Evidence: feedbackAt(2)},
			},
			Architecture: []ArchitectureModule{
				{Name: "Product Delivery Workspace", Children: []string{"Research intake", "Historical knowledge search", "Evidence map"}},
				{Name: "Product Discovery", Children: []string{"Problem statement", "Persona & JTBD", "Scenario & journey", "Pain point & opportunity"}},
				{Name: "Product Plan", Children: []string{"Option comparison", "MVP scope", "Priority decision", "Out-of-scope"}},
				{Name: "Enterprise PRD", Children: []string{"Requirements", "Business rules", "Permissions", "Exceptions", "Acceptance & tracking"}},
				{Name: "Review & Evaluation", Children: []string{"Readiness check", "Test plan", "Monitoring", "Launch & rollback"}},
			},
			UserFlow:   []string{"选择或输入业务问题", "检索历史访谈、工单、文档和数据", "阅读 Product Discovery 与证据", "确认方案/MVP 范围", "生成并编辑企业 PRD", "Reviewer 检查缺口", "进入设计、研发和测试评审"},
			SystemFlow: []string{"资料导入与元数据校验", "分块/结构化记录入库", "统一历史检索与相关性排序", "Evidence → Insight → Requirement 关联", "生成 PRD 草案", "检查可测试性、权限、埋点、性能和回滚缺口", "输出评审状态与待补充动作"},
			Permissions: []PermissionRule{
				{Capability: "查看公开研究结论", User: "✓", CSAgent: "✓", Admin: "✓"},
				{Capability: "查看受限内部资料原文", User: "按访问级别", CSAgent: "按访问级别", Admin: "✓"},
				{Capability: "编辑需求与规则", User: "创建者/协作者", CSAgent: "—", Admin: "✓"},
				{Capability: "批准进入研发评审", User: "指定 Owner", CSAgent: "—", Admin: "✓"},
				{Capability: "查看审计记录", User: "—", CSAgent: "—", Admin: "✓"},
			},
			Requirements: []ExecutionRequirement{
				{ID: "REQ-001", Title: "Evidence Detail Viewer", Priority: "P0", UserStory: "作为产品负责人，我希望点击结论即可查看原文、来源、时间和置信度，以便在评审前核验判断。", Preconditions: []string{"用户已登录", "研究任务或历史资料至少存在一条证据", "用户具备资料访问权限"}, MainFlow: []string{"用户点击报告或需求中的证据标签", "系统加载证据片段、来源、抓取/创建时间和访问状态", "系统同时显示相关支持或冲突资料", "用户可标记接受、存疑或待补充"}, BusinessRules: []string{"无证据时不得输出确定性结论", "资料已过期时必须显示过期标记", "受限资料不得把正文返回给无权限用户"}, Exceptions: []string{"来源链接失效：显示最后获取时间与失效状态", "资料权限不足：显示受限原因而非原文", "无冲突资料：明确显示未发现，不等于不存在"}, Permissions: []string{"普通用户仅访问其有权限的证据", "管理员可查看审计与访问范围"}, Acceptance: []string{"Given 用户有可读证据 When 点击引用 Then 2 秒内显示来源、时间、片段和置信度", "Given 用户无资料权限 When 打开引用 Then 不返回受限正文", "Given 结论无证据 When 生成报告 Then 显示证据不足"}, Tracking: []string{"evidence_detail_view", "evidence_access_denied", "evidence_marked_uncertain"}, Evidence: evidence},
				{ID: "REQ-002", Title: "Conflict & Gap Detection", Priority: "P0", UserStory: "作为研究分析师，我希望看见相互冲突和缺失的资料，以便避免把不确定性写成事实。", Preconditions: []string{"任务已完成至少一次资料读取"}, MainFlow: []string{"系统按结论主题聚合支持资料", "系统标记相反结论、过期资料和低置信度资料", "系统生成待验证问题清单"}, BusinessRules: []string{"冲突必须与原始资料并列展示", "低置信度不能被自动提升为高置信度"}, Exceptions: []string{"资料量不足：输出证据缺口，不虚构冲突"}, Permissions: []string{"遵循各资料原有访问级别"}, Acceptance: []string{"Given 同主题存在相反资料 When 生成分析 Then 同时列出支持和冲突来源", "Given 资料不足 When 生成需求 Then 产生待验证项"}, Tracking: []string{"conflict_detected", "evidence_gap_created"}, Evidence: painAt(1)},
				{ID: "REQ-003", Title: "Structured Feedback to Insight", Priority: "P1", UserStory: "作为客服运营负责人，我希望把工单转为带原始依据的洞察和需求候选，以便减少手工汇总。", Preconditions: []string{"已导入格式合法的工单或访谈资料"}, MainFlow: []string{"用户导入 CSV/文档", "系统检查字段、重复记录和访问级别", "系统按主题生成候选痛点", "用户确认后将候选写入 Insight 与 Requirement"}, BusinessRules: []string{"洞察必须保留来源记录 ID", "未经人工确认的候选不能成为已批准需求"}, Exceptions: []string{"字段映射不完整：要求用户补全", "文本无法解析：保留原文件并记录失败原因"}, Permissions: []string{"客服角色可上传和查看其范围内资料", "管理员可管理字段映射"}, Acceptance: []string{"Given 工单字段齐全 When 导入 Then 生成可回溯候选洞察", "Given 字段缺失 When 导入 Then 阻止发布并提示映射规则"}, Tracking: []string{"feedback_import_started", "feedback_import_failed", "insight_confirmed"}, Evidence: painAt(3)},
				{ID: "REQ-004", Title: "PRD Readiness Reviewer", Priority: "P1", UserStory: "作为产品 Owner，我希望在提交研发评审前看到 PRD 缺口，以便减少返工和模糊需求。", Preconditions: []string{"PRD 已包含至少一个 Requirement"}, MainFlow: []string{"系统扫描文档信息、背景、流程、规则、权限、异常、验收、埋点和回滚", "系统输出通过/注意/缺失状态", "系统给出可执行的补充动作"}, BusinessRules: []string{"“页面应简单”“响应要快”等不可测试语句必须标为不通过", "Readiness 未通过时不得标记为 Ready for Engineering"}, Exceptions: []string{"没有真实基线：标记为注意，不自动编造指标"}, Permissions: []string{"产品 Owner 可编辑", "Reviewer 可评论但不覆盖 Owner"}, Acceptance: []string{"Given 存在不可测需求 When Reviewer 运行 Then 返回具体替换建议", "Given 缺少回滚方案 When 评审 Then Readiness 标记为缺失"}, Tracking: []string{"prd_review_started", "prd_gap_found", "prd_marked_ready"}, Evidence: []EvidenceRef{evidenceAt(0), evidenceAt(1)}},
			},
			NonFunctional: []NonFunctionalRequirement{
				{Area: "Security & privacy", Requirement: "所有内部资料读取必须执行访问级别校验，并记录审计事件。", Verification: "权限自动化测试 + 审计抽样", Status: "必须在研发评审前确认"},
				{Area: "Performance", Requirement: "证据详情查询 P95 延迟目标 < 800ms；该数值需技术评审确认。", Verification: "压测报告", Status: "待工程确认"},
				{Area: "Reliability", Requirement: "来源失效、模型失败或解析失败时，不得伪造证据；必须展示失败状态。", Verification: "异常注入测试", Status: "必须"},
				{Area: "Cost control", Requirement: "每次研究运行记录查询次数、读取数、模型调用与预算上限。", Verification: "运行日志与月度成本报表", Status: "待产品/财务确认"},
			},
			OpenQuestions: []string{"真实用户的核验耗时、客服转人工率和报告接受率基线是多少？", "哪些内部资料允许进入检索，分别对应什么保留/脱敏规则？", "目标系统需要接入哪些工单、知识库、身份与审计平台？", "上线后的回滚开关、数据删除和历史版本策略是什么？"},
			DeliveryPlan:  defaultDeliveryPlan(),
			Prototype: ProductPrototype{Screens: []PrototypeScreen{
				{ID: "SCR-01", Name: "研究任务工作台", Route: "/", Purpose: "让产品负责人输入可验证问题，并在同一屏查看计划、实时工具轨迹、来源和证据。", PrimaryAction: "建立研究计划", Regions: []PrototypeRegion{{Name: "问题输入", Content: "研究问题、范围、时间和地区", Behavior: "校验最少 8 个字符；创建任务后锁定本次问题。"}, {Name: "计划与活动", Content: "五步计划、当前工具和进度", Behavior: "按 Agent 事件实时更新；失败显示可恢复原因。"}, {Name: "证据与来源", Content: "来源清单、置信度、冲突和 Evidence ID", Behavior: "点击证据只显示当前角色可访问的原文。"}}, States: []string{"空态：展示问题示例", "加载：显示当前工具与数量", "失败：显示失败范围和重试建议", "完成：开放报告、PRD 与导出"}},
				{ID: "SCR-02", Name: "证据与洞察工作台", Route: "/#evidence", Purpose: "把结论、原始证据、冲突和待验证问题并列，供产品评审确认。", PrimaryAction: "确认洞察并进入方案比较", Regions: []PrototypeRegion{{Name: "结论", Content: "主张、置信度、适用条件", Behavior: "未绑定证据的结论不可标记为已确认。"}, {Name: "证据地图", Content: "Evidence ID、来源、时间、摘录、支持/冲突", Behavior: "按相关性排序并保留原始链接。"}, {Name: "缺口", Content: "冲突资料、低置信度和待验证问题", Behavior: "进入 PRD 时自动作为开放问题带入。"}}, States: []string{"无证据：提示继续检索", "有冲突：支持与反证并列", "权限受限：隐藏正文且记录审计"}},
				{ID: "SCR-03", Name: "PRD 编辑与评审", Route: "/#delivery", Purpose: "让跨职能团队在需求、流程、数据、接口与验收上形成同一版本基线。", PrimaryAction: "保存 PRD 草案", Regions: []PrototypeRegion{{Name: "需求目录", Content: "P0/P1、状态和证据覆盖", Behavior: "选择需求后定位到规则、异常、验收与埋点。"}, {Name: "规格正文", Content: "用户故事、流程、原型、数据、API、验收", Behavior: "显示字段级缺口与关联证据。"}, {Name: "评审侧栏", Content: "Readiness、评论、审批队列", Behavior: "未达到门槛不可提交研发评审。"}}, States: []string{"草案：允许编辑", "评审中：锁定基线并收集角色决策", "已拒绝：要求新版本", "已批准：生成发布检查清单"}},
			}},
		},
		Solution: SolutionDesign{
			Pages:                []string{"问题输入与研究计划页", "历史知识搜索页", "Evidence ↔ Insight ↔ Requirement 工作台", "PRD 编辑与版本页", "Reviewer Readiness 页", "管理端：资料、权限与审计"},
			APIs:                 []string{"POST /api/research：研究与证据提取", "POST /api/knowledge/history：跨表历史检索", "GET /api/delivery-copilot：生成交付草案", "POST /api/delivery-copilot/persist：保存版本、需求、证据和评审", "POST /api/delivery-copilot/submit-review：通过 Readiness 门槛后进入审批", "POST /api/delivery-copilot/approvals：记录角色审批（接入身份认证后开放）"},
			DataEntities:         []string{"documents / document_chunks", "support_tickets / interviews / insights", "product_initiatives / prd_versions / prd_requirements", "evidence_links / prd_reviews / prd_approvals", "audit_events（后续：用于企业级合规审计）"},
			RAGConstraints:       []string{"Requirement 生成前必须检索旧 PRD、SOP、业务规则、工单和访谈", "把检索来源写入 requirement evidence，不允许无来源的确定性规则", "检索不到业务规则时生成 Gap，而不是推测"},
			TechnicalConstraints: []string{"权限校验在检索层和展示层同时执行", "模型输出必须经过结构校验和 Reviewer 检查", "高风险动作必须要求人工确认", "资料状态变为 deprecated 后不作为默认依据"},
			DataModel: []DataEntity{
				{Name: "ResearchRun", Purpose: "一次可复现的研究任务及工具轨迹", PrimaryKey: "run_id", Relationships: []string{"1:N Evidence", "1:N AgentEvent", "1:N Report"}, Retention: "按研究策略保留；删除时级联清理可删除产物", Access: "创建者、协作者与管理员"},
				{Name: "Evidence", Purpose: "可回到来源的摘录、评分、时间与冲突关系", PrimaryKey: "evidence_id", Relationships: []string{"N:1 ResearchRun", "N:M Requirement（经 EvidenceLink）"}, Retention: "保留来源、抓取时间和失效状态", Access: "继承来源访问级别"},
				{Name: "Insight", Purpose: "经人工确认的痛点、机会或待验证判断", PrimaryKey: "insight_id", Relationships: []string{"N:M Evidence", "1:N Requirement 候选"}, Retention: "保留确认者、确认时间与撤销原因", Access: "产品与授权业务角色"},
				{Name: "PRDVersion", Purpose: "可评审的需求基线与变更历史", PrimaryKey: "prd_version_id", Relationships: []string{"1:N Requirement", "1:N Review", "1:N Approval"}, Retention: "版本不可覆盖；新修改创建新版本", Access: "Owner 可编辑，Reviewer 可评论"},
				{Name: "Requirement", Purpose: "可实现、可测试并可追溯的单条需求", PrimaryKey: "requirement_id", Relationships: []string{"N:M Evidence", "N:1 PRDVersion", "1:N TestCase"}, Retention: "随 PRD 版本保留", Access: "产品、设计、研发、QA"},
			},
			DataDictionary: []DataDictionaryItem{
				{Entity: "Evidence", Field: "evidence_id", Type: "string / UUID", Required: "是", Description: "证据稳定标识，用于报告、洞察和需求追溯。", Example: "E-20261004-001", Classification: "内部元数据"},
				{Entity: "Evidence", Field: "source_url", Type: "URL", Required: "条件必填", Description: "公开或授权资料的原始地址；内部资料使用 document_id。", Example: "https://example.com/report", Classification: "来源元数据"},
				{Entity: "Evidence", Field: "quote", Type: "text ≤ 1600 chars", Required: "是", Description: "支撑或反驳主张的最小原文摘录。", Example: "…", Classification: "按来源分级"},
				{Entity: "Evidence", Field: "confidence", Type: "enum(high,medium,low)", Required: "是", Description: "基于来源质量、时效、相关性和交叉验证的标签。", Example: "high", Classification: "内部分析"},
				{Entity: "Requirement", Field: "requirement_key", Type: "string", Required: "是", Description: "PRD 中可引用的稳定需求编号。", Example: "REQ-001", Classification: "内部产品资料"},
				{Entity: "Requirement", Field: "priority", Type: "enum(P0,P1,P2)", Required: "是", Description: "P0 阻断核心价值或合规；P1 重要；P2 可延后。", Example: "P0", Classification: "内部产品资料"},
				{Entity: "PRDVersion", Field: "status", Type: "enum(draft,in_review,approved,rejected)", Required: "是", Description: "版本评审状态；批准后不可原地修改。", Example: "draft", Classification: "内部流程资料"},
				{Entity: "AuditEvent", Field: "actor_id", Type: "string", Required: "是", Description: "执行查看、导出、审批或修改的主体。", Example: "user_123", Classification: "受限审计数据"},
			},
			APIContracts: []APIContract{
				{Name: "启动研究", Method: "POST", Path: "/api/research", Auth: "当前本地原型无身份层；生产必须要求会话身份与配额", Request: []string{"question: string，至少 8 字符"}, Response: []string{"SSE progress：步骤、工具、数量、时间", "complete：ResearchResult + Evidence"}, Failures: []string{"400：问题无效", "429：超出研究预算（生产）", "502：上游来源或模型失败，保留已完成证据"}},
				{Name: "检索历史资料", Method: "POST", Path: "/api/knowledge/history", Auth: "需按资料访问级别过滤", Request: []string{"query: string", "type: all | document | ticket | interview | insight | prd | note | dataset"}, Response: []string{"records：命中记录、类型与摘要", "total：命中数"}, Failures: []string{"400：请求无效", "403：无资料访问权限（生产）"}},
				{Name: "保存 PRD 基线", Method: "POST", Path: "/api/delivery-copilot/persist", Auth: "Product Owner 或授权协作者", Request: []string{"当前生成的可追溯 PRD 快照"}, Response: []string{"version、需求数、证据链接、评审和审批队列"}, Failures: []string{"409：当前资料无法生成草案", "403：非 Owner（生产）"}},
				{Name: "导出完整 PRD", Method: "GET", Path: "/api/delivery-copilot/export?format=docx", Auth: "仅导出调用者有权查看的版本", Request: []string{"format: docx"}, Response: []string{"下载包含流程、原型、数据字典、接口、测试和证据链的文件"}, Failures: []string{"400：格式不支持", "403：导出权限不足（生产）"}},
			},
			Dependencies: []DeliveryDependency{
				{Name: "身份与权限服务", Owner: "Security / Platform", NeedBy: "研发开始前", Risk: "缺少用户与资料级授权会造成越权读取。", Fallback: "仅使用隔离测试账号与模拟访问级别，不接入真实受限资料。"},
				{Name: "工单与知识库连接器", Owner: "Customer Ops / IT", NeedBy: "试点前", Risk: "字段与同步策略未确认，无法建立真实基线。", Fallback: "先导入脱敏 CSV 并固定字段映射。"},
				{Name: "模型与搜索供应商", Owner: "Engineering", NeedBy: "开发阶段", Risk: "限流、费用或失败降级不明确。", Fallback: "限制任务预算，保留证据报告降级和人工补充。"},
			},
		},
		Evaluation: DeliveryEvaluation{
			Tracking:        []string{"evidence_detail_view", "history_search", "insight_confirmed", "requirement_created", "prd_gap_found", "prd_marked_ready"},
			TestFocus:       []string{"权限越权与受限原文泄露", "无证据/冲突证据/失效来源的展示", "CSV 字段映射和重复导入", "Requirement 验收条件的 Given/When/Then 可测试性"},
			LaunchChecklist: []string{"真实业务基线已补齐", "Owner、Reviewer、目标发布日期已指定", "权限与保留策略已通过安全评审", "P0 Requirement 均有验收、埋点和异常流程"},
			Monitoring:      []string{"资料解析失败率", "无证据结论拦截数", "权限拒绝事件", "P95 证据详情延迟", "人工接受/修改/拒绝分析建议比例"},
			Rollback:        []string{"关闭自动 PRD 发布，仅保留草案", "将新资料源设为不可检索", "回退到上一版 requirement/PRD", "保留审计记录并通知 Owner"},
		},
		Visuals: DeliveryVisuals{
			Title: "Evidence-driven Product Delivery Map",
			// Flow is the cross-functional delivery lifecycle, not the in-product click path.
			Flow: []VisualFlowNode{
				{ID: "D1", Title: "问题发现与证据核验", Owner: "Product + Research", Output: "Validated problem brief"},
				{ID: "D2", Title: "方案比较与范围决策", Owner: "Product + Business", Output: "Decision memo + MVP scope"},
				{ID: "D3", Title: "方案设计与技术可行性", Owner: "Design + Engineering + Security", Output: "Prototype + technical feasibility"},
				{ID: "D4", Title: "PRD 基线与评审", Owner: "Product Owner + reviewers", Output: "Versioned PRD baseline"},
				{ID: "D5", Title: "开发与测试设计", Owner: "Engineering + QA", Output: "Build plan + test cases"},
				{ID: "D6", Title: "QA、UAT 与发布评审", Owner: "QA + Pilot + Security", Output: "Test evidence + go/no-go"},
				{ID: "D7", Title: "灰度、监控与复盘", Owner: "Operations + Product", Output: "Release metrics + change request"},
			},
			ProductFlow: []VisualFlowNode{
				{ID: "P1", Title: "输入业务问题", Owner: "Product Owner", Output: "Research brief"},
				{ID: "P2", Title: "检索与阅读资料", Owner: "Research Agent", Output: "Source set"},
				{ID: "P3", Title: "证据、冲突与洞察", Owner: "Research + Product", Output: "Evidence map"},
				{ID: "P4", Title: "方案比较", Owner: "Product Owner", Output: "Decision memo"},
				{ID: "P5", Title: "编辑需求与方案", Owner: "Product + Design", Output: "PRD draft"},
				{ID: "P6", Title: "Readiness 与角色评审", Owner: "Cross-functional reviewers", Output: "Review decision"},
			},
			SolutionFlow: []VisualFlowNode{
				{ID: "S1", Title: "Web / Internal sources", Owner: "Connectors", Output: "Authorized source records"},
				{ID: "S2", Title: "Research & retrieval", Owner: "Research Engine", Output: "Read pages + chunks"},
				{ID: "S3", Title: "Evidence & insight store", Owner: "SQLite / Evidence Store", Output: "Traceable links"},
				{ID: "S4", Title: "PRD workspace", Owner: "Product Delivery Copilot", Output: "Versioned requirements"},
				{ID: "S5", Title: "Review & audit", Owner: "Reviewers", Output: "Readiness + approvals"},
			},
			SOP: []SOPStep{
				{Order: 1, Title: "发现并确认问题", Owner: "Product Owner + Research", Action: "定义决策问题、目标用户、业务边界；检索并核验支持、冲突和证据缺口。", Input: "业务问题、访谈、工单和研究资料", Output: "Validated problem brief", QualityGate: "问题有可追溯证据；不把单一来源或低置信度当事实。"},
				{Order: 2, Title: "比较方案并锁定 MVP", Owner: "Product + Business", Action: "比较候选方案的收益、成本、风险与依赖，明确不做事项和优先级。", Input: "确认后的洞察与业务约束", Output: "Decision memo + MVP scope", QualityGate: "方案选择有取舍依据，P0/P1 与非范围明确。"},
				{Order: 3, Title: "设计体验与技术可行性", Owner: "Design + Engineering + Security", Action: "审查低保真原型、关键流程、数据模型、接口、权限和失败路径。", Input: "MVP 范围与用户场景", Output: "Prototype + solution design", QualityGate: "关键依赖、数据分级、异常与性能假设有负责人。"},
				{Order: 4, Title: "冻结 PRD 评审基线", Owner: "Product Owner", Action: "形成用户故事、规则、异常、验收、埋点、数据字典和测试范围，创建版本。", Input: "方案设计与评审意见", Output: "Versioned PRD baseline", QualityGate: "每个 P0/P1 均有证据或明确 Gap、验收和责任人。"},
				{Order: 5, Title: "开发、QA 与 UAT", Owner: "Engineering + QA + Pilot", Action: "按版本执行开发、自动化/手工测试、权限测试和业务试点验收。", Input: "PRD、测试数据、环境与发布计划", Output: "Test evidence + defects", QualityGate: "阻断缺陷关闭，关键用例、权限与失败降级均有结果。"},
				{Order: 6, Title: "发布、监控与复盘", Owner: "Operations + Product + Security", Action: "执行 go/no-go、灰度、监控、回滚准备和发布后复盘；新问题进入下一版本。", Input: "批准版本、测试证据和回滚方案", Output: "Release decision + metrics", QualityGate: "审批完整、回滚可执行、监控指标和 Owner 已明确。"},
			},
			TestPlan: TestPlan{
				Strategy: "分层测试：先验证需求可测性与证据链，再执行功能、权限、异常、性能和试点验收。所有测试结果必须绑定 PRD 版本。",
				Roles: []TestRole{
					{Role: "产品验收", Owner: "Product Owner", HowToTest: "逐条核对用户故事、范围、业务规则、异常与 Given/When/Then 验收标准。", Exit: "P0/P1 需求均有可验证验收结论。"},
					{Role: "功能与回归测试", Owner: "QA Engineer", HowToTest: "按冒烟、主流程、异常与回归用例执行；记录版本、步骤、结果和缺陷。", Exit: "阻断/严重缺陷为 0，核心用例通过率达到发布门槛。"},
					{Role: "权限与安全测试", Owner: "Security / QA", HowToTest: "使用不同访问级别账号测试原文、证据详情、导出与审批接口。", Exit: "无越权读取、无受限正文泄露，审计事件完整。"},
					{Role: "技术验证", Owner: "Engineering", HowToTest: "在测试环境压测证据详情、模拟来源失效、模型失败和回滚开关。", Exit: "性能目标、失败降级和回滚路径均有证据。"},
					{Role: "业务试点验收", Owner: "Customer Success / Pilot users", HowToTest: "用脱敏真实场景或合成样本完成研究到 PRD 的完整任务，记录理解度与修改建议。", Exit: "试点用户能独立复核关键结论，问题被分类并进入迭代。"},
				},
				Cases: []TestCase{
					{ID: "TC-EVD-001", Title: "证据详情可追溯", Level: "P0 / Functional", Steps: []string{"生成含引用的结论", "点击证据标签", "查看来源、时间、摘录与置信度"}, Expected: "2 秒内展示可访问资料；无权限时不返回正文。"},
					{ID: "TC-CON-002", Title: "冲突与证据不足不被隐藏", Level: "P0 / Functional", Steps: []string{"导入相反观点或不足资料", "运行分析", "查看结论与 Evidence Map"}, Expected: "系统并列展示冲突，或明确标记证据不足与待验证问题。"},
					{ID: "TC-PERM-003", Title: "受限资料权限校验", Level: "P0 / Security", Steps: []string{"以低权限账号打开受限证据", "调用详情和导出操作", "检查审计事件"}, Expected: "请求被拒绝，正文不泄露，审计记录包含主体、资源与时间。"},
					{ID: "TC-IMP-004", Title: "工单导入字段与重复处理", Level: "P1 / Data", Steps: []string{"导入字段完整、缺失字段和重复记录的 CSV", "检查导入结果", "核对候选洞察"}, Expected: "完整数据可追溯入库；缺失字段阻止发布；重复记录被标记。"},
					{ID: "TC-FAIL-005", Title: "来源失效与模型失败降级", Level: "P1 / Resilience", Steps: []string{"模拟链接失效或模型请求失败", "重新生成报告", "查看 PRD 和日志"}, Expected: "不伪造结论；明确失败、最后获取时间和人工补充动作。"},
				},
				DataSets: []TestDataSet{
					{Name: "合成客服工单集", Purpose: "验证导入、分类、重复识别与洞察候选。", Example: "12 条脱敏/合成工单：引用缺失、权限问题、导入失败、冲突反馈。", Notice: "不得使用真实客户个人信息。"},
					{Name: "证据冲突资料集", Purpose: "验证冲突与低置信度展示。", Example: "同一主题的支持观点、反对观点、过期资料与无来源片段。", Notice: "用于测试，不代表市场事实。"},
					{Name: "角色权限账号集", Purpose: "验证访问范围、导出与审批权限。", Example: "普通用户、客服、产品 Owner、管理员、安全 Reviewer 五种测试账号。", Notice: "使用隔离测试租户和虚拟身份。"},
					{Name: "失败注入资料集", Purpose: "验证来源失效、解析失败和模型降级。", Example: "404 URL、乱码 HTML、空文件、超大文件、超时模拟响应。", Notice: "仅在测试环境使用。"},
				},
				ExitRules: []string{"所有 P0 用例通过，且没有阻断/严重未关闭缺陷", "权限、受限原文和导出路径完成安全验证", "Readiness ≥ 80，Owner、发布日期和回滚责任已填写", "试点验收的反馈、缺口和变更进入版本化 PRD"},
			},
			DataNotice: "测试数据均为合成或脱敏样本；当前不会自动把模拟测试结果当作生产通过证据。",
		}, Readiness: PRDReadiness{
			Score: 74, Verdict: "Not ready for engineering review — PRD 结构已补齐；仍须确认真实基线、Owner/Release、技术可行性、性能目标与回滚责任。",
			Checks: []ReadinessCheck{
				{Area: "Document Control", Status: "attention", Note: "PRD ID 和版本已生成，但 Owner、评审人和目标发布日期尚未指定。"},
				{Area: "Background & Evidence", Status: "attention", Note: "证据链已存在；当前包含演示资料，真实业务基线缺失。"},
				{Area: "User / Scenario", Status: "pass", Note: "已覆盖产品、客服运营与安全角色及触发场景。"},
				{Area: "Flow / Rules / Exceptions", Status: "pass", Note: "产品流程、交付生命周期和 P0/P1 的规则、异常已定义。"},
				{Area: "Prototype & Data", Status: "attention", Note: "低保真原型、数据模型和字典已给出；需设计、数据和工程共同确认字段与状态。"},
				{Area: "API & Dependencies", Status: "attention", Note: "接口契约与依赖已列出；生产身份、连接器和限流方案尚未确认。"},
				{Area: "Permissions", Status: "attention", Note: "权限矩阵已定义，需落到具体身份系统和资料分级。"},
				{Area: "Acceptance Criteria", Status: "pass", Note: "核心需求已使用可测试的 Given/When/Then。"},
				{Area: "Analytics", Status: "attention", Note: "埋点事件已列出，事件属性与基线目标待补。"},
				{Area: "Performance & Rollback", Status: "missing", Note: "性能阈值和回滚责任人尚未经过工程/运维确认。"},
			},
			NextActions: []string{"指定 Product Owner、Reviewers 和目标发布日期", "用真实访谈、工单与业务指标替换演示基线", "让设计、工程和安全确认原型状态、数据字典、接口契约与依赖 Owner", "与工程确定性能预算、接口依赖和回滚开关", "让 Reviewer Agent 对每个 Requirement 的验收标准做可测试性检查"},
		},
		Chains: []EvidenceChain{
			{Source: "客户访谈/工单：结论无法定位到原始来源", Insight: "团队需要可核验的证据，而非仅有摘要。", Requirement: "REQ-001 Evidence Detail Viewer", Evidence: feedbackEvidence},
			{Source: "客户访谈/工单：冲突和陈旧资料影响判断", Insight: "不确定性必须被显式暴露。", Requirement: "REQ-002 Conflict & Gap Detection", Evidence: painAt(1)},
			{Source: "客服工单：导入字段和洞察难以复用", Insight: "原始反馈需要结构化、人工确认后才能进入需求。", Requirement: "REQ-003 Structured Feedback to Insight", Evidence: painAt(3)},
			{Source: "企业评审需要明确规则、异常和验收", Insight: "PRD 质量需要专门检查，不应只靠再次生成文档。", Requirement: "REQ-004 PRD Readiness Reviewer", Evidence: evidence},
		},
		GeneratedAt: time.Now().UTC(),
	}
}
