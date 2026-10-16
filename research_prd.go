package main

import (
	"fmt"
	"strings"
	"time"
)

func defaultDeliveryPlan() DeliveryPlan {
	return DeliveryPlan{
		Estimate: "建议 6 周交付首个受控 MVP（不含外部系统采购/审批等待）；每一阶段通过质量门后才进入下一阶段。",
		Assumptions: []string{
			"投入：1 名 Product Owner、1 名设计师（0.5 人力）、2 名工程师、1 名 QA；安全/业务 Owner 按评审节点参与。",
			"首版只覆盖一个目标用户场景和一个已授权资料源，不承诺多租户、全量集成或生产自动化。",
			"若身份权限、数据连接器、真实试点用户或安全评审未到位，工期从对应依赖确认后重新计算。",
		},
		Phases: []DeliveryPhase{
			{Window: "第 0 周（2–3 天）", Stage: "发现与立项校验", Owner: "Product + Business", Outcome: "目标用户、问题、证据、MVP 假设和停止条件", EntryGate: "研究结论已完成并可追溯", ExitGate: "指定 Owner；确认一个试点场景和不做事项", DelayRisk: "证据冲突未解或找不到目标用户"},
			{Window: "第 1 周", Stage: "方案与可行性设计", Owner: "Product + Design + Engineering + Security", Outcome: "低保真原型、数据/接口边界、技术方案和依赖清单", EntryGate: "MVP 及优先级已确认", ExitGate: "关键页面、权限、数据字典和外部依赖可评审", DelayRisk: "权限方案或连接器责任不明确"},
			{Window: "第 2 周", Stage: "PRD 基线与计划", Owner: "Product Owner + QA", Outcome: "版本化 PRD、验收标准、埋点、测试计划和排期", EntryGate: "设计与技术可行性已评审", ExitGate: "P0/P1 均有 Owner、验收、证据或明确 Gap", DelayRisk: "性能指标、合规要求或依赖未确认"},
			{Window: "第 3–4 周", Stage: "MVP 开发与联调", Owner: "Engineering", Outcome: "可演示 MVP、接口联调和审计/失败降级", EntryGate: "PRD 基线冻结", ExitGate: "核心主流程完成；阻断缺陷归零", DelayRisk: "外部系统不稳定、数据质量不足或范围膨胀"},
			{Window: "第 5 周", Stage: "QA 与试点验收", Owner: "QA + Pilot users + Security", Outcome: "功能、权限、异常、性能和业务试点证据", EntryGate: "测试环境与脱敏测试数据可用", ExitGate: "P0 用例通过；试点结果支持、修改或否定假设", DelayRisk: "测试数据/账号缺失或试点用户不可用"},
			{Window: "第 6 周", Stage: "发布评审与灰度", Owner: "Product + Operations + Security", Outcome: "Go/No-Go 决定、灰度计划、监控和回滚责任", EntryGate: "Readiness ≥ 80 且审批角色齐全", ExitGate: "监控、回滚、Owner 和复盘时间已明确", DelayRisk: "安全审批、发布窗口或关键指标未达标"},
		},
	}
}

