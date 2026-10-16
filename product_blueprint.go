package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ProductBlueprint is the model-generated, reviewable product-design layer for a
// product-discovery task. It deliberately uses the same typed fields as the Enterprise
// PRD so the UI, local persistence and DOCX export all receive one consistent artifact.
type ProductBlueprint struct {
	ProductName     string                     `json:"productName"`
	OneSentence     string                     `json:"oneSentence"`
	Goals           []string                   `json:"goals"`
	NonGoals        []string                   `json:"nonGoals"`
	SuccessMetrics  []SuccessMetric            `json:"successMetrics"`
	Users           []PRDUserScenario          `json:"users"`
	Options         []DecisionOption           `json:"options"`
	Decision        string                     `json:"decision"`
	Reasoning       string                     `json:"reasoning"`
	MVP             []string                   `json:"mvp"`
	OutOfScope      []string                   `json:"outOfScope"`
	UserFlow        []string                   `json:"userFlow"`
	SystemFlow      []string                   `json:"systemFlow"`
	SOP             []SOPStep                  `json:"sop"`
	Permissions     []PermissionRule           `json:"permissions"`
	Requirements    []ExecutionRequirement     `json:"requirements"`
	Prototype       ProductPrototype           `json:"prototype"`
	DataModel       []DataEntity               `json:"dataModel"`
	DataDictionary  []DataDictionaryItem       `json:"dataDictionary"`
	APIContracts    []APIContract              `json:"apiContracts"`
	Dependencies    []DeliveryDependency       `json:"dependencies"`
	NonFunctional   []NonFunctionalRequirement `json:"nonFunctional"`
	TestPlan        TestPlan                   `json:"testPlan"`
	DeliveryPlan    DeliveryPlan               `json:"deliveryPlan"`
	Tracking        []string                   `json:"tracking"`
	TestFocus       []string                   `json:"testFocus"`
	LaunchChecklist []string                   `json:"launchChecklist"`
	Monitoring      []string                   `json:"monitoring"`
	Rollback        []string                   `json:"rollback"`
	OpenQuestions   []string                   `json:"openQuestions"`
}

const productBlueprintPrompt = `你是一位资深产品负责人、架构师与交付负责人。请把“产品发现型研究任务”转成可以进入设计、研发、测试评审的产品蓝图。

事实边界：只能把输入报告和证据包中的内容当作事实依据。对目标用户、市场规模、转化率、工期、指标数值、技术选择等没有证据支持的内容，必须明确标记为“假设/待验证”，不能伪装为用户访谈事实。每条用户、方案、功能需求可以引用的 evidence.id 仅能是输入中出现的 E#；若无法关联，evidence 数组填 [{"label":"待通过访谈或试点验证","type":"gap"}]。

输出边界：只输出一个合法 JSON 对象，不要 Markdown、代码围栏或任何说明。必须完整输出以下字段：
{
  "productName":"", "oneSentence":"", "goals":[""], "nonGoals":[""],
  "successMetrics":[{"name":"","definition":"","baseline":"","target":"","window":"","owner":""}],
  "users":[{"role":"","buyer":"","scenario":"","trigger":"","goal":"","evidence":[{"id":"E1"}]}],
  "options":[{"name":"","benefit":"","tradeoff":"","verdict":""}], "decision":"", "reasoning":"", "mvp":[""], "outOfScope":[""],
  "userFlow":[""], "systemFlow":[""],
  "sop":[{"order":1,"title":"","owner":"","action":"","input":"","output":"","qualityGate":""}],
  "permissions":[{"capability":"","user":"","csAgent":"","admin":""}],
  "requirements":[{"id":"REQ-001","title":"","priority":"P0","userStory":"","preconditions":[""],"mainFlow":[""],"businessRules":[""],"exceptions":[""],"permissions":[""],"acceptance":[""],"tracking":[""],"evidence":[{"id":"E1"}]}],
  "prototype":{"screens":[{"id":"SCR-01","name":"","route":"","purpose":"","primaryAction":"","regions":[{"name":"","content":"","behavior":""}],"states":[""]}]},
  "dataModel":[{"name":"","purpose":"","primaryKey":"","relationships":[""],"retention":"","access":""}],
  "dataDictionary":[{"entity":"","field":"","type":"","required":"","description":"","example":"","classification":""}],
  "apiContracts":[{"name":"","method":"POST","path":"","auth":"","request":[""],"response":[""],"failures":[""]}],
  "dependencies":[{"name":"","owner":"","needBy":"","risk":"","fallback":""}],
  "nonFunctional":[{"area":"","requirement":"","verification":"","status":""}],
  "testPlan":{"strategy":"","roles":[{"role":"","owner":"","howToTest":"","exit":""}],"cases":[{"id":"TC-001","title":"","level":"P0","steps":[""],"expected":""}],"dataSets":[{"name":"","purpose":"","example":"","notice":""}],"exitRules":[""]},
  "deliveryPlan":{"estimate":"","assumptions":[""],"phases":[{"window":"","stage":"","owner":"","outcome":"","entryGate":"","exitGate":"","delayRisk":""}]},
  "tracking":[""], "testFocus":[""], "launchChecklist":[""], "monitoring":[""], "rollback":[""], "openQuestions":[""]
}

质量要求：
- 必须结合任务主题给出具体的产品名、目标用户、工作流、数据对象、原型页面与功能；禁止输出“通用 Agent 工作台”“证据到假设工作台”等套话，除非它确实是题目中的产品。
- 至少 3 类用户、3 个方案、4 条需求（含 P0/P1）、3 个原型页、4 个数据实体、6 个数据字典字段、3 个接口、4 类非功能要求、4 个测试用例、5 个交付阶段。
- 每条功能需求都必须有主流程、规则、异常、权限、验收标准、埋点和证据或 Gap。
- 对自媒体、内容创作、营销、运营等语境，明确区分个人创作者、MCN/运营、品牌方/审核方；不要凭空称其为已验证客户。`

