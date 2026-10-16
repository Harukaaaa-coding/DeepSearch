package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type PRDCoEditRequest struct {
	Copilot DeliveryCopilot `json:"copilot"`
	Actor   string          `json:"actor"`
	Message string          `json:"message"`
	Mode    string          `json:"mode,omitempty"`
}

type PRDCoEditResponse struct {
	Reply    string            `json:"reply"`
	Warning  string            `json:"warning,omitempty"`
	Proposal PRDCoEditProposal `json:"proposal"`
}

type PRDCoEditProposal struct {
	Summary   string            `json:"summary"`
	Changes   []PRDCoEditChange `json:"changes"`
	Questions []string          `json:"questions"`
}

type PRDCoEditChange struct {
	ID             string                `json:"id"`
	Target         string                `json:"target"`
	Operation      string                `json:"operation"`
	Before         string                `json:"before,omitempty"`
	After          string                `json:"after,omitempty"`
	BeforeItems    []string              `json:"beforeItems,omitempty"`
	AfterItems     []string              `json:"afterItems,omitempty"`
	Rationale      string                `json:"rationale"`
	EvidenceStatus string                `json:"evidenceStatus"`
	Requirement    *ExecutionRequirement `json:"requirement,omitempty"`
	Feedback       *FeedbackItem         `json:"feedback,omitempty"`
}

const prdCoEditPrompt = `你是产品负责人，正在与客户、业务、产品、设计、研发或客服共同编辑一份 PRD。你的职责是提出可审阅的改动建议，不得自动实施，也不得把人工观点伪装为已验证研究事实。

只输出合法 JSON，不要 Markdown 或说明。输出：
{
  "reply":"",
  "proposal":{
    "summary":"",
    "questions":[""],
    "changes":[
      {
        "id":"C-01",
        "target":"background.userProblem | background.currentSituation | background.baselineMetrics | goals | nonGoals | decision.decision | decision.mvp | decision.outOfScope | openQuestions | discovery.userFeedback | requirements.new",
        "operation":"replace | append",
        "before":"",
        "after":"",
        "beforeItems":["替换列表时的完整原列表"],
        "afterItems":["替换列表时的完整新列表"],
        "rationale":"",
        "evidenceStatus":"human_input | assumption | evidence_gap",
        "feedback":{"audience":"","feedback":""},
        "requirement":{"id":"","title":"","priority":"P0|P1","userStory":"","preconditions":[""],"mainFlow":[""],"businessRules":[""],"exceptions":[""],"permissions":[""],"acceptance":["Given ... When ... Then ..."],"tracking":["module_object_action | trigger | properties | client/server"],"evidence":[{"label":"人工输入待验证","type":"human_input"}]}
      }
    ]
  }
}

规则：
- 最多提出 4 条 changes；只使用目标列表中的 target。对 list 类 target 默认使用 append（文字写 after，省略 beforeItems/afterItems）；若用户要细化/修改现有 goals、nonGoals、decision.mvp、decision.outOfScope，可以使用 replace，此时必须逐字复制输入基线为 beforeItems，完整新列表写 afterItems，省略 before/after。每个列表最多 12 项、每项最多 1600 字。不可把要求修改的原文与新文重复追加。对 background 与 decision.decision 使用 replace；其他列表不支持 replace。
- 客户观点可写入 discovery.userFeedback，但它必须带 evidenceStatus=human_input，且不能被描述成“市场事实”。
- 若输入缺少用户、场景、阻碍或代价，优先提出澄清问题，不要虚构数值、市场规模、工期、RICE 分数或访谈结论。
- 新需求必须有主流程、规则、异常、权限、Given/When/Then 验收与埋点。异常优先检查空状态、首次使用、权限、网络、重复、并发、极端数据和恢复。
- 把“研究事实”“人工观点”“待验证假设”明确区分。所有改动都需要人工点击应用后才会写进草案。` + productScopeDetailPrompt