// productAnalysisFromResearch turns one completed web-research run into a product-discovery
// package. It only promotes the run's own evidence, conflicts and limitations; it does not
// disguise public web evidence as customer interviews.
func productAnalysisFromResearch(result ResearchResult) (ProductAnalysis, error) {
	question := strings.TrimSpace(result.Question)
	if len([]rune(question)) < 8 {
		return ProductAnalysis{}, fmt.Errorf("研究问题不足，无法生成专属 PRD")
	}
	refs := evidenceRefsFromResearch(result.Evidence)
	summary := researchExcerpt(result.Report.Analysis, 520)
	if summary == "" {
		summary = "研究已完成，但报告摘要为空；只能输出待验证的问题和证据链。"
	}
	conflictText := strings.Join(result.Report.Conflicts, "；")
	if conflictText == "" {
		conflictText = "当前运行未发现明确冲突；这不等于不存在反证。"
	}
	limitations := strings.Join(result.Report.Limitations, "；")
	if limitations == "" {
		limitations = "需要在目标用户、真实付费意愿和实施约束上继续验证。"
	}
	return ProductAnalysis{
		Title:       "研究专属产品发现包",
		Scope:       question,
		DataNotice:  fmt.Sprintf("基于本次研究运行 %s：%d 条可追溯证据、%d 个候选来源、%d 项冲突。公开研究证据不等同于客户访谈。", result.RunID, len(result.Evidence), result.Stats.Discovered, result.Stats.ConflictCount),
		GeneratedAt: time.Now().UTC(),
		Summary: ProductSummary{
			Opportunity: "待验证的研究主题：" + question,
			Decision:    summary,
			Evidence:    refs,
		},
		Competition: competitorAnalysisForProduct(result),
		// Public research is not relabelled as customer voice. Customer feedback only appears
		// here after a human imports or enters it through the co-editing workflow.
		UserFeedback: []FeedbackItem{},
		DemandResearch: []DemandFinding{
			{Need: "待验证的问题假设", Finding: "研究材料支持继续探索，但尚未证明目标用户愿意切换、付费或持续使用。", Confidence: researchConfidence(result), Evidence: takeEvidenceRefs(refs, 0, 4)},
			{Need: "需要被推翻或缩小的判断", Finding: conflictText, Confidence: "待验证", Evidence: takeEvidenceRefs(refs, 1, 4)},
			{Need: "试点前必须补齐的输入", Finding: "目标用户、真实工作流、现有替代方案、价值指标、成本与数据边界。", Confidence: "低", Evidence: takeEvidenceRefs(refs, 0, 2)},
		},
		Scenarios: productScenariosFromResearch(result),
		PainPoints: []PainPoint{
			{Title: "研究结论尚未被验证为真实需求", Description: summary, Severity: "高", Evidence: takeEvidenceRefs(refs, 0, 3)},
			{Title: "冲突或反证可能改变产品判断", Description: conflictText, Severity: "高", Evidence: takeEvidenceRefs(refs, 1, 3)},
			{Title: "从机会结论到产品范围缺少可测试规格", Description: "需要明确用户、场景、MVP、数据、权限、验收和发布边界。", Severity: "中", Evidence: takeEvidenceRefs(refs, 0, 2)},
			{Title: "证据存在时效与外推限制", Description: limitations, Severity: "中", Evidence: takeEvidenceRefs(refs, 2, 3)},
		},
		FeaturePlan: []FeatureProposal{
			{Priority: "P0", Feature: "可追溯机会判断", Outcome: "让每个产品假设回到支持与反对证据。", Scope: "结论、Evidence ID、来源、时间和冲突并列。", Evidence: takeEvidenceRefs(refs, 0, 4)},
			{Priority: "P0", Feature: "假设到 MVP 的评审工作台", Outcome: "把研究结论变成明确的用户、场景、范围和验收。", Scope: "方案比较、范围、依赖和版本化 PRD。", Evidence: takeEvidenceRefs(refs, 1, 3)},
			{Priority: "P1", Feature: "试点验证与复盘", Outcome: "在投入扩张前验证用户价值、成本和安全边界。", Scope: "测试、试点反馈、监控、回滚和变更请求。", Evidence: takeEvidenceRefs(refs, 2, 3)},
		},
		PRD: ProductRequirement{
			Name:                   "PRD 草案：" + shorten(question, 72),
			Problem:                summary,
			TargetUsers:            []string{"产品负责人：判断是否立项和先做什么", "目标业务用户：在试点中验证场景与价值", "研发、设计、QA 与安全：确认可实现、可测试和可治理的范围"},
			Goals:                  []string{"将本次研究结论转化为可验证的产品机会假设", "让 MVP 范围、风险、依赖与验收可跨团队评审", "保留支持、冲突与证据限制，避免把研究推断写成事实"},
			FunctionalRequirements: []string{"建立证据到机会假设的可追溯链路", "比较方案并定义 MVP、非范围和依赖", "输出包含流程、原型、数据、接口和验收的版本化 PRD", "在真实试点前阻断没有证据、验收或责任人的高风险结论"},
			NonGoals:               []string{"不把公开研究直接等同于真实客户需求", "不自动批准立项、预算或上线", "不在缺少业务数据时编造市场规模、付费意愿或性能指标"},
			SuccessMetrics:         []string{"关键产品假设被目标用户验证的比例", "P0 需求带证据、验收和 Owner 的比例", "从研究完成到可评审 PRD 的耗时", "试点后接受、修改或否决假设的比例"},
			Risks:                  []string{conflictText, limitations, "研究问题与目标用户不匹配时，PRD 只能作为假设而非承诺。"},
			Evidence:               refs,
		},
	}, nil
}

