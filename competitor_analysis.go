package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type CompetitorAnalysis struct {
	Status        string                  `json:"status"`
	Scope         string                  `json:"scope"`
	Coverage      string                  `json:"coverage"`
	Competitors   []CompetitorProfile     `json:"competitors"`
	Opportunities []CompetitorOpportunity `json:"opportunities"`
	OpenQuestions []string                `json:"openQuestions"`
}

type CompetitorProfile struct {
	Name            string        `json:"name"`
	Category        string        `json:"category"`
	Positioning     string        `json:"positioning"`
	TargetUser      string        `json:"targetUser"`
	PrimaryWorkflow string        `json:"primaryWorkflow"`
	Capabilities    []string      `json:"capabilities"`
	Pricing         string        `json:"pricing"`
	Gaps            []string      `json:"gaps"`
	Confidence      string        `json:"confidence"`
	Evidence        []EvidenceRef `json:"evidence"`
}

type CompetitorOpportunity struct {
	Hypothesis string        `json:"hypothesis"`
	WhiteSpace string        `json:"whiteSpace"`
	Validation string        `json:"validation"`
	Evidence   []EvidenceRef `json:"evidence"`
}

const competitorAnalysisPrompt = `你是一位严格证据约束的产品战略研究员。请只根据提供的公开来源，为一个研究任务制作竞品与替代方案分析。

只输出合法 JSON，不要 Markdown、代码块或解释。格式：
{
  "status":"ready | insufficient_evidence",
  "scope":"",
  "coverage":"",
  "competitors":[{
    "name":"", "category":"", "positioning":"", "targetUser":"", "primaryWorkflow":"",
    "capabilities":[""], "pricing":"", "gaps":[""], "confidence":"高|中|低",
    "evidence":[{"id":"E1"}]
  }],
  "opportunities":[{"hypothesis":"", "whiteSpace":"", "validation":"", "evidence":[{"id":"E1"}]}],
  "openQuestions":[""]
}

硬规则：
- 只可使用输入中的 Evidence ID。每个竞品、价格、功能、机会判断至少有一个 Evidence ID；没有证据就不要输出。
- 名称必须在来源标题或正文中被明确提及；不得根据网址、常识或模型记忆猜测竞品。
- pricing 只有在来源明确提及收费或价格时才可写；否则写“公开资料未核验到定价”。
- “空白机会”只能是待验证假设，必须写验证动作，不能表示为市场事实。
- 最多 6 个竞品、3 个机会假设。资料不够时 status 使用 insufficient_evidence，并在 openQuestions 指出还需找什么。`

func wantsCompetitorAnalysis(question string) bool {
	value := strings.ToLower(strings.TrimSpace(question))
	if isProductDiscoveryTask(question) {
		return true
	}
	for _, token := range []string{"竞品", "竞争", "市场", "创业", "赛道", "compet", "market", "startup", "pricing", "alternative"} {
		if strings.Contains(value, token) {
			return true
		}
	}
	return false
}

