package main

import (
	"fmt"
	"time"
)

type ProductAnalysis struct {
	Title          string             `json:"title"`
	Scope          string             `json:"scope"`
	DataNotice     string             `json:"dataNotice"`
	GeneratedAt    time.Time          `json:"generatedAt"`
	Summary        ProductSummary     `json:"summary"`
	Competition    CompetitorAnalysis `json:"competition"`
	UserFeedback   []FeedbackItem     `json:"userFeedback"`
	DemandResearch []DemandFinding    `json:"demandResearch"`
	Scenarios      []ProductScenario  `json:"scenarios"`
	PainPoints     []PainPoint        `json:"painPoints"`
	FeaturePlan    []FeatureProposal  `json:"featurePlan"`
	PRD            ProductRequirement `json:"prd"`
}

type ProductSummary struct {
	Opportunity string        `json:"opportunity"`
	Decision    string        `json:"decision"`
	Evidence    []EvidenceRef `json:"evidence"`
}

type EvidenceRef struct {
	Label string `json:"label"`
	Type  string `json:"type"`
	ID    string `json:"id"`
}

type FeedbackItem struct {
	Audience string        `json:"audience"`
	Feedback string        `json:"feedback"`
	Evidence []EvidenceRef `json:"evidence"`
}

type DemandFinding struct {
	Need       string        `json:"need"`
	Finding    string        `json:"finding"`
	Confidence string        `json:"confidence"`
	Evidence   []EvidenceRef `json:"evidence"`
}

type ProductScenario struct {
	ID               string                 `json:"id,omitempty"`
	Name             string                 `json:"name"`
	Actor            string                 `json:"actor"`
	Trigger          string                 `json:"trigger"`
	CurrentGap       string                 `json:"currentGap"`
	Outcome          string                 `json:"outcome"`
	Evidence         []EvidenceRef          `json:"evidence"`
	Context          string                 `json:"context,omitempty"`
	Inputs           []string               `json:"inputs,omitempty"`
	CurrentWorkflow  []string               `json:"currentWorkflow,omitempty"`
	Impact           string                 `json:"impact,omitempty"`
	ProposedWorkflow []string               `json:"proposedWorkflow,omitempty"`
	Acceptance       []string               `json:"acceptance,omitempty"`
	Exceptions       []string               `json:"exceptions,omitempty"`
	Validation       string                 `json:"validation,omitempty"`
	Basis            string                 `json:"basis,omitempty"`
	EvidenceLinks    []ScenarioEvidenceLink `json:"evidenceLinks,omitempty"`
}

type ScenarioEvidenceLink struct {
	ID          string `json:"id"`
	Field       string `json:"field"`
	Quote       string `json:"quote"`
	Explanation string `json:"explanation"`
	SourceTitle string `json:"sourceTitle,omitempty"`
	URL         string `json:"url,omitempty"`
	PublishedAt string `json:"publishedAt,omitempty"`
}

type PainPoint struct {
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Severity    string        `json:"severity"`
	Evidence    []EvidenceRef `json:"evidence"`
}

type FeatureProposal struct {
	Priority string        `json:"priority"`
	Feature  string        `json:"feature"`
	Outcome  string        `json:"outcome"`
	Scope    string        `json:"scope"`
	Evidence []EvidenceRef `json:"evidence"`
}

type ProductRequirement struct {
	Name                   string        `json:"name"`
	Problem                string        `json:"problem"`
	TargetUsers            []string      `json:"targetUsers"`
	Goals                  []string      `json:"goals"`
	FunctionalRequirements []string      `json:"functionalRequirements"`
	NonGoals               []string      `json:"nonGoals"`
	SuccessMetrics         []string      `json:"successMetrics"`
	Risks                  []string      `json:"risks"`
	Evidence               []EvidenceRef `json:"evidence"`
}

type productInterview struct {
	ID, Persona, Scenario, Transcript string
}

type productTicket struct {
	ID, ExternalID, Message, Category, Resolution string
}