func competitorAnalysisForProduct(result ResearchResult) CompetitorAnalysis {
	if result.Competition != nil {
		return *result.Competition
	}
	return competitorAnalysisFallback(result)
}
func evidenceRefsFromResearch(evidence []Evidence) []EvidenceRef {
	refs := make([]EvidenceRef, 0, len(evidence))
	for _, item := range evidence {
		label := strings.TrimSpace(item.SourceTitle)
		if label == "" {
			label = "公开来源"
		}
		if item.ID != "" {
			label = item.ID + " · " + label
		}
		refs = append(refs, EvidenceRef{Label: label, Type: "public_research", ID: item.ID})
	}
	if len(refs) == 0 {
		return []EvidenceRef{{Label: "本次研究未提取到足够证据，需要补充资料", Type: "gap"}}
	}
	return refs
}

func takeEvidenceRefs(refs []EvidenceRef, offset, limit int) []EvidenceRef {
	if len(refs) == 0 || limit <= 0 {
		return []EvidenceRef{{Label: "证据待补充", Type: "gap"}}
	}
	items := make([]EvidenceRef, 0, limit)
	for index := 0; index < limit && index < len(refs); index++ {
		items = append(items, refs[(offset+index)%len(refs)])
	}
	return items
}

func researchExcerpt(text string, limit int) string {
	text = strings.NewReplacer("#", "", "*", "", "`", "", "\n", " ").Replace(text)
	return shorten(strings.Join(strings.Fields(text), " "), limit)
}

func researchConfidence(result ResearchResult) string {
	if result.Stats.HighConfidence >= 3 && len(result.Evidence) >= 4 {
		return "中"
	}
	return "低"
}