func (e *ResearchEngine) buildCompetitorAnalysis(ctx context.Context, result ResearchResult) (CompetitorAnalysis, error) {
	fallback := competitorAnalysisFallback(result)
	if len(result.Evidence) < 3 || result.Stats.Read < 3 {
		return fallback, errors.New("已读竞品相关公开资料不足，未生成未经证实的竞品画像")
	}
	provider, err := reportProviderFromEnv(os.Getenv)
	if err != nil {
		return fallback, err
	}
	text, err := e.requestModelText(ctx, provider, competitorAnalysisPrompt, competitorAnalysisInput(result), 4000)
	if err != nil {
		return fallback, err
	}
	var analysis CompetitorAnalysis
	if err := decodeProductBlueprintJSON(text, &analysis); err != nil {
		return fallback, fmt.Errorf("竞品分析 JSON 无法解析：%w", err)
	}
	analysis = sanitizeCompetitorAnalysis(analysis, result)
	if len(analysis.Competitors) == 0 {
		return fallback, errors.New("未从已读公开资料中提取到带证据的竞品画像")
	}
	// Valid citation IDs alone do not establish a complete competitive landscape.
	analysis.Status = "insufficient_evidence"
	if len(analysis.Competitors) >= 3 && result.Coverage.Status == "sufficient" {
		analysis.Status = "ready"
	}
	analysis.OpenQuestions = uniqueStrings(append(analysis.OpenQuestions, result.Coverage.Gaps...))
	if len(analysis.Competitors) < 3 {
		analysis.OpenQuestions = append(analysis.OpenQuestions, "尚不足 3 个已核验竞品画像，不代表市场只有这些产品；继续补官方产品、价格和真实使用证据。")
	}
	if analysis.Scope == "" {
		analysis.Scope = "围绕“" + shorten(result.Question, 100) + "”的已读公开资料竞品与替代方案比较"
	}
	if analysis.Coverage == "" {
		analysis.Coverage = fmt.Sprintf("已读取 %d 个来源，使用 %d 条可追溯证据；未列出的产品不代表不存在。", result.Stats.Read, len(result.Evidence))
	}
	return analysis, nil
}

func competitorAnalysisInput(result ResearchResult) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "研究问题：%s\n已读取来源：%d；证据：%d\n\n", result.Question, result.Stats.Read, len(result.Evidence))
	builder.WriteString(coveragePrompt(result.Coverage))
	builder.WriteString("可用证据：\n")
	for _, item := range blueprintInputEvidence(result, 40) {
		fmt.Fprintf(&builder, "[%s] 标题：%s\n网址：%s\n发布日期：%s；读取日期：%s\n摘录：%s\n\n", item.ID, shorten(item.SourceTitle, 100), item.URL, item.PublishedAt, item.FetchedAt.Format("2006-01-02"), shorten(item.Quote, 400))
	}
	builder.WriteString("已读来源正文摘要（仅用于确认名称与上下文，不得将未出现的信息补成事实）：\n")
	for index, source := range result.Sources {
		if source.Status != "已阅读" || strings.TrimSpace(source.Content) == "" {
			continue
		}
		fmt.Fprintf(&builder, "来源 %d：%s\n%s\n\n", index+1, shorten(source.Title, 160), shorten(source.Content, 900))
	}
	return shorten(builder.String(), 24000)
}