// The blueprint is produced in two bounded responses. This avoids asking a model
// to serialize an entire PRD, data model and delivery plan in one JSON response.
type productBlueprintCore struct {
	ProductName    string                 `json:"productName"`
	OneSentence    string                 `json:"oneSentence"`
	Goals          []string               `json:"goals"`
	NonGoals       []string               `json:"nonGoals"`
	SuccessMetrics []SuccessMetric        `json:"successMetrics"`
	Users          []PRDUserScenario      `json:"users"`
	Options        []DecisionOption       `json:"options"`
	Decision       string                 `json:"decision"`
	Reasoning      string                 `json:"reasoning"`
	MVP            []string               `json:"mvp"`
	OutOfScope     []string               `json:"outOfScope"`
	UserFlow       []string               `json:"userFlow"`
	SystemFlow     []string               `json:"systemFlow"`
	SOP            []SOPStep              `json:"sop"`
	Permissions    []PermissionRule       `json:"permissions"`
	Requirements   []ExecutionRequirement `json:"requirements"`
	Prototype      ProductPrototype       `json:"prototype"`
}

type productBlueprintDelivery struct {
	DataModel       []DataEntity               `json:"dataModel"`
	DataDictionary  []DataDictionaryItem       `json:"dataDictionary"`
	APIContracts    []APIContract              `json:"apiContracts"`
	Dependencies    []DeliveryDependency       `json:"dependencies"`
	NonFunctional   []NonFunctionalRequirement `json:"nonFunctional"`
	TestPlan        TestPlan                   `json:"testPlan"`
	DeliveryPlan    DeliveryPlan               `json:"deliveryPlan"`
	Tracking        []string                   `json:"tracking"`
	TestFocus       []string                   `json:"testFocus"`
	LaunchChecklist []string                   `json:"launchChecklist"`
	Monitoring      []string                   `json:"monitoring"`
	Rollback        []string                   `json:"rollback"`
	OpenQuestions   []string                   `json:"openQuestions"`
}

func (part productBlueprintCore) applyTo(blueprint *ProductBlueprint) {
	blueprint.ProductName, blueprint.OneSentence = part.ProductName, part.OneSentence
	blueprint.Goals, blueprint.NonGoals, blueprint.SuccessMetrics = part.Goals, part.NonGoals, part.SuccessMetrics
	blueprint.Users, blueprint.Options, blueprint.Decision, blueprint.Reasoning = part.Users, part.Options, part.Decision, part.Reasoning
	blueprint.MVP, blueprint.OutOfScope, blueprint.UserFlow, blueprint.SystemFlow = part.MVP, part.OutOfScope, part.UserFlow, part.SystemFlow
	blueprint.SOP, blueprint.Permissions, blueprint.Requirements, blueprint.Prototype = part.SOP, part.Permissions, part.Requirements, part.Prototype
}

func (part productBlueprintDelivery) applyTo(blueprint *ProductBlueprint) {
	blueprint.DataModel, blueprint.DataDictionary, blueprint.APIContracts = part.DataModel, part.DataDictionary, part.APIContracts
	blueprint.Dependencies, blueprint.NonFunctional, blueprint.TestPlan, blueprint.DeliveryPlan = part.Dependencies, part.NonFunctional, part.TestPlan, part.DeliveryPlan
	blueprint.Tracking, blueprint.TestFocus, blueprint.LaunchChecklist = part.Tracking, part.TestFocus, part.LaunchChecklist
	blueprint.Monitoring, blueprint.Rollback, blueprint.OpenQuestions = part.Monitoring, part.Rollback, part.OpenQuestions
}

