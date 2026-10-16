package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const demoKnowledgeSource = "demo_synthetic"

type DemoSeedResult struct {
	Created bool            `json:"created"`
	Status  KnowledgeStatus `json:"status"`
}

type demoDocument struct {
	key        string
	filename   string
	content    string
	title      string
	docType    string
	department string
}

// SeedAIAgentDemo creates fictional data for exercising the local knowledge base.
// It is explicitly demo data, never web-research evidence or a customer record.
func (s *KnowledgeStore) SeedAIAgentDemo() (DemoSeedResult, error) {
	seeds := []demoDocument{
		{key: "market_note", filename: "simulated-ai-agent-market-note.md", title: "[模拟] 企业 AI Agent 市场研究笔记", docType: "research_note", department: "strategy", content: "# 模拟研究笔记：企业 AI Agent\n\n> 数据性质：这是为产品演示生成的虚构资料，不是市场事实、客户证词或投资建议。\n\n## 观察假设\n\n- 企业采购团队在意的不只是模型能力，还在意权限边界、可观测性和可复核的交付结果。\n- 研究型 Agent 的机会可能集中在高价值决策、证据溯源和团队协作，而非泛化问答。\n- 成功试点需要明确的人审节点、失败回退与成本上限。\n\n## 待验证问题\n\n- 哪些行业愿意为可审计的研究报告付费？\n- 客户会把引用链还是自动执行视为第一购买理由？\n"},
		{key: "interview_ops", filename: "simulated-interview-support-ops.md", title: "[模拟] 客服运营负责人访谈", docType: "user_interview", department: "customer_operations", content: "# 模拟客户访谈：客服运营负责人\n\n> 数据性质：虚构角色与访谈，仅用于演示检索和洞察结构。\n\n角色：200 人客服团队的运营负责人。\n\n痛点：每天需要从工单中归纳反复出现的问题；现有报表只能统计数量，不能展示支持某个结论的原始对话。\n\n期望：希望 Agent 给出问题聚类、引用片段和低置信度提醒；在回答前保留人工审批。\n\n购买障碍：担心客服对话中含有个人信息，且担心模型把少量个案扩写成趋势。\n"},
		{key: "interview_product", filename: "simulated-interview-product-lead.md", title: "[模拟] 产品负责人访谈", docType: "user_interview", department: "product", content: "# 模拟客户访谈：产品负责人\n\n> 数据性质：虚构角色与访谈，仅用于演示检索和洞察结构。\n\n场景：产品负责人每周整理竞品、访谈与工单，准备路线图评审。\n\n痛点：资料散落在文档、表格和聊天记录；团队很难判断一条结论是否来自真实来源。\n\n期望：输入一个机会假设后，系统能找出支持证据、冲突证据和仍需访谈的问题。\n\n成功标准：一页摘要可点击回到具体资料；低置信度不被包装成确定结论。\n"},
		{key: "interview_it", filename: "simulated-interview-it-security.md", title: "[模拟] IT 安全负责人访谈", docType: "user_interview", department: "it_security", content: "# 模拟客户访谈：IT 安全负责人\n\n> 数据性质：虚构角色与访谈，仅用于演示检索和洞察结构。\n\n关切：Agent 访问内部资料时需要最小权限、来源记录和数据保留策略。\n\n期望：每一个外部网页与内部文档都能标记来源、时间和访问级别；支持撤销失效资料。\n\n风险：若系统无法解释为什么引用某段资料，安全团队不会批准扩大试点。\n"},
		{key: "tickets", filename: "simulated-agent-support-tickets.csv", title: "[模拟] AI Agent 客服工单数据集", docType: "support_export", department: "customer_operations", content: "ticket_id,role,category,summary,priority,resolution\nT-001,研究分析师,引用链,报告无法定位到原网页,high,增加证据卡链接\nT-002,运营经理,权限,团队成员看不到共享资料,high,检查角色策略\nT-003,产品经理,成本,长问题触发过多检索,medium,显示预算上限\nT-004,客服主管,时效性,旧资料仍出现在摘要中,medium,增加资料日期筛选\nT-005,研究分析师,冲突,不同来源给出相反结论,medium,显示冲突证据\nT-006,IT 管理员,审计,无法导出访问记录,high,增加审计日志导出\nT-007,产品经理,格式,报告里的标题显示异常,low,规范 Markdown 渲染\nT-008,运营经理,导入,CSV 字段没有自动识别,medium,提供字段映射\n"},
		{key: "prd", filename: "simulated-prd-evidence-workspace.md", title: "[模拟] PRD：可审计研究工作台 v0.1", docType: "prd", department: "product", content: "# [模拟] PRD：可审计研究工作台 v0.1\n\n> 数据性质：演示用虚构需求，不代表已承诺的路线图。\n\n## 问题\n\n团队在做市场判断时，结论与原始来源脱节，无法快速检查反例。\n\n## 目标\n\n1. 用户提交研究问题后看见计划、搜索、阅读和证据提取过程。\n2. 每条结论可回到来源片段，并显示时间、相关性与置信度。\n3. 内部资料支持版本、访问级别和失效状态。\n\n## 非目标\n\n不在 v0.1 自动替用户作出投资或人事决策；不把模拟资料当成外部事实。\n\n## 验收标准\n\n- 报告中的至少 80% 关键结论附带可点击的证据引用。\n- 搜索结果能区分支持、冲突和证据缺口。\n- 使用者可导出 DOCX，并保留来源列表。\n"},
		{key: "dataset", filename: "simulated-agent-pilot-metrics.csv", title: "[模拟] AI Agent 试点指标结构化数据集", docType: "structured_dataset", department: "product_analytics", content: "week,workspace,questions_submitted,source_read_rate,evidence_coverage,human_acceptance_rate,notes\n2026-W01,市场研究,42,0.61,0.73,0.64,模拟基线\n2026-W02,市场研究,47,0.68,0.77,0.69,增加冲突检查\n2026-W03,客服洞察,35,0.58,0.71,0.60,导入工单样本\n2026-W04,客服洞察,38,0.66,0.79,0.72,增加引用链接\n"},
	}

	documentIDs := make(map[string]string, len(seeds))
	created := false
	for _, seed := range seeds {
		var existingID string
		err := s.db.QueryRow(`SELECT id FROM documents WHERE source = ? AND original_name = ? LIMIT 1`, demoKnowledgeSource, seed.filename).Scan(&existingID)
		if err == nil {
			documentIDs[seed.key] = existingID
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return DemoSeedResult{}, fmt.Errorf("检查模拟文档失败：%w", err)
		}
		document, err := s.Ingest(seed.filename, []byte(seed.content), KnowledgeMetadata{Source: demoKnowledgeSource, Title: seed.title, DocumentType: seed.docType, Department: seed.department, Version: "demo-1", Status: "active", AccessLevel: "demo"})
		if err != nil {
			return DemoSeedResult{}, err
		}
		documentIDs[seed.key] = document.ID
		created = true
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.Begin()
	if err != nil {
		return DemoSeedResult{}, err
	}
	rollback := func(cause error) (DemoSeedResult, error) { _ = tx.Rollback(); return DemoSeedResult{}, cause }
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT OR IGNORE INTO research_notes (id,document_id,title,topic,key_findings_json,open_questions_json,confidence,created_at) VALUES (?,?,?,?,?,?,?,?)`, []any{"demo-note-001", documentIDs["market_note"], "[模拟] 企业 AI Agent 市场研究笔记", "enterprise_agent", `["可靠性与可审计性可能比通用问答更重要","需验证付费意愿"]`, `["哪些行业购买？","引用链的支付意愿如何？"]`, 0.35, now}},
		{`INSERT OR IGNORE INTO interviews (id,document_id,persona,scenario,transcript,occurred_at,created_at) VALUES (?,?,?,?,?,?,?)`, []any{"demo-interview-ops", documentIDs["interview_ops"], "客服运营负责人（模拟）", "工单洞察周报", "希望保留原始引用与人工审批。", "2026-10-01", now}},
		{`INSERT OR IGNORE INTO interviews (id,document_id,persona,scenario,transcript,occurred_at,created_at) VALUES (?,?,?,?,?,?,?)`, []any{"demo-interview-product", documentIDs["interview_product"], "产品负责人（模拟）", "路线图评审", "需要同时看到支持与冲突资料。", "2026-10-02", now}},
		{`INSERT OR IGNORE INTO interviews (id,document_id,persona,scenario,transcript,occurred_at,created_at) VALUES (?,?,?,?,?,?,?)`, []any{"demo-interview-security", documentIDs["interview_it"], "IT 安全负责人（模拟）", "试点安全评估", "需要最小权限、来源与数据保留策略。", "2026-10-03", now}},
		{`INSERT OR IGNORE INTO insights (id,interview_id,pain_points_json,behaviors_json,needs_json,evidence_chunk_ids_json,confidence,created_at) VALUES (?,?,?,?,?,?,?,?)`, []any{"demo-insight-001", "demo-interview-ops", `["工单总结缺少可核查原文"]`, `["人工复核摘要"]`, `["引用片段","审批节点"]`, `[]`, 0.60, now}},
		{`INSERT OR IGNORE INTO insights (id,interview_id,pain_points_json,behaviors_json,needs_json,evidence_chunk_ids_json,confidence,created_at) VALUES (?,?,?,?,?,?,?,?)`, []any{"demo-insight-002", "demo-interview-product", `["信息分散","结论不可追溯"]`, `["每周人工整理资料"]`, `["支持/冲突并列","证据地图"]`, `[]`, 0.58, now}},
		{`INSERT OR IGNORE INTO insights (id,interview_id,pain_points_json,behaviors_json,needs_json,evidence_chunk_ids_json,confidence,created_at) VALUES (?,?,?,?,?,?,?,?)`, []any{"demo-insight-003", "demo-interview-security", `["权限与数据保留风险"]`, `["安全评审后再扩大试点"]`, `["访问控制","审计记录"]`, `[]`, 0.65, now}},
		{`INSERT OR IGNORE INTO insights (id,interview_id,pain_points_json,behaviors_json,needs_json,evidence_chunk_ids_json,confidence,created_at) VALUES (?,?,?,?,?,?,?,?)`, []any{"demo-insight-004", "demo-interview-product", `["低置信度易被误读"]`, `["用一页摘要评审"]`, `["置信度提醒","反证显示"]`, `[]`, 0.52, now}},
		{`INSERT OR IGNORE INTO product_requirements (id,document_id,product_area,version,status,summary,created_at) VALUES (?,?,?,?,?,?,?)`, []any{"demo-prd-001", documentIDs["prd"], "evidence_workspace", "demo-0.1", "draft", "模拟 PRD：让研究结论回到可点击的证据与冲突资料。", now}},
		{`INSERT OR IGNORE INTO structured_datasets (id,document_id,name,schema_json,row_count,freshness,created_at) VALUES (?,?,?,?,?,?,?)`, []any{"demo-dataset-001", documentIDs["dataset"], "[模拟] AI Agent 试点指标", `{"fields":["week","workspace","questions_submitted","source_read_rate","evidence_coverage","human_acceptance_rate"]}`, 4, "2026-W04（模拟）", now}},
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement.query, statement.args...); err != nil {
			return rollback(fmt.Errorf("写入模拟结构化数据失败：%w", err))
		}
	}
	tickets := []struct{ id, externalID, message, category, sentiment, resolution string }{
		{"demo-ticket-001", "T-001", "报告无法定位到原网页", "citation", "negative", "增加证据卡链接"},
		{"demo-ticket-002", "T-002", "团队成员看不到共享资料", "access_control", "negative", "检查角色策略"},
		{"demo-ticket-003", "T-003", "长问题触发过多检索", "cost_control", "neutral", "显示预算上限"},
		{"demo-ticket-004", "T-004", "旧资料仍出现在摘要中", "freshness", "negative", "增加资料日期筛选"},
		{"demo-ticket-005", "T-005", "不同来源给出相反结论", "conflict", "neutral", "显示冲突证据"},
		{"demo-ticket-006", "T-006", "无法导出访问记录", "audit", "negative", "增加审计日志导出"},
		{"demo-ticket-007", "T-007", "报告里的标题显示异常", "rendering", "negative", "规范 Markdown 渲染"},
		{"demo-ticket-008", "T-008", "CSV 字段没有自动识别", "ingestion", "neutral", "提供字段映射"},
	}
	for _, ticket := range tickets {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO support_tickets (id,document_id,external_ticket_id,user_message,category,sub_category,sentiment,product_area,resolution,occurred_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, ticket.id, documentIDs["tickets"], ticket.externalID, ticket.message, ticket.category, "agent_workspace", ticket.sentiment, "research_agent", ticket.resolution, "2026-10-03", now); err != nil {
			return rollback(fmt.Errorf("写入模拟客服工单失败：%w", err))
		}
	}
	if err := tx.Commit(); err != nil {
		return DemoSeedResult{}, fmt.Errorf("提交模拟数据失败：%w", err)
	}
	return DemoSeedResult{Created: created, Status: s.Status()}, nil
}