func tailorDeliveryForResearch(copilot DeliveryCopilot, result ResearchResult) DeliveryCopilot {
	copilot.ResearchRunID = result.RunID
	question := strings.TrimSpace(result.Question)
	refs := evidenceRefsFromResearch(result.Evidence)
	feature := "研究机会 PRD · " + shorten(question, 58)
	copilot.PRD.Info = PRDDocumentInfo{PRDID: "RPRD-" + strings.ToUpper(shortID(result.RunID+question)), Product: "Research-to-Product Copilot", Feature: feature, Owner: "待指定 Product Owner", Reviewers: "Product · Design · Engineering · QA · Security", Status: "Draft — research-specific review required", Version: "0.1", CreatedDate: time.Now().Format("2006-01-02"), TargetRelease: "待完成目标用户验证后确定", LastUpdated: time.Now().Format("2006-01-02 15:04")}
	copilot.PRD.Background = PRDBackground{BusinessContext: "本 PRD 由一条独立研究任务生成，服务于把研究判断转为可验证产品机会。", UserProblem: "团队需要把“" + shorten(question, 100) + "”收口为明确的目标用户、场景、阻碍与代价；当前尚未完成客户验证。", CurrentSituation: "当前阶段仅完成公开研究与证据核验；尚未确认首批目标用户、真实工作流、现有替代方案、使用基线及试点 Owner。研究摘要保留在 Research 区，不再重复写入 PRD 现状。", BaselineMetrics: "当前仅有公开研究与本次证据包；目标用户数量、付费意愿、交付成本和真实使用行为必须在试点前补齐。", Evidence: refs}
	copilot.PRD.Goals = []string{"围绕“" + shorten(question, 86) + "”形成可验证的产品假设", "把证据、冲突、范围、依赖和验收汇入同一评审基线", "在投入开发前定义目标用户与试点验证动作"}
	copilot.PRD.NonGoals = []string{"不将研究报告直接当作已验证市场需求", "不自动批准预算、立项或发布", "不使用未经授权的内部数据补强结论"}
	copilot.PRD.SuccessMetrics = []SuccessMetric{
		{Name: "机会假设验证覆盖", Definition: "已指定目标用户、场景、Owner、支持/反证和停止条件的机会假设比例", Baseline: "当前未知：本次为公开研究证据，不是用户验证", Target: "第 0 周与业务 Owner 确认；未完成前不可立项", Window: "立项校验", Owner: "Product Owner + Business"},
		{Name: "P0 可交付性", Definition: "同时具备证据/Gap、验收、数据边界和责任人的 P0 需求比例", Baseline: "本 PRD 自动检查", Target: "100% P0 满足后方可提交研发评审", Window: "PRD 基线", Owner: "Product + Engineering + QA"},
		{Name: "试点决策质量", Definition: "试点结束后明确支持、修改或否定原始机会假设的比例", Baseline: "待试点采集", Target: "100% 试点完成书面复盘；不预设支持率", Window: "试点结束", Owner: "Business Owner"},
		{Name: "安全与回滚准备度", Definition: "具有授权、审计、监控和回滚责任的试点比例", Baseline: "待安全评审", Target: "100% 接入真实资料的试点满足；否则仅用脱敏数据", Window: "发布评审", Owner: "Security + Operations"},
	}
	copilot.PRD.Users = []PRDUserScenario{{Role: "产品负责人", Buyer: "业务负责人", Scenario: "将本次研究转为立项与路线图决策", Trigger: "研究报告完成并具备可追溯证据", Goal: "判断问题、用户和 MVP 是否值得投入", Evidence: takeEvidenceRefs(refs, 0, 3)}, {Role: "目标试点用户", Buyer: "业务 Owner", Scenario: "在真实工作流中验证假设", Trigger: "候选 MVP 已明确", Goal: "确认价值、可用性与替代方案", Evidence: takeEvidenceRefs(refs, 1, 3)}, {Role: "交付评审组", Buyer: "研发与治理负责人", Scenario: "评审可行性、风险和发布条件", Trigger: "PRD 草案冻结", Goal: "确认数据、权限、成本、测试和回滚边界", Evidence: takeEvidenceRefs(refs, 2, 3)}}
	copilot.Decision = ProductDecision{Goal: "验证研究问题能否形成值得投入的产品机会。", Options: []DecisionOption{{Name: "继续研究，不立项", Benefit: "降低误判风险", Tradeoff: "推迟学习与试点", Verdict: "适用于证据不足或冲突未解"}, {Name: "小范围试点 MVP", Benefit: "用真实用户验证价值与成本", Tradeoff: "需要指定 Owner 和治理边界", Verdict: "推荐：本 PRD 的默认下一步"}, {Name: "直接规模化开发", Benefit: "速度快", Tradeoff: "容易把研究推断当成事实", Verdict: "不建议在当前证据下采用"}}, Decision: "建议将本次研究作为机会假设，先用受控 MVP 试点验证，而不是直接规模化开发。", Reasoning: researchExcerpt(result.Report.Analysis, 520), MVP: []string{"证据到假设的追溯", "一个目标用户场景的端到端试点", "明确的验收、监控与回滚"}, OutOfScope: []string{"没有目标用户验证的通用平台化", "未经权限确认的数据连接", "无人工评审的高风险自动化"}, Evidence: refs}
	copilot.PRD.Requirements = researchSpecificRequirements(question, refs)
	copilot.PRD.OpenQuestions = append([]string{"谁是第一个可接触并愿意试点的目标用户？", "该用户当前如何解决问题，替代方案和切换成本是什么？", "何种试点结果会支持、削弱或否定本次机会假设？"}, result.Report.Conflicts...)
	copilot.Visuals.Title = "Research-specific Product Delivery Map"
	copilot.Readiness.Score = min(74, 60+len(result.Evidence)*3)
	copilot.Readiness.Verdict = "Not ready for engineering review — 此 PRD 已绑定本次研究，但仍需真实用户、Owner、依赖和试点数据验证。"
	copilot.Chains = []EvidenceChain{{Source: "本次研究：" + question, Insight: researchExcerpt(result.Report.Analysis, 260), Requirement: "REQ-001 Evidence to Hypothesis Workspace", Evidence: takeEvidenceRefs(refs, 0, 4)}, {Source: "本次研究冲突与限制", Insight: strings.Join(result.Report.Conflicts, "；"), Requirement: "REQ-002 Counterevidence and Validation Queue", Evidence: takeEvidenceRefs(refs, 1, 3)}, {Source: "试点验证需要", Insight: "将研究结论变成目标用户、MVP、指标与测试的可评审规格。", Requirement: "REQ-003 MVP Experiment and Review", Evidence: takeEvidenceRefs(refs, 2, 3)}}
	if result.ProductBlueprint != nil {
		copilot = applyProductBlueprint(copilot, *result.ProductBlueprint, result)
	}
	return applyResearchCoverageToPRD(copilot, result.Coverage)
}