const productBlueprintCorePrompt = `你是中国互联网产品负责人。你要产出的不是通用 PRD 模板，而是对当前研究任务的“发现 → MVP”决策稿第一部分。

先在心中提取任务中的行业、地域、时间、目标用户、要解决的工作与限制。每个目标、场景、功能、原型页必须带有这些具体名词和动作；不能用“提升效率、统一工作台、全链路、赋能”等可以套用到任何产品的空话。没有证据支持的市场、用户、指标、价格、周期，一律写“假设/待验证”，不可冒充事实。

只输出一个紧凑、完整、合法 JSON；禁止 Markdown、解释、代码围栏。普通字符串最多 45 个中文字符、普通数组短句最多 24 个中文字符，但 goals、nonGoals、mvp、outOfScope 明确例外，遵循末尾的范围与目标详细度要求。需求 userStory、businessRules、acceptance 以及指标 definition、baseline、target 可用 100–180 字说明关键输入、行为、边界、预期和衡量口径，不为达到长度而填充。每条需求主流程 3–5 步、验收 2–3 条、业务规则和异常各最多 3 条；总共先输出 4–6 条核心需求。只输出如下字段，字段不得缺失：
{
"productName":"","oneSentence":"","goals":[""],"nonGoals":[""],
"successMetrics":[{"name":"","definition":"","baseline":"","target":"","window":"","owner":""}],
"users":[{"role":"","buyer":"","scenario":"","trigger":"","goal":"","evidence":[{"id":"E1"}]}],
"options":[{"name":"","benefit":"","tradeoff":"","verdict":""}],"decision":"","reasoning":"","mvp":[""],"outOfScope":[""],
"userFlow":[""],"systemFlow":[""],
"sop":[{"order":1,"title":"","owner":"","action":"","input":"","output":"","qualityGate":""}],
"permissions":[{"capability":"","user":"","csAgent":"","admin":""}],
"requirements":[{"id":"REQ-001","title":"","priority":"P0","userStory":"","preconditions":[""],"mainFlow":[""],"businessRules":[""],"exceptions":[""],"permissions":[""],"acceptance":[""],"tracking":[""],"evidence":[{"id":"E1"}]}],
"prototype":{"screens":[{"id":"SCR-01","name":"","route":"","purpose":"","primaryAction":"","regions":[{"name":"","content":"","behavior":""}],"states":[""]}]}
}

最少输出：2 个目标、2 个非目标、3 个指标、1 类明确的目标用户、2 个可比较方案、4 条需求（P0 和 P1 都要有）、3 个原型页、userFlow 至少 2 步、systemFlow 至少 2 步、4 个 SOP 步骤、3 条权限规则。用户角色依据本次证据和任务确定，不为凑数量虚构 3 类用户；提议中的目标用户必须在 scenario 标明“假设/待验证”。

产品经理工具箱质量规则：
- 方案排序先看 RICE 所需的 Reach、Impact、Confidence、Effort 是否有真实数据；没有就写“待验证”，不得编造分数。Kano 的必备体验/合规风险可优先于魅力功能。
- systemFlow 必须写一个关键对象的状态迁移（状态 — 事件 → 新状态），包含至少一个禁止迁移或拦截规则。
- 所有 P0 需求合计覆盖：空状态、首次使用、权限不足、网络/超时、重复提交/幂等、并发冲突、极端数据、中断恢复；无需适用的项要说明原因。
- 每条需求必须有明确主流程、业务规则、异常、权限、验收、埋点和 E# 或 Gap。验收使用 Given / When / Then；埋点用“模块_对象_动作 | 触发时机 | 属性 | 前端/后端”的紧凑格式，关键成功事件优先后端确认后上报。
- 结论必须明确选择先做什么、暂不做什么、以什么信号停止。
- 必须利用输入的竞品定位、功能、价格与证据缺口，说明所选 MVP 与现有替代方案的区别；没有覆盖的能力只能列为待验证机会，不能断言竞品没有。
- 研究覆盖存在缺口时，目标和方案明确适用范围；不得把“来源多”当作需求已验证。E# 只证明对应原文，设计决策、商业假设及指标目标需单独标注。
- 成功指标写清分子/分母或统计对象、排除条件、采集事件与周期。没有真实业务数据时 baseline 写“未知；需采集……”，不可编造当前人数、转化率、耗时或成本。target 可以提出可测试的目标，但必须写“目标假设/待试点验证”，并说明验证方式；不得把规划目标写成已经达成的结果。公开行业数字不能冒充客户基线，用户角色、预算和付费意愿不能因为资料不足就补造。` + productScopeDetailPrompt

const productBlueprintDeliveryPrompt = `你是中国互联网产品负责人和交付负责人。请为当前研究任务的第一期 MVP 编写“数据 → 测试 → 上线”决策稿第二部分。

这是具体产品设计，不是通用交付模板：数据实体、接口、测试数据和排期必须使用任务中的行业、地域、目标用户与 MVP 功能名。无证据的工期、量级、指标、合规结论必须标记“假设/待验证”。不允许写“待确认”“持续优化”来代替具体动作；每项都要有实际 Owner、进入条件、完成条件或风险触发器。

只输出一个紧凑、完整、合法 JSON；禁止 Markdown、解释、代码围栏。普通字符串最多 45 个中文字符，普通数组短句最多 24 个中文字符。测试 steps、expected、测试数据 example 和非功能 verification 可用 100–180 字说明输入条件、边界、可执行操作和可观测结果。测试用例 4–6 条、每例步骤 3–5 个，覆盖第一部分所有 P0；只展开当前 MVP，避免重复相同说明。只输出如下字段，字段不得缺失：
{
"dataModel":[{"name":"","purpose":"","primaryKey":"","relationships":[""],"retention":"","access":""}],
"dataDictionary":[{"entity":"","field":"","type":"","required":"","description":"","example":"","classification":""}],
"apiContracts":[{"name":"","method":"POST","path":"","auth":"","request":[""],"response":[""],"failures":[""]}],
"dependencies":[{"name":"","owner":"","needBy":"","risk":"","fallback":""}],
"nonFunctional":[{"area":"","requirement":"","verification":"","status":""}],
"testPlan":{"strategy":"","roles":[{"role":"","owner":"","howToTest":"","exit":""}],"cases":[{"id":"TC-001","title":"","level":"P0","steps":[""],"expected":""}],"dataSets":[{"name":"","purpose":"","example":"","notice":""}],"exitRules":[""]},
"deliveryPlan":{"estimate":"","assumptions":[""],"phases":[{"window":"","stage":"","owner":"","outcome":"","entryGate":"","exitGate":"","delayRisk":""}]},
"tracking":[""],"testFocus":[""],"launchChecklist":[""],"monitoring":[""],"rollback":[""],"openQuestions":[""]
}

最少输出：4 个数据实体、6 条数据字典、3 个接口、3 项依赖、4 条非功能要求、4 个测试用例、2 套测试数据、5 个交付阶段、3 个监控项、2 个回滚触发器。
必须承接输入中的第一部分产品与需求基线：保留 REQ ID、MVP、角色、流程和原型名称，不另起一套产品方案。每个 P0 至少一个测试用例，title 用“[REQ-001] 具体场景”关联原需求。dataDictionary.entity 必须匹配 dataModel.name；每个实体的主键都要有字典条目。接口 request/response 使用这些字段，排期阶段说明交付哪些 REQ ID。把研究缺口、需求证据不足、客户基线缺失及前后部分无法对齐的事项写入 openQuestions，并写具体补证动作。`