// BuildProductAnalysis derives a traceable product-discovery package from the local structured tables.
// The response deliberately carries evidence labels rather than presenting invented customer claims.
func (s *KnowledgeStore) BuildProductAnalysis() (ProductAnalysis, error) {
	status := s.Status()
	if status.Interviews == 0 && status.SupportTickets == 0 && status.ProductRequirements == 0 {
		return ProductAnalysis{}, fmt.Errorf("还没有足够的内部结构化资料；请先导入访谈、工单或点击“加载 AI Agent 模拟样本”")
	}

	interviews, err := s.productInterviews()
	if err != nil {
		return ProductAnalysis{}, err
	}
	tickets, err := s.productTickets()
	if err != nil {
		return ProductAnalysis{}, err
	}

	interviewRef := func(index int) EvidenceRef {
		if index >= 0 && index < len(interviews) {
			return EvidenceRef{Label: "访谈 · " + interviews[index].Persona, Type: "interview", ID: interviews[index].ID}
		}
		return EvidenceRef{Label: "访谈资料待补充", Type: "gap", ID: ""}
	}
	ticketRef := func(category, fallback string) EvidenceRef {
		for _, ticket := range tickets {
			if ticket.Category == category {
				label := ticket.ExternalID
				if label == "" {
					label = ticket.ID
				}
				return EvidenceRef{Label: "工单 " + label + " · " + ticket.Message, Type: "support_ticket", ID: ticket.ID}
			}
		}
		return EvidenceRef{Label: fallback, Type: "gap", ID: ""}
	}
	prdRef := EvidenceRef{Label: "本地 PRD 资料", Type: "product_requirement", ID: ""}
	var prdSummary string
	_ = s.db.QueryRow(`SELECT id, summary FROM product_requirements ORDER BY created_at DESC LIMIT 1`).Scan(&prdRef.ID, &prdSummary)
	if prdRef.ID != "" {
		prdRef.Label = "PRD · " + prdSummary
	}

	dataNotice := fmt.Sprintf("基于本地 %d 份文档、%d 条工单、%d 份访谈与 %d 条洞察生成。", status.Documents, status.SupportTickets, status.Interviews, status.Insights)
	var demoCount int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM documents WHERE source = ?`, demoKnowledgeSource).Scan(&demoCount)
	if demoCount > 0 {
		dataNotice += " 其中包含明确标注为“模拟”的演示资料：下列方案只能验证流程，不能被解释为真实市场结论。"
	}

	analysis := ProductAnalysis{
		Title:       "AI Agent 产品发现包",
		Scope:       "将内部用户反馈、客服问题和现有 PRD 转换为可审阅的需求调研、场景、功能优先级与 PRD 草案。",
		DataNotice:  dataNotice,
		GeneratedAt: time.Now().UTC(),
		Summary: ProductSummary{
			Opportunity: "把 AI Agent 从“能回答”推进到“能被团队信任”：让结论回到来源、让冲突和低置信度可见、让权限与审计成为产品能力。",
			Decision:    "优先验证面向产品/客服洞察团队的“可审计研究工作台”，而不是先做泛化 Agent 平台。",
			Evidence:    []EvidenceRef{interviewRef(0), interviewRef(1), ticketRef("citation", "缺少引用链工单样本"), ticketRef("access_control", "缺少权限工单样本"), prdRef},
		},
		UserFeedback: []FeedbackItem{
			{Audience: audienceOf(interviews, 0, "客服运营负责人"), Feedback: transcriptOf(interviews, 0, "需要把工单洞察回到原始对话，并保留人工审批。"), Evidence: []EvidenceRef{interviewRef(0), ticketRef("citation", "缺少引用链工单样本")}},
			{Audience: audienceOf(interviews, 1, "产品负责人"), Feedback: transcriptOf(interviews, 1, "路线图评审需要同时看见支持、冲突与证据缺口。"), Evidence: []EvidenceRef{interviewRef(1), ticketRef("conflict", "缺少冲突工单样本")}},
			{Audience: audienceOf(interviews, 2, "IT 安全负责人"), Feedback: transcriptOf(interviews, 2, "扩大试点前，需要最小权限、资料来源与数据保留策略。"), Evidence: []EvidenceRef{interviewRef(2), ticketRef("audit", "缺少审计工单样本")}},
		},
		DemandResearch: []DemandFinding{
			{Need: "可信的结论交付", Finding: "用户的核心不是再得到一个摘要，而是验证摘要依据、反例和不确定性。", Confidence: confidenceLabel(status.Interviews + status.SupportTickets), Evidence: []EvidenceRef{interviewRef(0), interviewRef(1), ticketRef("citation", "待补充引用问题")}},
			{Need: "受控的内部知识调用", Finding: "权限、审计、时效性与资料撤销直接影响企业是否允许 Agent 进入生产流程。", Confidence: confidenceLabel(status.Interviews + status.SupportTickets), Evidence: []EvidenceRef{interviewRef(2), ticketRef("access_control", "待补充权限问题"), ticketRef("freshness", "待补充时效性问题")}},
			{Need: "稳定的团队工作流", Finding: "成本上限、导入映射和人工审批要作为产品流程，而不是异常处理。", Confidence: "中", Evidence: []EvidenceRef{ticketRef("cost_control", "待补充成本问题"), ticketRef("ingestion", "待补充导入问题"), prdRef}},
		},
		Scenarios: []ProductScenario{
			{Name: "机会假设评审", Actor: "产品负责人", Trigger: "准备 AI Agent 路线图或创业机会评审", CurrentGap: "资料散落且结论不可回溯，反证容易被忽略。", Outcome: "输入问题后生成支持/冲突/缺口三栏，并能点击回到内部或外部证据。", Evidence: []EvidenceRef{interviewRef(1), ticketRef("conflict", "待补充冲突证据"), prdRef}},
			{Name: "客服问题洞察", Actor: "客服运营负责人", Trigger: "每周复盘高频工单", CurrentGap: "报表只有数量，没有原文、类别和可行动建议。", Outcome: "导入工单后聚类问题、展示原始片段，并生成需人工确认的行动建议。", Evidence: []EvidenceRef{interviewRef(0), ticketRef("citation", "待补充引用问题"), ticketRef("ingestion", "待补充导入问题")}},
			{Name: "企业试点安全评审", Actor: "IT 安全负责人", Trigger: "申请扩大 Agent 可访问的数据范围", CurrentGap: "来源、权限和资料生命周期不可审计。", Outcome: "以资料访问级别、使用记录与过期状态组成审批包。", Evidence: []EvidenceRef{interviewRef(2), ticketRef("access_control", "待补充权限问题"), ticketRef("audit", "待补充审计问题")}},
		},
		PainPoints: []PainPoint{
			{Title: "结论无法回到原始证据", Description: "摘要或研究结论无法快速定位到具体网页、访谈片段或工单。", Severity: "高", Evidence: []EvidenceRef{interviewRef(0), interviewRef(1), ticketRef("citation", "待补充引用问题")}},
			{Title: "冲突、时效性与低置信度被隐藏", Description: "少量、过时或相互矛盾的资料容易被合成成看似确定的判断。", Severity: "高", Evidence: []EvidenceRef{interviewRef(1), ticketRef("conflict", "待补充冲突问题"), ticketRef("freshness", "待补充时效性问题")}},
			{Title: "企业治理进入产品太晚", Description: "权限、审计与导入边界若只在部署时处理，会阻塞试点扩大。", Severity: "高", Evidence: []EvidenceRef{interviewRef(2), ticketRef("access_control", "待补充权限问题"), ticketRef("audit", "待补充审计问题")}},
			{Title: "从原始资料到行动方案断裂", Description: "工单与访谈缺少结构化字段、责任人和验证动作，难以形成闭环。", Severity: "中", Evidence: []EvidenceRef{interviewRef(0), ticketRef("ingestion", "待补充导入问题")}},
		},
		FeaturePlan: []FeatureProposal{
			{Priority: "P0", Feature: "Claim ↔ Evidence Workspace", Outcome: "每个关键结论可查看原文、来源、时间、冲突资料和置信度。", Scope: "证据卡、支持/冲突关联、低置信度提醒、DOCX 引用附件。", Evidence: []EvidenceRef{interviewRef(0), interviewRef(1), ticketRef("citation", "待补充引用问题"), ticketRef("conflict", "待补充冲突问题")}},
			{Priority: "P0", Feature: "内部知识权限与审计层", Outcome: "让企业可以在可控范围内把访谈、工单和 PRD 交给 Agent。", Scope: "文档访问级别、资料状态、读取记录、过期资料提示。", Evidence: []EvidenceRef{interviewRef(2), ticketRef("access_control", "待补充权限问题"), ticketRef("audit", "待补充审计问题")}},
			{Priority: "P1", Feature: "客服工单到需求洞察管道", Outcome: "把重复问题转成带原始依据的痛点、需求假设和待验证事项。", Scope: "CSV 字段映射、问题聚类、主题趋势、人工确认后写入洞察。", Evidence: []EvidenceRef{interviewRef(0), ticketRef("ingestion", "待补充导入问题")}},
			{Priority: "P1", Feature: "研究运行控制台", Outcome: "减少长问题的检索成本并防止陈旧资料影响判断。", Scope: "搜索预算、时间过滤、失败回退、资料有效期和运行日志。", Evidence: []EvidenceRef{ticketRef("cost_control", "待补充成本问题"), ticketRef("freshness", "待补充时效性问题")}},
		},
		PRD: ProductRequirement{
			Name:                   "PRD 草案：可审计 AI Agent 研究工作台 v0.1",
			Problem:                "团队在市场研究、用户洞察和路线图评审中无法判断结论来自何处，也缺少对冲突、权限和时效性的治理。",
			TargetUsers:            []string{"产品负责人：准备机会/路线图评审", "客服运营负责人：从工单提炼可验证问题", "IT 安全负责人：评估内部资料访问与审计边界"},
			Goals:                  []string{"让关键结论回到可点击的内部或外部证据", "让支持、冲突与证据缺口同时可见", "让内部资料具备版本、状态、访问级别与可追溯分块"},
			FunctionalRequirements: []string{"研究任务显示计划、搜索、阅读、证据提取和报告", "每条证据显示原文片段、来源、时间、相关性和置信度", "支持导入访谈、工单、研究笔记、PRD 和结构化数据", "工单可形成待人工确认的洞察与产品需求", "导出报告时保留引用与冲突说明"},
			NonGoals:               []string{"不自动执行高风险业务决策", "不将模拟数据或单一资料包装成市场事实", "v0.1 不承诺自动完成所有领域的深度调研"},
			SuccessMetrics:         []string{"关键结论中可回溯引用的占比", "从工单导入到确认洞察的完成率", "人工接受/修改/拒绝建议的比例", "过期或冲突资料被主动标识的比例"},
			Risks:                  []string{"模拟资料容易被误读为真实客户声音", "检索质量受资料元数据和解析质量影响", "权限与审计若不先行会阻塞企业部署"},
			Evidence:               []EvidenceRef{prdRef, interviewRef(0), interviewRef(1), interviewRef(2), ticketRef("citation", "待补充引用问题")},
		},
	}
	return analysis, nil
}

func (s *KnowledgeStore) productInterviews() ([]productInterview, error) {
	rows, err := s.db.Query(`SELECT id, persona, scenario, transcript FROM interviews ORDER BY created_at ASC LIMIT 12`)
	if err != nil {
		return nil, fmt.Errorf("读取访谈资料失败：%w", err)
	}
	defer rows.Close()
	items := []productInterview{}
	for rows.Next() {
		var item productInterview
		if err := rows.Scan(&item.ID, &item.Persona, &item.Scenario, &item.Transcript); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *KnowledgeStore) productTickets() ([]productTicket, error) {
	rows, err := s.db.Query(`SELECT id, external_ticket_id, user_message, category, resolution FROM support_tickets ORDER BY occurred_at ASC, created_at ASC LIMIT 80`)
	if err != nil {
		return nil, fmt.Errorf("读取客服工单失败：%w", err)
	}
	defer rows.Close()
	items := []productTicket{}
	for rows.Next() {
		var item productTicket
		if err := rows.Scan(&item.ID, &item.ExternalID, &item.Message, &item.Category, &item.Resolution); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func audienceOf(items []productInterview, index int, fallback string) string {
	if index < len(items) && items[index].Persona != "" {
		return items[index].Persona
	}
	return fallback + "（待真实访谈验证）"
}
func transcriptOf(items []productInterview, index int, fallback string) string {
	if index < len(items) && items[index].Transcript != "" {
		return items[index].Transcript
	}
	return fallback
}
func confidenceLabel(n int) string {
	if n >= 10 {
		return "中"
	}
	return "低"
}