func researchSpecificRequirements(question string, refs []EvidenceRef) []ExecutionRequirement {
	return []ExecutionRequirement{
		{ID: "REQ-001", Title: "Evidence to Hypothesis Workspace", Priority: "P0", UserStory: "作为产品负责人，我希望将“" + shorten(question, 70) + "”的结论、来源和条件转为明确机会假设，以便评审是否值得试点。", Preconditions: []string{"本次研究至少有一条可追溯证据"}, MainFlow: []string{"查看研究结论与 Evidence ID", "标记支持、反证和限制", "创建可验证机会假设", "指定假设 Owner"}, BusinessRules: []string{"任何机会假设必须保留来源和适用条件", "没有证据的内容必须标为假设而非事实"}, Exceptions: []string{"来源失效时显示最后获取时间", "证据不足时阻止标记为可立项"}, Permissions: []string{"产品 Owner 可编辑", "Reviewer 仅评论", "受限来源按访问级别显示"}, Acceptance: []string{"Given 存在研究证据 When 创建假设 Then 假设保留至少一条 Evidence ID", "Given 证据为空 When 请求立项 Then 显示待补充原因"}, Tracking: []string{"hypothesis_created", "evidence_linked", "hypothesis_blocked"}, Evidence: takeEvidenceRefs(refs, 0, 4)},
		{ID: "REQ-002", Title: "Counterevidence and Validation Queue", Priority: "P0", UserStory: "作为评审者，我希望同时看见反证、冲突和待验证问题，以便不把研究推断误写成产品承诺。", Preconditions: []string{"研究报告已生成"}, MainFlow: []string{"聚合冲突和低置信度资料", "为每项资料创建验证动作", "指派 Owner 与截止时间", "评审通过或否定假设"}, BusinessRules: []string{"反证不能在报告或 PRD 中被隐藏", "低置信度资料不能单独支持 P0 决策"}, Exceptions: []string{"没有找到反证时显示检索范围和限制"}, Permissions: []string{"指定 Reviewer 可更新验证状态"}, Acceptance: []string{"Given 有冲突来源 When 打开假设 Then 同屏显示支持与反证", "Given 证据限制存在 When 生成 PRD Then 进入开放问题"}, Tracking: []string{"counterevidence_viewed", "validation_assigned", "hypothesis_rejected"}, Evidence: takeEvidenceRefs(refs, 1, 4)},
		{ID: "REQ-003", Title: "MVP Experiment and Review", Priority: "P1", UserStory: "作为业务 Owner，我希望为候选方向定义一个可控试点，以便测量真实价值、成本和风险。", Preconditions: []string{"机会假设已由产品 Owner 确认"}, MainFlow: []string{"选择目标用户和场景", "定义 MVP、成功指标和停止条件", "完成数据、权限与测试评审", "执行试点并记录结果"}, BusinessRules: []string{"试点指标必须包含支持与否定假设的阈值", "没有回滚与数据边界不得接入真实生产资料"}, Exceptions: []string{"目标用户不可用时暂停而不推断成功"}, Permissions: []string{"业务 Owner 配置场景", "安全 Reviewer 审查资料边界"}, Acceptance: []string{"Given 试点创建 When 提交评审 Then 包含用户、指标、时限、回滚和 Owner", "Given 试点结束 When 复盘 Then 输出支持、修改或否定的决定"}, Tracking: []string{"pilot_created", "pilot_started", "pilot_outcome_recorded"}, Evidence: takeEvidenceRefs(refs, 2, 3)},
		{ID: "REQ-004", Title: "Research PRD Readiness Reviewer", Priority: "P1", UserStory: "作为交付评审者，我希望在研发投入前检查研究专属 PRD 的缺口，以便减少错误立项。", Preconditions: []string{"存在至少一条需求和一个机会假设"}, MainFlow: []string{"检查证据、目标用户、范围、数据、接口、验收和回滚", "输出具体缺口", "阻止未达门槛的提交"}, BusinessRules: []string{"研究结论不能替代用户验证", "Readiness 低于 80 不可进入研发评审"}, Exceptions: []string{"未知指标必须标为待验证，不能编造"}, Permissions: []string{"Product Owner 编辑", "各角色 Reviewer 评论或审批"}, Acceptance: []string{"Given 缺少目标用户 When 评审 Then 标记为缺失", "Given 所有 P0 有证据和验收 When 评审 Then 返回可提交状态"}, Tracking: []string{"research_prd_reviewed", "research_prd_gap_found", "research_prd_submitted"}, Evidence: takeEvidenceRefs(refs, 0, 3)},
	}
}