func isProductDiscoveryTask(question string) bool {
	value := strings.ToLower(strings.TrimSpace(question))
	for _, token := range []string{"产品", "工作台", "mvp", "prd", "原型", "功能", "平台", "方案", "用户场景", "需求"} {
		if strings.Contains(value, token) {
			return true
		}
	}
	return false
}

func (e *ResearchEngine) buildProductBlueprint(ctx context.Context, result ResearchResult) (ProductBlueprint, error) {
	provider, err := reportProviderFromEnv(os.Getenv)
	if err != nil {
		return ProductBlueprint{}, err
	}
	input := productBlueprintInput(result)
	var core productBlueprintCore
	if err := e.requestProductBlueprintPart(ctx, provider, productBlueprintCorePrompt, input, "产品与需求", 11000, &core); err != nil {
		return ProductBlueprint{}, err
	}
	var delivery productBlueprintDelivery
	if err := e.requestProductBlueprintPart(ctx, provider, productBlueprintDeliveryPrompt, productBlueprintDeliveryInput(input, core), "数据与交付", 7800, &delivery); err != nil {
		return ProductBlueprint{}, err
	}
	blueprint := ProductBlueprint{}
	core.applyTo(&blueprint)
	delivery.applyTo(&blueprint)
	if err := validateProductBlueprint(blueprint); err != nil {
		return ProductBlueprint{}, err
	}
	normalizeProductBlueprint(&blueprint, result.Evidence)
	applyProductManagerToolkitReview(&blueprint)
	applyProductBlueprintTraceabilityReview(&blueprint, result)
	return blueprint, nil
}

func (e *ResearchEngine) requestProductBlueprintPart(ctx context.Context, provider reportProvider, prompt, input, stage string, maxTokens int, output any) error {
	var failures []string
	var repairReason string
	for attempt := 1; attempt <= 2; attempt++ {
		instructions := prompt
		if attempt == 2 {
			instructions += "\n\n上一轮未通过本阶段校验，具体问题：" + repairReason + "\n请修复列出的字段与数量，从头重新输出本阶段完整 JSON；不要只返回补丁，不要解释或省略其他字段，也不要在 JSON 之后继续输出文字。不得为凑数量虚构研究事实；设计项需保留假设标记。"
		}
		text, err := e.requestModelText(ctx, provider, instructions, input, maxTokens)
		if err != nil {
			repairReason = "请求失败：" + shorten(err.Error(), 300)
			failures = append(failures, fmt.Sprintf("第%d次请求失败：%s", attempt, shorten(err.Error(), 100)))
			continue
		}
		if err := decodeValidatedProductBlueprintPart(text, output); err != nil {
			repairReason = err.Error()
			failures = append(failures, fmt.Sprintf("第%d次输出无效：%s", attempt, err.Error()))
			continue
		}
		return nil
	}
	return fmt.Errorf("%s蓝图生成失败（已自动重试）：%s", stage, strings.Join(failures, "；"))
}

// Each attempt starts with an empty candidate. An omitted field in a retry must
// not inherit an earlier invalid response's value and appear complete.
func decodeValidatedProductBlueprintPart(text string, output any) error {
	var blueprint ProductBlueprint
	switch target := output.(type) {
	case *productBlueprintCore:
		var candidate productBlueprintCore
		if err := decodeProductBlueprintJSON(text, &candidate); err != nil {
			return err
		}
		candidate.applyTo(&blueprint)
		if gaps := productBlueprintCoreGaps(blueprint); len(gaps) > 0 {
			return fmt.Errorf("产品与需求字段不足：%s", strings.Join(gaps, "；"))
		}
		*target = candidate
	case *productBlueprintDelivery:
		var candidate productBlueprintDelivery
		if err := decodeProductBlueprintJSON(text, &candidate); err != nil {
			return err
		}
		candidate.applyTo(&blueprint)
		if gaps := productBlueprintDeliveryGaps(blueprint); len(gaps) > 0 {
			return fmt.Errorf("数据与交付字段不足：%s", strings.Join(gaps, "；"))
		}
		*target = candidate
	default:
		return fmt.Errorf("不支持的产品蓝图阶段类型 %T", output)
	}
	return nil
}