func (e *ResearchEngine) ProposePRDCoEdit(ctx context.Context, request PRDCoEditRequest) (PRDCoEditResponse, error) {
	if request.Mode != "" && request.Mode != "scope_detail" {
		return PRDCoEditResponse{}, fmt.Errorf("不支持的 PRD 共创模式。")
	}
	request.Actor = normalizeCoEditActor(request.Actor)
	request.Message = shorten(strings.TrimSpace(request.Message), 5000)
	if len([]rune(request.Message)) < 4 {
		return PRDCoEditResponse{}, fmt.Errorf("请至少输入 4 个字符，说明客户观点、约束、反对意见或修改目标。")
	}
	if strings.TrimSpace(request.Copilot.PRD.Info.Product) == "" {
		return PRDCoEditResponse{}, fmt.Errorf("请先生成产品交付工作流，再开始 PRD 共创。")
	}

	provider, err := reportProviderFromEnv(os.Getenv)
	if err != nil {
		return coEditUnavailable(request, "当前未配置报告模型，无法生成细化建议。"), nil
	}
	instructions, tokens, attempts := prdCoEditPrompt, 3600, 1
	if request.Mode == "scope_detail" {
		for _, target := range []string{"goals", "nonGoals", "decision.mvp", "decision.outOfScope"} {
			if !validCoEditList(coEditBaselineList(request.Copilot, target)) {
				return coEditUnavailable(request, "范围或目标列表为空或过长；请先在共创对话中单独补充或分段编辑。"), nil
			}
		}
		instructions, tokens, attempts = instructions+scopeDetailCoEditPrompt, 6500, 2
	}
	stageCtx, cancel := context.WithTimeout(ctx, 150*time.Second)
	defer cancel()
	var response PRDCoEditResponse
	validated := false
	failure := "没有生成可安全应用的改动。"
	for attempt := 0; attempt < attempts; attempt++ {
		prompt := instructions
		if attempt > 0 {
			prompt += "\n上次输出未通过校验：" + failure + " 请重新输出完整 JSON，不能只返回补丁。"
		}
		text, err := e.requestModelText(stageCtx, provider, prompt, prdCoEditInput(request), tokens)
		if err != nil {
			return coEditUnavailable(request, "模型请求失败，未生成可应用的细化建议。"), nil
		}
		response = PRDCoEditResponse{}
		if err := decodeProductBlueprintJSON(text, &response); err != nil {
			failure = "模型返回格式不完整。"
			continue
		}
		response.Proposal = validateCoEditListReplacements(sanitizePRDCoEditProposal(response.Proposal, request.Actor), request.Copilot)
		if request.Mode == "scope_detail" {
			changes := make([]PRDCoEditChange, 0, 4)
			for _, change := range response.Proposal.Changes {
				if coEditListTarget(change.Target) && change.Operation == "replace" && len(change.BeforeItems) == len(change.AfterItems) {
					changes = append(changes, change)
				}
			}
			response.Proposal.Changes = changes
			if len(changes) != 4 {
				failure = "四个列表必须各有一条 replace；beforeItems 与基线逐字一致、顺序一致，afterItems 逐项展开且数量与原列表一致。"
				continue
			}
		}
		if len(response.Proposal.Changes) == 0 {
			failure = "没有安全改动：请核对 target、operation、完整 beforeItems 与原列表是否一致。"
			continue
		}
		validated = true
		break
	}
	if !validated {
		return coEditUnavailable(request, failure), nil
	}
	response.Reply = shorten(strings.TrimSpace(response.Reply), 700)
	if response.Reply == "" {
		response.Reply = "我已把这条输入转为待人工确认的 PRD 修改建议。"
	}
	return response, nil
}

func coEditUnavailable(request PRDCoEditRequest, warning string) PRDCoEditResponse {
	if request.Mode == "scope_detail" {
		return PRDCoEditResponse{Reply: "未修改原稿，也未记录为客户反馈。请稍后重试或在共创区指定一项单独细化。", Warning: warning}
	}
	return localCoEditFallback(request, warning+" 已将人工输入保留为待确认观点。")
}