// applyProductBlueprint replaces the generic product-design shell only when the
// model has returned a typed, evidence-bounded blueprint. Missing fields retain the
// safe default PRD sections, so an incomplete model response cannot erase deliverables.
func applyProductBlueprint(copilot DeliveryCopilot, blueprint ProductBlueprint, result ResearchResult) DeliveryCopilot {
	question := strings.TrimSpace(result.Question)
	refs := evidenceRefsFromResearch(result.Evidence)
	if strings.TrimSpace(blueprint.ProductName) != "" {
		copilot.PRD.Info.Product = shorten(blueprint.ProductName, 96)
	}
	if strings.TrimSpace(blueprint.OneSentence) != "" {
		copilot.PRD.Info.Feature = shorten(blueprint.OneSentence, 180)
	}
	copilot.PRD.Info.Status = "Draft — model-synthesized product blueprint; human review required"
	copilot.PRD.Info.TargetRelease = "由试点范围、依赖与质量门确认后确定"
	copilot.PRD.Background = blueprintResearchBackground(copilot.PRD.Background, blueprint, question)

	if len(blueprint.Goals) > 0 {
		copilot.PRD.Goals = blueprint.Goals
	}
	if len(blueprint.NonGoals) > 0 {
		copilot.PRD.NonGoals = blueprint.NonGoals
	}
	if len(blueprint.SuccessMetrics) > 0 {
		copilot.PRD.SuccessMetrics = blueprint.SuccessMetrics
	}
	if len(blueprint.Users) > 0 {
		copilot.PRD.Users = blueprint.Users
	}
	if len(blueprint.Options) > 0 {
		copilot.Decision.Options = blueprint.Options
	}
	if strings.TrimSpace(blueprint.Decision) != "" {
		copilot.Decision.Decision = blueprint.Decision
	}
	if strings.TrimSpace(blueprint.Reasoning) != "" {
		copilot.Decision.Reasoning = blueprint.Reasoning
	}
	if len(blueprint.MVP) > 0 {
		copilot.Decision.MVP = blueprint.MVP
	}
	if len(blueprint.OutOfScope) > 0 {
		copilot.Decision.OutOfScope = blueprint.OutOfScope
	}
	copilot.Decision.Goal = "围绕“" + shorten(question, 96) + "”验证模型生成的产品方案是否可被真实用户接受。"
	copilot.Decision.Evidence = refs

	if len(blueprint.UserFlow) > 0 {
		copilot.PRD.UserFlow = blueprint.UserFlow
		copilot.Visuals.ProductFlow = blueprintFlowNodes("U", blueprint.UserFlow, "Product / User")
	}
	if len(blueprint.SystemFlow) > 0 {
		copilot.PRD.SystemFlow = blueprint.SystemFlow
		copilot.Visuals.SolutionFlow = blueprintFlowNodes("S", blueprint.SystemFlow, "System / Engineering")
	}
	if len(blueprint.Permissions) > 0 {
		copilot.PRD.Permissions = blueprint.Permissions
	}
	if len(blueprint.Requirements) > 0 {
		copilot.PRD.Requirements = blueprint.Requirements
		copilot.PRD.Architecture = blueprintArchitecture(blueprint.Requirements)
	}
	if len(blueprint.Prototype.Screens) > 0 {
		copilot.PRD.Prototype = blueprint.Prototype
		copilot.Solution.Pages = make([]string, 0, len(blueprint.Prototype.Screens))
		for _, screen := range blueprint.Prototype.Screens {
			copilot.Solution.Pages = append(copilot.Solution.Pages, screen.Name+" · "+screen.Purpose)
		}
	}
	if len(blueprint.DataModel) > 0 {
		copilot.Solution.DataModel = blueprint.DataModel
		copilot.Solution.DataEntities = make([]string, 0, len(blueprint.DataModel))
		for _, entity := range blueprint.DataModel {
			copilot.Solution.DataEntities = append(copilot.Solution.DataEntities, entity.Name+" · "+entity.Purpose)
		}
	}
	if len(blueprint.DataDictionary) > 0 {
		copilot.Solution.DataDictionary = blueprint.DataDictionary
	}
	if len(blueprint.APIContracts) > 0 {
		copilot.Solution.APIContracts = blueprint.APIContracts
		copilot.Solution.APIs = make([]string, 0, len(blueprint.APIContracts))
		for _, api := range blueprint.APIContracts {
			copilot.Solution.APIs = append(copilot.Solution.APIs, api.Method+" "+api.Path+" · "+api.Name)
		}
	}
	if len(blueprint.Dependencies) > 0 {
		copilot.Solution.Dependencies = blueprint.Dependencies
	}
	if len(blueprint.NonFunctional) > 0 {
		copilot.PRD.NonFunctional = blueprint.NonFunctional
	}
	if len(blueprint.SOP) > 0 {
		copilot.Visuals.SOP = blueprint.SOP
	}
	if len(blueprint.TestPlan.Cases) > 0 {
		copilot.Visuals.TestPlan = blueprint.TestPlan
	}
	if len(blueprint.DeliveryPlan.Phases) > 0 {
		copilot.PRD.DeliveryPlan = blueprint.DeliveryPlan
	}
	if len(blueprint.OpenQuestions) > 0 {
		copilot.PRD.OpenQuestions = append(blueprint.OpenQuestions, result.Report.Conflicts...)
	}
	if len(blueprint.Tracking) > 0 {
		copilot.Evaluation.Tracking = blueprint.Tracking
	}
	if len(blueprint.TestFocus) > 0 {
		copilot.Evaluation.TestFocus = blueprint.TestFocus
	}
	if len(blueprint.LaunchChecklist) > 0 {
		copilot.Evaluation.LaunchChecklist = blueprint.LaunchChecklist
	}
	if len(blueprint.Monitoring) > 0 {
		copilot.Evaluation.Monitoring = blueprint.Monitoring
	}
	if len(blueprint.Rollback) > 0 {
		copilot.Evaluation.Rollback = blueprint.Rollback
	}

	copilot.Discovery.Title = "研究专属产品发现包 · " + shorten(copilot.PRD.Info.Product, 52)
	copilot.Discovery.Summary = ProductSummary{Opportunity: blueprint.OneSentence, Decision: copilot.Decision.Decision, Evidence: refs}
	copilot.Discovery.DataNotice = "本 PRD 的用户、功能、原型、数据、测试和交付方案由当前研究证据包进行模型合成；没有证据支撑的项目均应作为假设，在访谈或试点中验证。"
	copilot.Visuals.Title = shorten(copilot.PRD.Info.Product, 56) + " · Delivery Map"
	copilot.Readiness.Score = min(76, 58+len(copilot.PRD.Requirements)*4)
	copilot.Readiness.Verdict = "Draft with product-specific content — 仍需由目标用户、设计、研发、QA 与安全共同确认后进入研发评审。"
	copilot.Chains = make([]EvidenceChain, 0, len(copilot.PRD.Requirements))
	for _, requirement := range copilot.PRD.Requirements {
		copilot.Chains = append(copilot.Chains, EvidenceChain{Source: "当前研究：" + question, Insight: shorten(blueprint.OneSentence, 260), Requirement: requirement.ID + " " + requirement.Title, Evidence: requirement.Evidence})
	}
	return applyResearchCoverageToPRD(copilot, result.Coverage)
}