func decodeProductBlueprintJSON(text string, output any) error {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimSpace(strings.TrimPrefix(text, "```json"))
		text = strings.TrimSpace(strings.TrimPrefix(text, "```"))
		text = strings.TrimSpace(strings.TrimSuffix(text, "```"))
	}
	start, end := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return fmt.Errorf("模型未返回完整 JSON 对象")
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), output); err != nil {
		return fmt.Errorf("JSON 无法解析：%w", err)
	}
	return nil
}

func validateProductBlueprint(blueprint ProductBlueprint) error {
	missing := append(productBlueprintCoreGaps(blueprint), productBlueprintDeliveryGaps(blueprint)...)
	if len(missing) > 0 {
		return fmt.Errorf("产品蓝图不完整：%s", strings.Join(missing, "；"))
	}
	return nil
}

type blueprintMinimumCount struct {
	field   string
	actual  int
	minimum int
}

func blueprintCountGaps(checks ...blueprintMinimumCount) []string {
	var gaps []string
	for _, check := range checks {
		if check.actual < check.minimum {
			gaps = append(gaps, fmt.Sprintf("%s 当前 %d 项，至少需要 %d 项", check.field, check.actual, check.minimum))
		}
	}
	return gaps
}

func productBlueprintCoreGaps(blueprint ProductBlueprint) []string {
	gaps := blueprintCountGaps(
		blueprintMinimumCount{"goals", len(blueprint.Goals), 2},
		blueprintMinimumCount{"nonGoals", len(blueprint.NonGoals), 2},
		blueprintMinimumCount{"successMetrics", len(blueprint.SuccessMetrics), 3},
		blueprintMinimumCount{"users", len(blueprint.Users), 1},
		blueprintMinimumCount{"options", len(blueprint.Options), 2},
		blueprintMinimumCount{"mvp", len(blueprint.MVP), 1},
		blueprintMinimumCount{"userFlow", len(blueprint.UserFlow), 2},
		blueprintMinimumCount{"systemFlow", len(blueprint.SystemFlow), 2},
		blueprintMinimumCount{"sop", len(blueprint.SOP), 4},
		blueprintMinimumCount{"permissions", len(blueprint.Permissions), 3},
		blueprintMinimumCount{"requirements", len(blueprint.Requirements), 4},
		blueprintMinimumCount{"prototype.screens", len(blueprint.Prototype.Screens), 3},
	)
	if strings.TrimSpace(blueprint.ProductName) == "" {
		gaps = append(gaps, "productName 不能为空")
	}
	if strings.TrimSpace(blueprint.OneSentence) == "" {
		gaps = append(gaps, "oneSentence 不能为空")
	}
	return gaps
}

func productBlueprintDeliveryGaps(blueprint ProductBlueprint) []string {
	return blueprintCountGaps(
		blueprintMinimumCount{"dataModel", len(blueprint.DataModel), 4},
		blueprintMinimumCount{"dataDictionary", len(blueprint.DataDictionary), 6},
		blueprintMinimumCount{"apiContracts", len(blueprint.APIContracts), 3},
		blueprintMinimumCount{"dependencies", len(blueprint.Dependencies), 3},
		blueprintMinimumCount{"nonFunctional", len(blueprint.NonFunctional), 4},
		blueprintMinimumCount{"testPlan.cases", len(blueprint.TestPlan.Cases), 4},
		blueprintMinimumCount{"testPlan.dataSets", len(blueprint.TestPlan.DataSets), 2},
		blueprintMinimumCount{"deliveryPlan.phases", len(blueprint.DeliveryPlan.Phases), 5},
		blueprintMinimumCount{"monitoring", len(blueprint.Monitoring), 3},
		blueprintMinimumCount{"rollback", len(blueprint.Rollback), 2},
	)
}

