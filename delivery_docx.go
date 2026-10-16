package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
)

// buildDeliveryPRDDOCX produces a self-contained, reviewable Word version of the
// generated delivery PRD. The browser only requests it after the user clicks export.
func buildDeliveryPRDDOCX(copilot DeliveryCopilot) ([]byte, error) {
	if strings.TrimSpace(copilot.PRD.Info.PRDID) == "" || len(copilot.PRD.Requirements) == 0 {
		return nil, fmt.Errorf("PRD 草案不完整，无法导出")
	}
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	files := map[string]string{
		"[Content_Types].xml":          docxContentTypes(),
		"_rels/.rels":                  docxRootRelationships(),
		"docProps/core.xml":            deliveryDocxCoreProperties(copilot),
		"docProps/app.xml":             docxAppProperties(),
		"word/styles.xml":              docxStyles(),
		"word/document.xml":            deliveryPRDDocxDocument(copilot),
		"word/_rels/document.xml.rels": docxDocumentRelationships(),
	}
	for name, content := range files {
		entry, err := archive.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func deliveryDocxCoreProperties(copilot DeliveryCopilot) string {
	title := docxEscape("产品需求文档 " + cleanDOCXText(copilot.PRD.Info.Feature))
	return `<?xml version="1.0" encoding="UTF-8"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:title>` + title + `</dc:title><dc:creator>DeepSearch</dc:creator><cp:lastModifiedBy>DeepSearch</cp:lastModifiedBy></cp:coreProperties>`
}

func deliveryPRDDocxDocument(copilot DeliveryCopilot) string {
	prd := copilot.PRD
	visuals := copilot.Visuals
	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	body.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	writeLine := func(text, style string) { body.WriteString(docxParagraph(docxLine{Text: text, Style: style})) }
	writeBullets := func(items []string) {
		for _, item := range items {
			if strings.TrimSpace(item) != "" {
				writeLine("• "+item, "Bullet")
			}
		}
	}
	writeTable := func(headers []string, rows [][]string) { body.WriteString(deliveryDocxTable(headers, rows)) }

	writeLine("产品需求文档", "Title")
	writeLine(cleanDOCXText(prd.Info.Product+" - "+prd.Info.Feature), "Subtitle")
	writeLine("版本 "+prd.Info.Version+" · 状态 "+prd.Info.Status+" · 最后更新 "+prd.Info.LastUpdated, "Metadata")
	writeLine("本文用于产品、设计、研发、测试和安全评审。结论与需求需回到证据链；示例资料不替代真实业务基线。", "Body")

	writeLine("1 文档控制", "Heading1")
	writeTable([]string{"项目", "内容"}, [][]string{{"PRD ID", prd.Info.PRDID}, {"产品", prd.Info.Product}, {"功能", prd.Info.Feature}, {"Owner", prd.Info.Owner}, {"评审角色", prd.Info.Reviewers}, {"目标发布", prd.Info.TargetRelease}, {"版本状态", prd.Info.Status}})

	writeLine("2 背景与问题定义", "Heading1")
	writeLine("业务背景", "Heading2")
	writeLine(prd.Background.BusinessContext, "Body")
	writeLine("用户问题", "Heading2")
	writeLine(prd.Background.UserProblem, "Body")
	writeLine("现状与基线", "Heading2")
	writeLine(prd.Background.CurrentSituation+" "+prd.Background.BaselineMetrics, "Body")
	writeLine("目标", "Heading2")
	writeBullets(prd.Goals)
	writeLine("非目标", "Heading2")
	writeBullets(prd.NonGoals)
	writeLine("成功指标与衡量口径", "Heading2")
	metricRows := make([][]string, 0, len(prd.SuccessMetrics))
	for _, metric := range prd.SuccessMetrics {
		metricRows = append(metricRows, []string{metric.Name, metric.Definition, metric.Baseline, metric.Target, metric.Window, metric.Owner})
	}
	writeTable([]string{"指标", "口径", "基线", "目标", "周期", "Owner"}, metricRows)

	writeLine("3 用户与场景", "Heading1")
	userRows := make([][]string, 0, len(prd.Users))
	for _, user := range prd.Users {
		userRows = append(userRows, []string{user.Role, user.Scenario, user.Trigger, user.Goal})
	}
	writeTable([]string{"角色", "场景", "触发", "目标"}, userRows)
	body.WriteString(deliveryScenarioDocx(copilot.Discovery.Scenarios))

	writeLine("4 决策与范围", "Heading1")
	writeLine("推荐决策", "Heading2")
	writeLine(copilot.Decision.Decision+" "+copilot.Decision.Reasoning, "Body")
	optionRows := make([][]string, 0, len(copilot.Decision.Options))
	for _, option := range copilot.Decision.Options {
		optionRows = append(optionRows, []string{option.Name, option.Benefit, option.Tradeoff, option.Verdict})
	}
	writeTable([]string{"方案", "收益", "取舍", "结论"}, optionRows)
	writeLine("MVP 范围", "Heading2")
	writeBullets(copilot.Decision.MVP)
	writeLine("非范围", "Heading2")
	writeBullets(copilot.Decision.OutOfScope)

	writeLine("5 关键流程与交付工作流", "Heading1")
	writeLine("产品用户流程", "Heading2")
	writeLine(strings.Join(prd.UserFlow, " → "), "Body")
	writeLine("系统处理流程", "Heading2")
	writeLine(strings.Join(prd.SystemFlow, " → "), "Body")
	writeLine("跨职能交付生命周期", "Heading2")
	writeTable([]string{"阶段", "负责人", "产出"}, visualFlowRows(visuals.Flow))
	writeLine("质量门 SOP", "Heading2")
	sopRows := make([][]string, 0, len(visuals.SOP))
	for _, step := range visuals.SOP {
		sopRows = append(sopRows, []string{fmt.Sprintf("%d %s", step.Order, step.Title), step.Owner, step.Input, step.Output, step.QualityGate})
	}
	writeTable([]string{"步骤", "Owner", "输入", "输出", "质量门"}, sopRows)

	writeLine("6 功能需求", "Heading1")
	for _, requirement := range prd.Requirements {
		writeLine(requirement.ID+" "+requirement.Title+" "+requirement.Priority, "Heading2")
		writeLine("用户故事："+requirement.UserStory, "Body")
		writeLine("前置条件："+strings.Join(requirement.Preconditions, "；"), "Body")
		writeLine("主流程："+strings.Join(requirement.MainFlow, " → "), "Body")
		writeLine("业务规则："+strings.Join(requirement.BusinessRules, "；"), "Body")
		writeLine("异常与降级："+strings.Join(requirement.Exceptions, "；"), "Body")
		writeLine("权限："+strings.Join(requirement.Permissions, "；"), "Body")
		writeLine("验收："+strings.Join(requirement.Acceptance, "；"), "Body")
		writeLine("埋点："+strings.Join(requirement.Tracking, "，"), "Body")
	}

	writeLine("7 低保真原型规格", "Heading1")
	for _, screen := range prd.Prototype.Screens {
		writeLine(screen.ID+" "+screen.Name+" "+screen.Route, "Heading2")
		writeLine("目的："+screen.Purpose, "Body")
		writeLine("主操作："+screen.PrimaryAction, "Body")
		regionRows := make([][]string, 0, len(screen.Regions))
		for _, region := range screen.Regions {
			regionRows = append(regionRows, []string{region.Name, region.Content, region.Behavior})
		}
		writeTable([]string{"区域", "内容", "交互或规则"}, regionRows)
		writeLine("状态："+strings.Join(screen.States, "；"), "Body")
	}

	writeLine("8 方案设计与权限", "Heading1")
	writeLine("方案模块", "Heading2")
	architectureRows := make([][]string, 0, len(prd.Architecture))
	for _, module := range prd.Architecture {
		architectureRows = append(architectureRows, []string{module.Name, strings.Join(module.Children, "；")})
	}
	writeTable([]string{"模块", "职责"}, architectureRows)
	writeLine("方案数据流", "Heading2")
	writeTable([]string{"节点", "负责人", "输出"}, visualFlowRows(visuals.SolutionFlow))
	writeLine("权限矩阵", "Heading2")
	permissionRows := make([][]string, 0, len(prd.Permissions))
	for _, permission := range prd.Permissions {
		permissionRows = append(permissionRows, []string{permission.Capability, permission.User, permission.CSAgent, permission.Admin})
	}
	writeTable([]string{"能力", "User", "CS Agent", "Admin"}, permissionRows)

	writeLine("9 数据模型与数据字典", "Heading1")
	entityRows := make([][]string, 0, len(copilot.Solution.DataModel))
	for _, entity := range copilot.Solution.DataModel {
		entityRows = append(entityRows, []string{entity.Name, entity.PrimaryKey, entity.Purpose, strings.Join(entity.Relationships, "；"), entity.Access, entity.Retention})
	}
	writeTable([]string{"实体", "主键", "用途", "关系", "访问", "保留"}, entityRows)
	dictionaryRows := make([][]string, 0, len(copilot.Solution.DataDictionary))
	for _, item := range copilot.Solution.DataDictionary {
		dictionaryRows = append(dictionaryRows, []string{item.Entity + "." + item.Field, item.Type + "；必填：" + item.Required, item.Description, "示例：" + item.Example + "；分级：" + item.Classification})
	}
	writeTable([]string{"字段", "类型与必填", "说明", "示例与分级"}, dictionaryRows)

	writeLine("10 接口契约与依赖", "Heading1")
	for _, api := range copilot.Solution.APIContracts {
		writeLine(api.Method+" "+api.Path+" "+api.Name, "Heading2")
		writeLine("授权："+api.Auth, "Body")
		writeLine("请求："+strings.Join(api.Request, "；"), "Body")
		writeLine("响应："+strings.Join(api.Response, "；"), "Body")
		writeLine("失败与降级："+strings.Join(api.Failures, "；"), "Body")
	}
	dependencyRows := make([][]string, 0, len(copilot.Solution.Dependencies))
	for _, dependency := range copilot.Solution.Dependencies {
		dependencyRows = append(dependencyRows, []string{dependency.Name, dependency.Owner, dependency.NeedBy, dependency.Risk, dependency.Fallback})
	}
	writeTable([]string{"依赖", "Owner", "需要时间", "风险", "替代方案"}, dependencyRows)

	writeLine("11 工期与交付计划", "Heading1")
	writeLine(prd.DeliveryPlan.Estimate, "Body")
	writeLine("工期前提", "Heading2")
	writeBullets(prd.DeliveryPlan.Assumptions)
	scheduleRows := make([][]string, 0, len(prd.DeliveryPlan.Phases))
	for _, phase := range prd.DeliveryPlan.Phases {
		scheduleRows = append(scheduleRows, []string{phase.Window, phase.Stage, phase.Owner, phase.Outcome, phase.EntryGate, phase.ExitGate, phase.DelayRisk})
	}
	writeTable([]string{"时间", "阶段", "Owner", "产出", "进入条件", "退出条件", "延期风险"}, scheduleRows)

	writeLine("12 非功能、测试与发布", "Heading1")
	nfrRows := make([][]string, 0, len(prd.NonFunctional))
	for _, nfr := range prd.NonFunctional {
		nfrRows = append(nfrRows, []string{nfr.Area, nfr.Requirement, nfr.Verification, nfr.Status})
	}
	writeTable([]string{"领域", "要求", "验证", "状态"}, nfrRows)
	writeLine("测试策略", "Heading2")
	writeLine(visuals.TestPlan.Strategy, "Body")
	testRows := make([][]string, 0, len(visuals.TestPlan.Cases))
	for _, testCase := range visuals.TestPlan.Cases {
		testRows = append(testRows, []string{testCase.ID + " " + testCase.Title, testCase.Level, strings.Join(testCase.Steps, " → "), testCase.Expected})
	}
	writeTable([]string{"用例", "级别", "步骤", "预期"}, testRows)
	writeLine("发布退出标准", "Heading2")
	writeBullets(visuals.TestPlan.ExitRules)
	writeLine("回滚", "Heading2")
	writeBullets(copilot.Evaluation.Rollback)

	writeLine("13 证据链、风险与待决项", "Heading1")
	chainRows := make([][]string, 0, len(copilot.Chains))
	for _, chain := range copilot.Chains {
		chainRows = append(chainRows, []string{chain.Source, chain.Insight, chain.Requirement, evidenceRefText(chain.Evidence)})
	}
	writeTable([]string{"来源问题", "洞察", "需求", "证据引用"}, chainRows)
	writeLine("Readiness", "Heading2")
	writeLine(fmt.Sprintf("%d/100 · %s", copilot.Readiness.Score, copilot.Readiness.Verdict), "Body")
	readinessRows := make([][]string, 0, len(copilot.Readiness.Checks))
	for _, check := range copilot.Readiness.Checks {
		readinessRows = append(readinessRows, []string{check.Area, check.Status, check.Note})
	}
	writeTable([]string{"检查项", "状态", "说明"}, readinessRows)
	writeLine("待决项", "Heading2")
	writeBullets(prd.OpenQuestions)
	writeLine("进入下一阶段前动作", "Heading2")
	writeBullets(copilot.Readiness.NextActions)

	body.WriteString(`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1080" w:right="1080" w:bottom="1080" w:left="1080" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr></w:body></w:document>`)
	return body.String()
}

// deliveryScenarioDocx keeps research-backed business scenarios in the review
// baseline without presenting proposed workflows as observed facts.
func deliveryScenarioDocx(scenarios []ProductScenario) string {
	var body strings.Builder
	writeLine := func(text, style string) {
		if strings.TrimSpace(text) != "" {
			body.WriteString(docxParagraph(docxLine{Text: text, Style: style}))
		}
	}
	writeField := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			writeLine(label+"："+value, "Body")
		}
	}
	writeItems := func(label string, items []string) {
		started := false
		for _, item := range items {
			if strings.TrimSpace(item) == "" {
				continue
			}
			if !started {
				writeLine(label, "Body")
				started = true
			}
			writeLine("• "+item, "Bullet")
		}
	}
	started := false
	for index, scenario := range scenarios {
		if strings.TrimSpace(scenario.Name+scenario.Actor+scenario.Trigger+scenario.CurrentGap+scenario.Outcome+scenario.Context) == "" {
			continue
		}
		if !started {
			writeLine("详细业务场景", "Heading2")
			writeLine("以下场景用于界定待解决的业务任务。证据只支持其标注的字段；拟议方案、预期结果与验收条件仍需目标用户验证。", "Body")
			started = true
		}
		name := strings.TrimSpace(scenario.ID + " " + scenario.Name)
		if name == "" {
			name = fmt.Sprintf("场景 %d", index+1)
		}
		writeLine(name, "Heading2")
		basis := "历史场景：证据关联与方案有效性待核验"
		switch scenario.Basis {
		case "research_hypothesis":
			basis = "研究支持的场景假设：尚未完成真实用户验证"
		case "unverified_hypothesis":
			basis = "未验证场景假设：不可作为既定需求"
		case "needs_research":
			basis = "场景证据不足：需继续调研"
		}
		writeLine(basis, "Metadata")
		writeField("业务角色", scenario.Actor)
		writeField("业务环境与约束", scenario.Context)
		writeField("具体触发", scenario.Trigger)
		writeItems("输入与前置资料", scenario.Inputs)
		writeItems("现有流程（事实范围以证据为准）", scenario.CurrentWorkflow)
		writeField("业务缺口（待验证）", scenario.CurrentGap)
		writeField("业务影响（待验证）", scenario.Impact)
		writeItems("拟议工作流（方案）", scenario.ProposedWorkflow)
		writeField("预期结果（拟议）", scenario.Outcome)
		writeItems("验收条件（拟议）", scenario.Acceptance)
		writeItems("异常与边界", scenario.Exceptions)
		writeField("验证动作", scenario.Validation)
		if len(scenario.EvidenceLinks) > 0 {
			writeLine("字段级证据与适用范围", "Body")
			for _, link := range scenario.EvidenceLinks {
				label := strings.TrimSpace(link.ID + " " + link.SourceTitle)
				if label != "" {
					writeLine(label, "Source")
				}
				writeField("支持字段", scenarioEvidenceFieldLabel(link.Field))
				writeField("原文摘录", link.Quote)
				writeField("支持理由与限制", link.Explanation)
				writeField("发布时间", link.PublishedAt)
				writeField("原始网址", link.URL)
			}
		} else if len(scenario.Evidence) > 0 {
			writeField("历史引用（尚未建立字段级支持关系）", evidenceRefText(scenario.Evidence))
		}
	}
	return body.String()
}