// Describe the proposed product using generated fields, while keeping its proposed
// workflow separate from customer operations that this research has not observed.
func blueprintResearchBackground(background PRDBackground, blueprint ProductBlueprint, question string) PRDBackground {
	context := []string{"本次研究范围：" + question}
	if name := strings.TrimSpace(blueprint.ProductName); name != "" {
		context = append(context, "候选产品："+name)
	}
	if proposition := strings.TrimSpace(blueprint.OneSentence); proposition != "" {
		context = append(context, "待验证的产品主张："+proposition)
	}
	if reasoning := strings.TrimSpace(blueprint.Reasoning); reasoning != "" {
		context = append(context, "方案推导（待评审）："+reasoning)
	}
	background.BusinessContext = strings.Join(context, "；")
	problems := make([]string, 0, len(blueprint.Users))
	for _, user := range blueprint.Users {
		parts := make([]string, 0, 3)
		if strings.TrimSpace(user.Role) != "" {
			parts = append(parts, "候选用户："+user.Role)
		}
		if strings.TrimSpace(user.Scenario) != "" {
			parts = append(parts, "场景："+user.Scenario)
		}
		if strings.TrimSpace(user.Goal) != "" {
			parts = append(parts, "待验证目标："+user.Goal)
		}
		if len(parts) > 0 {
			problems = append(problems, strings.Join(parts, "，"))
		}
	}
	if len(problems) > 0 {
		background.UserProblem = strings.Join(problems, "；") + "。以上为蓝图中的用户与问题假设，仍需客户访谈或试点验证。"
	}
	situation := []string{"当前已形成“" + strings.TrimSpace(blueprint.ProductName) + "”候选方案草案"}
	if len(blueprint.MVP) > 0 {
		situation = append(situation, "拟验证范围："+strings.Join(blueprint.MVP, "、"))
	}
	if len(blueprint.UserFlow) > 0 {
		situation = append(situation, "拟议用户流程："+strings.Join(blueprint.UserFlow, " → "))
	}
	situation = append(situation, "客户当前使用的工具、实际操作步骤、处理耗时与发生频率尚未由本次研究验证；上述拟议流程不代表客户现行业务事实")
	background.CurrentSituation = strings.Join(situation, "；")
	return background
}