// applyProductManagerToolkitReview makes missing review details visible instead of
// allowing a polished-looking PRD to hide unmodelled exception or analytics work.
func applyProductManagerToolkitReview(blueprint *ProductBlueprint) {
	exceptionText := make([]string, 0, len(blueprint.Requirements))
	acceptanceCount, trackingCount := 0, 0
	for _, requirement := range blueprint.Requirements {
		if strings.EqualFold(requirement.Priority, "P0") {
			exceptionText = append(exceptionText, strings.Join(requirement.Exceptions, " "))
		}
		if hasBlueprintAcceptance(requirement.Acceptance) {
			acceptanceCount++
		}
		if hasBlueprintTracking(requirement.Tracking) {
			trackingCount++
		}
	}
	combined := strings.ToLower(strings.Join(exceptionText, " "))
	labels := []struct {
		label string
		terms []string
	}{
		{"空状态", []string{"空状态", "无数据", "empty"}},
		{"首次使用", []string{"首次", "新用户", "onboarding"}},
		{"权限不足", []string{"权限", "无权", "未授权"}},
		{"网络或超时", []string{"网络", "超时", "timeout"}},
		{"重复提交或幂等", []string{"重复", "幂等", "idempot"}},
		{"并发冲突", []string{"并发", "冲突", "锁", "409"}},
		{"极端数据量", []string{"10 万", "大数据", "极端", "上限"}},
		{"中断恢复", []string{"中断", "恢复", "草稿", "重试"}},
	}
	missing := make([]string, 0, len(labels))
	for _, item := range labels {
		matched := false
		for _, term := range item.terms {
			if strings.Contains(combined, term) {
				matched = true
				break
			}
		}
		if !matched {
			missing = append(missing, item.label)
		}
	}
	if len(missing) > 0 {
		blueprint.OpenQuestions = append(blueprint.OpenQuestions, "PM 工具箱自查：P0 异常覆盖待补充 — "+strings.Join(missing, "、"))
	}
	if acceptanceCount < len(blueprint.Requirements) {
		blueprint.OpenQuestions = append(blueprint.OpenQuestions, "PM 工具箱自查：存在需求缺少 Given/When/Then 验收标准")
	}
	if trackingCount < len(blueprint.Requirements) {
		blueprint.OpenQuestions = append(blueprint.OpenQuestions, "PM 工具箱自查：存在需求缺少事件、触发时机、属性或上报端定义")
	}
}

func hasBlueprintAcceptance(items []string) bool {
	for _, item := range items {
		text := strings.ToLower(item)
		if strings.Contains(text, "given") && strings.Contains(text, "when") && strings.Contains(text, "then") {
			return true
		}
	}
	return false
}

func hasBlueprintTracking(items []string) bool {
	for _, item := range items {
		parts := strings.Split(item, "|")
		if len(parts) != 4 {
			continue
		}
		complete := true
		for _, part := range parts {
			complete = complete && strings.TrimSpace(part) != ""
		}
		if complete {
			return true
		}
	}
	return false
}

func productBlueprintInput(result ResearchResult) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "任务：%s\n\n", shorten(result.Question, 1000))
	if len(result.Scenarios) > 0 {
		// This stage receives the scenario contract, not an unrelated project-review template.
		fmt.Fprintf(&builder, "目标用户业务场景假设（SCN ID 必须关联后续需求/测试；不得当作已验证客户事实）：\n%s\n\n", scenarioBlueprintContext(result.Scenarios))
		builder.WriteString("沿用这些角色、具体触发、输入与产物；P0 用户故事注明关联 SCN ID。验收承接场景 acceptance，数据实体对应场景输入输出。不要把客户使用场景改写为产品经理立项、研发评审或上线评审。\n")
	}
	fmt.Fprintf(&builder, "研究覆盖状态：%s；停止原因：%s\n未解决缺口：%s\n\n", result.Coverage.Status, shorten(result.Coverage.StopReason, 180), shorten(strings.Join(result.Coverage.Gaps, "；"), 900))
	fmt.Fprintf(&builder, "研究报告（只作输入，不可把推断误写成事实）：\n%s\n\n", shorten(result.Report.Analysis, 2200))
	fmt.Fprintf(&builder, "冲突：%s\n限制：%s\n\n", shorten(strings.Join(result.Report.Conflicts, "；"), 600), shorten(strings.Join(result.Report.Limitations, "；"), 600))
	if competition := result.Competition; competition != nil {
		fmt.Fprintf(&builder, "竞品与替代方案（状态：%s；范围：%s）：\n", competition.Status, shorten(competition.Scope, 180))
		for index, profile := range competition.Competitors {
			if index == 6 {
				break
			}
			fmt.Fprintf(&builder, "- %s：定位=%s；用户=%s；工作流=%s；能力=%s；定价=%s；待验证缺口=%s；证据=%s\n", shorten(profile.Name, 60), shorten(profile.Positioning, 100), shorten(profile.TargetUser, 70), shorten(profile.PrimaryWorkflow, 90), shorten(strings.Join(profile.Capabilities, "、"), 120), shorten(profile.Pricing, 90), shorten(strings.Join(profile.Gaps, "、"), 100), blueprintEvidenceIDs(profile.Evidence))
		}
		for index, opportunity := range competition.Opportunities {
			if index == 3 {
				break
			}
			fmt.Fprintf(&builder, "机会假设：%s；验证动作：%s；证据=%s\n", shorten(opportunity.Hypothesis, 160), shorten(opportunity.Validation, 140), blueprintEvidenceIDs(opportunity.Evidence))
		}
		fmt.Fprintf(&builder, "竞品待补资料：%s\n\n", shorten(strings.Join(competition.OpenQuestions, "；"), 600))
	}
	builder.WriteString("允许引用的关键证据（只能使用这些 E#）：\n")
	sources := make(map[string]Source, len(result.Sources))
	for _, source := range result.Sources {
		sources[source.ID] = source
	}
	included := 0
	for _, evidence := range blueprintInputEvidence(result, 40) {
		source := sources[evidence.SourceID]
		row := fmt.Sprintf("[%s] 标题：%s；来源：%s；类型：%s；发表时间：%s；摘录：%s；置信度：%s\n", evidence.ID, shorten(evidence.SourceTitle, 64), shorten(evidence.URL, 120), shorten(source.Kind, 24), shorten(source.PublishedAt, 32), shorten(evidence.Quote, 220), evidence.Confidence)
		if len([]rune(builder.String()))+len([]rune(row)) > 19800 {
			break
		}
		builder.WriteString(row)
		included++
	}
	fmt.Fprintf(&builder, "\n输入选择 %d/%d 条证据，优先覆盖竞品引用及不同来源。缺少直接支持时 evidence 使用 gap；不能用无关 E# 为需求背书。\n", included, len(result.Evidence))
	return builder.String()
}