func sanitizeCompetitorAnalysis(analysis CompetitorAnalysis, result ResearchResult) CompetitorAnalysis {
	known := make(map[string]EvidenceRef, len(result.Evidence))
	quotes := make(map[string]Evidence, len(result.Evidence))
	for _, item := range result.Evidence {
		known[item.ID] = EvidenceRef{ID: item.ID, Label: item.ID + " · " + shorten(item.SourceTitle, 100), Type: "public_research"}
		quotes[item.ID] = item
	}
	profiles := make([]CompetitorProfile, 0, min(len(analysis.Competitors), 6))
	seen := make(map[string]bool)
	for _, profile := range analysis.Competitors {
		profile.Name = shorten(strings.TrimSpace(profile.Name), 100)
		key := strings.ToLower(profile.Name)
		if profile.Name == "" || seen[key] {
			continue
		}
		profile.Evidence = validCompetitorEvidence(profile.Evidence, known, 3)
		// Require the cited evidence itself to mention the named competitor.
		matchedRefs := []EvidenceRef{}
		var support strings.Builder
		for _, ref := range profile.Evidence {
			item := quotes[ref.ID]
			if strings.Contains(strings.ToLower(item.SourceTitle+" "+item.Quote), key) {
				matchedRefs = append(matchedRefs, ref)
				support.WriteString(item.Quote + "\n")
			}
		}
		profile.Evidence = matchedRefs
		if len(profile.Evidence) == 0 {
			continue
		}
		seen[key] = true
		profile.Category = shorten(strings.TrimSpace(profile.Category), 100)
		profile.Positioning = shorten(strings.TrimSpace(profile.Positioning), 360)
		profile.TargetUser = shorten(strings.TrimSpace(profile.TargetUser), 240)
		profile.PrimaryWorkflow = shorten(strings.TrimSpace(profile.PrimaryWorkflow), 300)
		profile.Pricing = shorten(strings.TrimSpace(profile.Pricing), 240)
		if profile.Pricing == "" || !supportedCompetitorPricing(profile.Pricing, support.String()) {
			profile.Pricing = "公开资料未核验到定价"
		}
		profile.Capabilities = uniqueStrings(trimmedCompetitorStrings(profile.Capabilities, 6, 150))
		profile.Gaps = uniqueStrings(trimmedCompetitorStrings(profile.Gaps, 4, 180))
		if profile.Confidence != "高" && profile.Confidence != "中" && profile.Confidence != "低" {
			profile.Confidence = "低"
		}
		profiles = append(profiles, profile)
	}
	opportunities := make([]CompetitorOpportunity, 0, min(len(analysis.Opportunities), 3))
	for _, opportunity := range analysis.Opportunities {
		opportunity.Hypothesis = shorten(strings.TrimSpace(opportunity.Hypothesis), 300)
		opportunity.WhiteSpace = shorten(strings.TrimSpace(opportunity.WhiteSpace), 360)
		opportunity.Validation = shorten(strings.TrimSpace(opportunity.Validation), 300)
		opportunity.Evidence = validCompetitorEvidence(opportunity.Evidence, known, 3)
		if opportunity.Hypothesis == "" || opportunity.Validation == "" || len(opportunity.Evidence) == 0 {
			continue
		}
		opportunities = append(opportunities, opportunity)
	}
	analysis.Competitors = profiles
	analysis.Opportunities = opportunities
	analysis.OpenQuestions = uniqueStrings(trimmedCompetitorStrings(analysis.OpenQuestions, 5, 280))
	return analysis
}

var pricingNumberRE = regexp.MustCompile(`[0-9]+(?:[.,][0-9]+)*`)

func supportedCompetitorPricing(claim, support string) bool {
	if !containsAny(strings.ToLower(support), "定价", "收费", "套餐", "免费", "价格", "pricing", "price", "billing", "free", "subscription", "$/", "¥") {
		return false
	}
	known := map[string]bool{}
	for _, n := range pricingNumberRE.FindAllString(support, -1) {
		known[n] = true
	}
	for _, n := range pricingNumberRE.FindAllString(claim, -1) {
		if !known[n] {
			return false
		}
	}
	return true
}

func validCompetitorEvidence(items []EvidenceRef, known map[string]EvidenceRef, limit int) []EvidenceRef {
	output := make([]EvidenceRef, 0, limit)
	seen := make(map[string]bool)
	for _, item := range items {
		item, ok := known[strings.TrimSpace(item.ID)]
		if !ok || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		output = append(output, item)
		if len(output) == limit {
			break
		}
	}
	return output
}

func trimmedCompetitorStrings(items []string, limit, maxLen int) []string {
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

func competitorAnalysisFallback(result ResearchResult) CompetitorAnalysis {
	return CompetitorAnalysis{
		Status:   "insufficient_evidence",
		Scope:    "围绕“" + shorten(result.Question, 100) + "”的竞品与替代方案分析",
		Coverage: fmt.Sprintf("当前只读取 %d 个来源、提取 %d 条证据；不足以可靠列出竞品、功能和定价。", result.Stats.Read, len(result.Evidence)),
		OpenQuestions: []string{
			"每个候选竞品的官方产品页、功能文档和定价页分别是什么？",
			"目标用户在什么工作流中选择、替代或放弃这些方案？",
			"哪些能力是基础必备，哪些能力仍是待验证的差异化机会？",
		},
	}
}