// Persist coverage boundaries in fields already rendered by the UI and DOCX exporter.
// A zero-value coverage belongs to an older task; it must not be called insufficient.
func applyResearchCoverageToPRD(copilot DeliveryCopilot, coverage ResearchCoverage) DeliveryCopilot {
	if coverage.Status == "" && coverage.StopReason == "" && len(coverage.Gaps) == 0 {
		return copilot
	}
	status := "未记录覆盖状态"
	switch coverage.Status {
	case "sufficient":
		status = "已达到来源与维度覆盖门槛，仍需核验具体结论并验证真实客户需求"
	case "insufficient":
		status = "证据覆盖不足，当前 PRD 为待补证的产品假设草案"
	}
	boundary := "研究评审边界：" + status
	if coverage.StopReason != "" {
		reasons := map[string]string{"coverage_target": "已达到本次研究覆盖目标", "time_budget": "达到研究时间预算", "source_budget": "达到来源读取预算", "no_new_evidence": "继续搜索未发现新增证据", "round_budget": "达到搜索轮次预算", "query_budget": "达到搜索查询预算"}
		reason := reasons[coverage.StopReason]
		if reason == "" {
			reason = coverage.StopReason
		}
		boundary += "；停止原因：" + reason
	}
	copilot.PRD.OpenQuestions = append(copilot.PRD.OpenQuestions, boundary)
	copilot.Solution.TechnicalConstraints = append(copilot.Solution.TechnicalConstraints, boundary)
	for _, gap := range coverage.Gaps {
		if gap = strings.TrimSpace(gap); gap != "" {
			copilot.PRD.OpenQuestions = append(copilot.PRD.OpenQuestions, "研究补证："+gap)
		}
	}
	copilot.PRD.OpenQuestions = uniqueStrings(copilot.PRD.OpenQuestions)
	copilot.Solution.TechnicalConstraints = uniqueStrings(copilot.Solution.TechnicalConstraints)
	return copilot
}

func blueprintFlowNodes(prefix string, steps []string, owner string) []VisualFlowNode {
	nodes := make([]VisualFlowNode, 0, len(steps))
	for index, step := range steps {
		nodes = append(nodes, VisualFlowNode{ID: fmt.Sprintf("%s%d", prefix, index+1), Title: shorten(step, 84), Owner: owner, Output: "下一阶段可执行输入"})
	}
	return nodes
}

func blueprintArchitecture(requirements []ExecutionRequirement) []ArchitectureModule {
	modules := make([]ArchitectureModule, 0, len(requirements))
	for _, requirement := range requirements {
		children := append([]string(nil), requirement.MainFlow...)
		if len(children) > 3 {
			children = children[:3]
		}
		modules = append(modules, ArchitectureModule{Name: requirement.ID + " · " + requirement.Title, Children: children})
	}
	return modules
}