func blueprintEvidenceIDs(refs []EvidenceRef) string {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.ID != "" {
			ids = append(ids, ref.ID)
		}
	}
	return strings.Join(ids, ",")
}

// Give cited competitors and distinct sources a chance before second/third
// extracts from one long page consume the bounded synthesis context.
func blueprintInputEvidence(result ResearchResult, limit int) []Evidence {
	selected := make([]Evidence, 0, min(limit, len(result.Evidence)))
	seen, sources := make(map[string]bool), make(map[string]bool)
	known := make(map[string]Evidence, len(result.Evidence))
	for _, item := range result.Evidence {
		known[item.ID] = item
	}
	sourceKey := func(item Evidence) string {
		if item.SourceID != "" {
			return item.SourceID
		}
		if item.URL != "" {
			return item.URL
		}
		return item.ID
	}
	add := func(item Evidence) {
		if item.ID == "" || seen[item.ID] || len(selected) >= limit {
			return
		}
		selected = append(selected, item)
		seen[item.ID], sources[sourceKey(item)] = true, true
	}
	for _, scenario := range result.Scenarios {
		for _, link := range scenario.EvidenceLinks {
			add(known[link.ID])
		}
	}
	if result.Competition != nil {
		for _, profile := range result.Competition.Competitors {
			for _, ref := range profile.Evidence {
				add(known[ref.ID])
			}
		}
		for _, opportunity := range result.Competition.Opportunities {
			for _, ref := range opportunity.Evidence {
				add(known[ref.ID])
			}
		}
	}
	for _, item := range result.Evidence {
		if !sources[sourceKey(item)] {
			add(item)
		}
	}
	for _, item := range result.Evidence {
		add(item)
	}
	return selected
}

func productBlueprintDeliveryInput(researchInput string, core productBlueprintCore) string {
	encoded, _ := json.Marshal(core) // This struct contains only serializable values.
	return researchInput + "\n\n第一部分产品与需求基线（是设计草案，不是新增研究事实；第二部分必须沿用其 REQ ID、范围和对象）：\n" + string(encoded)
}

func applyProductBlueprintTraceabilityReview(blueprint *ProductBlueprint, result ResearchResult) {
	for _, gap := range result.Coverage.Gaps {
		blueprint.OpenQuestions = append(blueprint.OpenQuestions, "研究补证："+gap)
	}
	if result.Competition != nil {
		for _, gap := range result.Competition.OpenQuestions {
			blueprint.OpenQuestions = append(blueprint.OpenQuestions, "竞品补证："+gap)
		}
	}
	seenRequirements := make(map[string]bool)
	for _, requirement := range blueprint.Requirements {
		if seenRequirements[requirement.ID] {
			blueprint.OpenQuestions = append(blueprint.OpenQuestions, "需求追溯：重复需求 ID "+requirement.ID+"，需重新编号并修复测试关联")
		}
		seenRequirements[requirement.ID] = true
		if strings.TrimSpace(requirement.UserStory) == "" || len(requirement.MainFlow) == 0 || len(requirement.BusinessRules) == 0 || len(requirement.Permissions) == 0 {
			blueprint.OpenQuestions = append(blueprint.OpenQuestions, "需求规格："+requirement.ID+" 缺用户故事、主流程、业务规则或权限；补齐后才能估时")
		}
		if !strings.EqualFold(requirement.Priority, "P0") {
			continue
		}
		hasEvidence := false
		for _, ref := range requirement.Evidence {
			hasEvidence = hasEvidence || (ref.Type == "public_research" && ref.ID != "")
		}
		if !hasEvidence {
			blueprint.OpenQuestions = append(blueprint.OpenQuestions, "P0 证据缺口："+requirement.ID+" 仍是设计假设；需目标用户访谈或试点验证，不能视为已证实需求")
		}
		linkedTest := false
		for _, test := range blueprint.TestPlan.Cases {
			text := test.Title + " " + strings.Join(test.Steps, " ") + " " + test.Expected
			if requirement.ID != "" && regexp.MustCompile(`\b`+regexp.QuoteMeta(requirement.ID)+`\b`).MatchString(text) {
				linkedTest = true
				break
			}
		}
		if !linkedTest {
			blueprint.OpenQuestions = append(blueprint.OpenQuestions, "测试追溯："+requirement.ID+" 尚无明确引用该 ID 的验收用例")
		}
	}
	entities := make(map[string]DataEntity, len(blueprint.DataModel))
	fields := make(map[string]bool, len(blueprint.DataDictionary))
	for _, entity := range blueprint.DataModel {
		entities[strings.ToLower(strings.TrimSpace(entity.Name))] = entity
	}
	for _, field := range blueprint.DataDictionary {
		entityKey := strings.ToLower(strings.TrimSpace(field.Entity))
		if _, exists := entities[entityKey]; !exists {
			blueprint.OpenQuestions = append(blueprint.OpenQuestions, "数据一致性：字典 "+field.Entity+"."+field.Field+" 指向未定义的实体")
		}
		fields[entityKey+"."+strings.TrimSpace(field.Field)] = true
		if strings.TrimSpace(field.Type) == "" || strings.TrimSpace(field.Required) == "" {
			blueprint.OpenQuestions = append(blueprint.OpenQuestions, "数据字典："+field.Entity+"."+field.Field+" 缺类型或必填约束")
		}
	}
	for _, entity := range blueprint.DataModel {
		key := strings.ToLower(strings.TrimSpace(entity.Name)) + "." + strings.TrimSpace(entity.PrimaryKey)
		if strings.TrimSpace(entity.PrimaryKey) == "" || !fields[key] {
			blueprint.OpenQuestions = append(blueprint.OpenQuestions, "数据一致性：实体 "+entity.Name+" 的主键缺少字典定义")
		}
	}
	for _, metric := range blueprint.SuccessMetrics {
		if metric.Definition == "" || metric.Baseline == "" || metric.Target == "" || metric.Window == "" || metric.Owner == "" {
			blueprint.OpenQuestions = append(blueprint.OpenQuestions, "指标口径："+metric.Name+" 需补定义、基线采集动作、目标、周期和负责人")
		}
		if blueprintUnqualifiedMetricNumber(metric.Baseline, result.Evidence) {
			blueprint.OpenQuestions = append(blueprint.OpenQuestions, "指标基线："+metric.Name+" 的数值没有引用本次证据或标注未知/假设；需提供真实业务记录后核验")
		}
		if blueprintUnqualifiedMetricNumber(metric.Target, result.Evidence) {
			blueprint.OpenQuestions = append(blueprint.OpenQuestions, "指标目标："+metric.Name+" 的数值应标为目标假设并注明验证方式，不能作为已验证结论")
		}
	}
	blueprint.OpenQuestions = uniqueStrings(blueprint.OpenQuestions)
}