func scenarioEvidenceFieldLabel(field string) string {
	labels := map[string]string{
		"actor": "业务角色", "context": "业务环境与约束", "trigger": "具体触发",
		"inputs": "输入与前置资料", "currentWorkflow": "现有流程", "currentGap": "业务缺口",
		"impact": "业务影响", "proposedWorkflow": "拟议工作流", "outcome": "预期结果",
		"acceptance": "验收条件", "exceptions": "异常与边界", "validation": "验证动作",
	}
	if label, ok := labels[field]; ok {
		return label
	}
	return field
}

func visualFlowRows(items []VisualFlowNode) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{item.ID + " " + item.Title, item.Owner, item.Output})
	}
	return rows
}

func evidenceRefText(refs []EvidenceRef) string {
	parts := make([]string, 0, len(refs))
	for _, ref := range refs {
		text := strings.TrimSpace(ref.Label)
		if ref.ID != "" {
			text += " (" + ref.ID + ")"
		}
		if text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return "待补充证据"
	}
	return strings.Join(parts, "；")
}

func deliveryDocxTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}
	var table strings.Builder
	width := 9360 / len(headers)
	table.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="9360" w:type="dxa"/><w:tblBorders><w:top w:val="single" w:sz="6" w:color="D9D9D9"/><w:left w:val="single" w:sz="6" w:color="D9D9D9"/><w:bottom w:val="single" w:sz="6" w:color="D9D9D9"/><w:right w:val="single" w:sz="6" w:color="D9D9D9"/><w:insideH w:val="single" w:sz="6" w:color="D9D9D9"/><w:insideV w:val="single" w:sz="6" w:color="D9D9D9"/></w:tblBorders><w:tblCellMar><w:top w:w="90" w:type="dxa"/><w:left w:w="100" w:type="dxa"/><w:bottom w:w="90" w:type="dxa"/><w:right w:w="100" w:type="dxa"/></w:tblCellMar></w:tblPr><w:tblGrid>`)
	for range headers {
		table.WriteString(fmt.Sprintf(`<w:gridCol w:w="%d"/>`, width))
	}
	table.WriteString(`</w:tblGrid><w:tr><w:trPr><w:tblHeader/></w:trPr>`)
	for _, header := range headers {
		table.WriteString(deliveryDocxTableCell(header, true, width))
	}
	table.WriteString(`</w:tr>`)
	for _, row := range rows {
		table.WriteString(`<w:tr>`)
		for index := range headers {
			value := ""
			if index < len(row) {
				value = row[index]
			}
			table.WriteString(deliveryDocxTableCell(value, false, width))
		}
		table.WriteString(`</w:tr>`)
	}
	table.WriteString(`</w:tbl>`)
	return table.String()
}

func deliveryDocxTableCell(value string, header bool, width int) string {
	fill, color, bold := "FFFFFF", "202020", ""
	if header {
		fill, color, bold = "274156", "FFFFFF", `<w:b/>`
	}
	return fmt.Sprintf(`<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/><w:shd w:val="clear" w:fill="%s"/><w:vAlign w:val="center"/></w:tcPr><w:p><w:pPr><w:spacing w:after="60"/></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Aptos" w:hAnsi="Aptos" w:eastAsia="Microsoft YaHei"/><w:color w:val="%s"/><w:sz w:val="18"/>%s</w:rPr><w:t xml:space="preserve">%s</w:t></w:r></w:p></w:tc>`, width, fill, color, bold, docxEscape(cleanDOCXText(value)))
}