func prdCoEditInput(request PRDCoEditRequest) string {
	prd := request.Copilot.PRD
	var builder strings.Builder
	fmt.Fprintf(&builder, "发言身份：%s\n人工输入：%s\n\n", request.Actor, request.Message)
	fmt.Fprintf(&builder, "产品：%s\n功能：%s\n", shorten(prd.Info.Product, 120), shorten(prd.Info.Feature, 180))
	fmt.Fprintf(&builder, "业务背景：%s\n用户问题：%s\n当前现状：%s\n基线：%s\n", shorten(prd.Background.BusinessContext, 380), shorten(prd.Background.UserProblem, 380), shorten(prd.Background.CurrentSituation, 520), shorten(prd.Background.BaselineMetrics, 380))
	fmt.Fprintf(&builder, "决策：%s\n", shorten(request.Copilot.Decision.Decision, 500))
	builder.WriteString("完整列表基线（用于 beforeItems 精确核对，设计草案不是研究事实）：\n")
	for _, target := range []string{"goals", "nonGoals", "decision.mvp", "decision.outOfScope"} {
		items := coEditBaselineList(request.Copilot, target)
		if !validCoEditList(items) {
			fmt.Fprintf(&builder, "%s：为空或超出安全大小，不允许整体 replace；请建议单独补充/分段编辑。\n", target)
			continue
		}
		encoded, _ := json.Marshal(items)
		fmt.Fprintf(&builder, "%s：%s\n", target, encoded)
	}
	builder.WriteString("本稿成功指标（不得擅自更改目标数值）：\n")
	for i, metric := range prd.SuccessMetrics {
		if i >= 6 {
			break
		}
		fmt.Fprintf(&builder, "%s：口径=%s；基线=%s；目标=%s；周期=%s；Owner=%s\n", shorten(metric.Name, 100), shorten(metric.Definition, 260), shorten(metric.Baseline, 220), shorten(metric.Target, 220), shorten(metric.Window, 100), shorten(metric.Owner, 100))
	}
	builder.WriteString("本稿用户场景（未验证角色依旧为假设）：\n")
	for i, user := range prd.Users {
		if i >= 4 {
			break
		}
		fmt.Fprintf(&builder, "%s：%s；触发=%s；结果=%s\n", shorten(user.Role, 100), shorten(user.Scenario, 240), shorten(user.Trigger, 200), shorten(user.Goal, 200))
	}
	builder.WriteString("现有需求：\n")
	for index, requirement := range prd.Requirements {
		if index == 5 {
			break
		}
		fmt.Fprintf(&builder, "%s %s：%s；主流程=%s；规则=%s；验收=%s\n", requirement.ID, shorten(requirement.Title, 100), shorten(requirement.UserStory, 280), shorten(strings.Join(requirement.MainFlow, "；"), 350), shorten(strings.Join(requirement.BusinessRules, "；"), 350), shorten(strings.Join(requirement.Acceptance, "；"), 500))
	}
	fmt.Fprintf(&builder, "开放问题：%s\n", shorten(strings.Join(prd.OpenQuestions, "；"), 600))
	return builder.String()
}