func blueprintUnqualifiedMetricNumber(value string, evidence []Evidence) bool {
	if !regexp.MustCompile(`[0-9０-９]`).MatchString(value) {
		return false
	}
	lower := strings.ToLower(value)
	for _, qualifier := range []string{"未知", "假设", "待验证", "待试点验证", "unknown", "hypothes"} {
		if strings.Contains(lower, qualifier) {
			return false
		}
	}
	for _, item := range evidence {
		if item.ID != "" && regexp.MustCompile(`\b`+regexp.QuoteMeta(item.ID)+`\b`).MatchString(value) {
			return false
		}
	}
	return true
}

func decodeProductBlueprint(text string) (ProductBlueprint, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimSpace(strings.TrimPrefix(text, "```json"))
		text = strings.TrimSpace(strings.TrimPrefix(text, "```"))
		text = strings.TrimSpace(strings.TrimSuffix(text, "```"))
	}
	start, end := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return ProductBlueprint{}, fmt.Errorf("产品蓝图模型未返回 JSON 对象")
	}
	var blueprint ProductBlueprint
	if err := json.Unmarshal([]byte(text[start:end+1]), &blueprint); err != nil {
		return ProductBlueprint{}, fmt.Errorf("产品蓝图 JSON 无法解析：%w", err)
	}
	return blueprint, nil
}

func normalizeProductBlueprint(blueprint *ProductBlueprint, evidence []Evidence) {
	allowed := make(map[string]EvidenceRef, len(evidence))
	for _, item := range evidence {
		label := strings.TrimSpace(item.SourceTitle)
		if label == "" {
			label = "公开研究来源"
		}
		allowed[item.ID] = EvidenceRef{ID: item.ID, Label: item.ID + " · " + label, Type: "public_research"}
	}
	normalizeRefs := func(refs []EvidenceRef) []EvidenceRef {
		items := make([]EvidenceRef, 0, len(refs))
		seen := make(map[string]bool)
		for _, ref := range refs {
			if found, ok := allowed[strings.TrimSpace(ref.ID)]; ok && !seen[found.ID] {
				items, seen[found.ID] = append(items, found), true
			}
		}
		if len(items) == 0 {
			return []EvidenceRef{{Label: "待通过访谈或试点验证", Type: "gap"}}
		}
		return items
	}
	for index := range blueprint.Users {
		blueprint.Users[index].Evidence = normalizeRefs(blueprint.Users[index].Evidence)
	}
	for index := range blueprint.Requirements {
		requirement := &blueprint.Requirements[index]
		if strings.TrimSpace(requirement.ID) == "" {
			requirement.ID = fmt.Sprintf("REQ-%03d", index+1)
		}
		if strings.TrimSpace(requirement.Priority) == "" {
			requirement.Priority = "P1"
		}
		requirement.Evidence = normalizeRefs(requirement.Evidence)
	}
	for index := range blueprint.SOP {
		if blueprint.SOP[index].Order <= 0 {
			blueprint.SOP[index].Order = index + 1
		}
	}
	for index := range blueprint.Prototype.Screens {
		if strings.TrimSpace(blueprint.Prototype.Screens[index].ID) == "" {
			blueprint.Prototype.Screens[index].ID = fmt.Sprintf("SCR-%02d", index+1)
		}
	}
}
