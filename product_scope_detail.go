package main

import "strings"

// Shared by first-pass generation and human-reviewed expansion. These fields
// are explanations, not the short labels used for navigation and diagrams.
const productScopeDetailPrompt = `

范围与目标详细度（覆盖普通数组短句的长度限制）：goals、nonGoals、mvp、outOfScope 每项可以写 100–260 个中文字，按信息需要写，不为凑字数扩写，也不能只给一个功能名。保留 JSON 字符串数组，不输出 Markdown 标题。四块各司其职，不复制相同口号：
- mvp 回答“第一期具体交付到哪里”：每项写业务名称、具体使用者与触发、必需输入字段、主要处理步骤、可交付输出、边界/失败时的人工替代，以及一个可检查的完成条件。关联当前真实存在的 REQ ID；没有关联需求时标为待补规格，不能虚构 ID。数量服从本期范围，不擅自增加功能。解释 HITL 为人工确认、母版为可衍生的平台无关原稿等术语；只在当前产品确有该概念时使用。
- goals 回答“为什么做、如何知道有用”：写目标用户与业务结果、衡量对象/统计口径、何时由谁验证、哪些情况不算成功。优先引用本稿 successMetrics 中与这个目标直接相关的指标；没有适配指标时明确“本稿缺此指标，建议采集……”并提出统计对象/事件，不能拿无关指标充数。没有业务基线时写具体采集动作，目标值只能标为待试点验证的目标假设。不能把“完成开发、交付 PRD”当成终端用户的业务目标。
- outOfScope 回答“第一期不交付的具体能力”：说明排除什么动作/输入/输出、为什么暂不做、当前如何处理或明确禁止、未来重新评估需要什么前置条件。永久不支持的行为不可包装成后续必做。不要借细化为用户增加新承诺。
- nonGoals 回答“产品不承担哪些业务结果或责任”：说明不承诺的结果、控制不了的变量、仍承担的可检查责任和失败时如何向用户解释。不要与 outOfScope 重复列同一份功能清单；如果同一主题必须出现，两处分别说实现边界与结果承诺。
所有新增规则、阈值和优先级属于设计提议，不是已经验证的用户事实。未确认的外部 API、权限、自动发布/写操作等依赖必须明确为待验证；写清未接通时的人工导出或其它已有替代路径，禁止把“生成完成”“已审批”“已导出”记作“外部执行成功”。必须区分“可执行/待人工执行”和“外部执行成功”：只有明确的成功回执或授权用户提交外部结果后才可标记成功，不能批准后直接进入 Published/成功状态；人工记录须注明人工确认而非平台核验。输入最小字段可以提出设计建议，不能只写“自定义列/字段待定”而不说明如何关联任务、时间、来源、单位与缺失值。技术术语首次使用要给出具体业务含义。没有证据的收益、涨粉、效率增幅、市场结论不得补造。`

const scopeDetailCoEditPrompt = `

本次是“细化现有范围与目标”，不是重新立项：仅允许对 goals、nonGoals、decision.mvp、decision.outOfScope 四个列表提出 operation=replace。每个目标最多一条 change，通常四条；不得修改背景、需求、指标值或追加用户反馈。完整保留每个列表的既有意图、排除项、平台及责任边界，逐条展开，不删减、不扩大范围、不替换为另一个产品。前值 beforeItems 必须逐字复制输入的同名基线数组，后值 afterItems 必须为详细字符串数组；before/after 留空。假设标记写在 afterItems 本文内，不能只写 evidenceStatus。需要新数据的地方给出验证动作，不编造数字。新的建议规则与原有规格不一致时保留原规格，在 questions 中列出冲突交由人工决定。`

func coEditListTarget(target string) bool {
	switch target {
	case "goals", "nonGoals", "decision.mvp", "decision.outOfScope":
		return true
	}
	return false
}

func coEditBaselineList(copilot DeliveryCopilot, target string) []string {
	switch target {
	case "goals":
		return copilot.PRD.Goals
	case "nonGoals":
		return copilot.PRD.NonGoals
	case "decision.mvp":
		return copilot.Decision.MVP
	case "decision.outOfScope":
		return copilot.Decision.OutOfScope
	}
	return nil
}

// Never shorten a before-value: it is an optimistic concurrency token as well
// as the text being reviewed. Oversized baselines must be edited separately.
func validCoEditList(items []string) bool {
	if len(items) == 0 || len(items) > 12 {
		return false
	}
	for _, item := range items {
		if strings.TrimSpace(item) == "" || len([]rune(item)) > 1600 {
			return false
		}
	}
	return true
}

func equalCoEditLists(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func validateCoEditListReplacements(proposal PRDCoEditProposal, copilot DeliveryCopilot) PRDCoEditProposal {
	changes := make([]PRDCoEditChange, 0, len(proposal.Changes))
	seen := make(map[string]bool)
	for _, change := range proposal.Changes {
		if coEditListTarget(change.Target) && change.Operation == "replace" {
			if seen[change.Target] || !validCoEditList(change.BeforeItems) || !validCoEditList(change.AfterItems) || !equalCoEditLists(change.BeforeItems, coEditBaselineList(copilot, change.Target)) {
				continue
			}
			seen[change.Target] = true
		}
		changes = append(changes, change)
	}
	proposal.Changes = changes
	return proposal
}