func sanitizePRDCoEditProposal(proposal PRDCoEditProposal, actor string) PRDCoEditProposal {
	allowedTargets := map[string]bool{
		"background.userProblem": true, "background.currentSituation": true, "background.baselineMetrics": true,
		"goals": true, "nonGoals": true, "decision.decision": true, "decision.mvp": true,
		"decision.outOfScope": true, "openQuestions": true, "discovery.userFeedback": true, "requirements.new": true,
	}
	changes := make([]PRDCoEditChange, 0, min(len(proposal.Changes), 4))
	for index, change := range proposal.Changes {
		if len(changes) == 4 {
			break
		}
		change.Target = strings.TrimSpace(change.Target)
		change.Operation = strings.ToLower(strings.TrimSpace(change.Operation))
		change.After = shorten(strings.TrimSpace(change.After), 1000)
		change.Before = shorten(strings.TrimSpace(change.Before), 600)
		change.Rationale = shorten(strings.TrimSpace(change.Rationale), 500)
		change.EvidenceStatus = strings.ToLower(strings.TrimSpace(change.EvidenceStatus))
		if !allowedTargets[change.Target] || (change.Operation != "append" && change.Operation != "replace") {
			continue
		}
		if (change.Target == "background.userProblem" || change.Target == "background.currentSituation" || change.Target == "background.baselineMetrics" || change.Target == "decision.decision") && change.Operation != "replace" {
			continue
		}
		listReplace := coEditListTarget(change.Target) && change.Operation == "replace"
		if listReplace {
			if !validCoEditList(change.BeforeItems) || !validCoEditList(change.AfterItems) {
				continue
			}
			change.Before, change.After = "", ""
		} else {
			change.BeforeItems, change.AfterItems = nil, nil
			if change.Operation == "replace" && change.Target != "background.userProblem" && change.Target != "background.currentSituation" && change.Target != "background.baselineMetrics" && change.Target != "decision.decision" {
				continue
			}
		}
		if !listReplace && change.Target != "requirements.new" && change.Target != "discovery.userFeedback" && change.After == "" {
			continue
		}
		if change.EvidenceStatus != "human_input" && change.EvidenceStatus != "assumption" && change.EvidenceStatus != "evidence_gap" {
			change.EvidenceStatus = "human_input"
		}
		change.ID = fmt.Sprintf("C-%02d", index+1)
		if change.Target == "discovery.userFeedback" {
			if change.Feedback == nil {
				change.Feedback = &FeedbackItem{Audience: actor, Feedback: change.After}
			}
			change.Feedback.Audience = shorten(strings.TrimSpace(change.Feedback.Audience), 80)
			if change.Feedback.Audience == "" {
				change.Feedback.Audience = actor
			}
			change.Feedback.Feedback = shorten(strings.TrimSpace(change.Feedback.Feedback), 1000)
			if change.Feedback.Feedback == "" {
				continue
			}
			change.Feedback.Evidence = []EvidenceRef{{Label: "人工输入 · " + actor, Type: "human_input"}}
		}
		if change.Target == "requirements.new" {
			if change.Requirement == nil || strings.TrimSpace(change.Requirement.Title) == "" || strings.TrimSpace(change.Requirement.UserStory) == "" {
				continue
			}
			change.Requirement.ID = ""
			change.Requirement.Title = shorten(strings.TrimSpace(change.Requirement.Title), 140)
			change.Requirement.UserStory = shorten(strings.TrimSpace(change.Requirement.UserStory), 600)
			if change.Requirement.Priority != "P0" && change.Requirement.Priority != "P1" {
				change.Requirement.Priority = "P1"
			}
			change.Requirement.Evidence = []EvidenceRef{{Label: "人工输入待验证 · " + actor, Type: "human_input"}}
		}
		changes = append(changes, change)
	}
	proposal.Summary = shorten(strings.TrimSpace(proposal.Summary), 500)
	proposal.Questions = uniqueStrings(trimCoEditStrings(proposal.Questions, 5, 320))
	proposal.Changes = changes
	return proposal
}

func localCoEditFallback(request PRDCoEditRequest, warning string) PRDCoEditResponse {
	message := shorten(request.Message, 1000)
	return PRDCoEditResponse{
		Reply:   "已记录这条人工观点。它尚未被当作研究事实；请确认后再写入 PRD。",
		Warning: warning,
		Proposal: PRDCoEditProposal{
			Summary: "保守记录人工输入，等待产品负责人确认如何改变 PRD。",
			Changes: []PRDCoEditChange{{
				ID: "C-01", Target: "discovery.userFeedback", Operation: "append", After: message,
				Rationale: "先保留客户/协作者输入，再决定是否转为需求、约束或待验证假设。", EvidenceStatus: "human_input",
				Feedback: &FeedbackItem{Audience: request.Actor, Feedback: message, Evidence: []EvidenceRef{{Label: "人工输入 · " + request.Actor, Type: "human_input"}}},
			}},
			Questions: []string{"这条观点来自哪类客户、发生在什么场景、造成了什么代价？"},
		},
	}
}

func normalizeCoEditActor(actor string) string {
	switch strings.TrimSpace(actor) {
	case "客户", "业务负责人", "产品负责人", "设计", "研发", "客服/运营", "安全/合规":
		return strings.TrimSpace(actor)
	default:
		return "协作者"
	}
}

func trimCoEditStrings(items []string, limit, maxLen int) []string {
	output := make([]string, 0, min(len(items), limit))
	for _, item := range items {
		item = shorten(strings.TrimSpace(item), maxLen)
		if item != "" {
			output = append(output, item)
		}
		if len(output) == limit {
			break
		}
	}
	return output
}
