package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/publicsuffix"
)

// Limits are ceilings, not promises of research quality or source availability.
type ResearchPolicy struct {
	MaxRounds, MaxQueries, MaxSources, ReadBatch                          int
	TargetSources, TargetDomains, TargetEvidence, MaxEvidence, MaxSeconds int
}

func defaultResearchPolicy() ResearchPolicy {
	return ResearchPolicy{5, 30, 36, 8, 20, 8, 40, 80, 240}
}

func loadResearchPolicy() ResearchPolicy {
	p := defaultResearchPolicy()
	data, err := os.ReadFile("research_policy.yaml")
	if err != nil {
		return p
	}
	fields := map[string]*int{"max_rounds": &p.MaxRounds, "max_queries": &p.MaxQueries, "max_sources": &p.MaxSources, "read_batch": &p.ReadBatch, "target_sources": &p.TargetSources, "target_domains": &p.TargetDomains, "target_evidence": &p.TargetEvidence, "max_evidence": &p.MaxEvidence, "max_seconds": &p.MaxSeconds}
	for _, line := range strings.Split(string(data), "\n") {
		key, raw, ok := strings.Cut(strings.SplitN(line, "#", 2)[0], ":")
		if !ok {
			continue
		}
		if dst := fields[strings.TrimSpace(key)]; dst != nil {
			if value, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && value > 0 {
				*dst = value
			}
		}
	}
	p.MaxRounds = min(p.MaxRounds, 8)
	p.MaxQueries = min(p.MaxQueries, 48)
	p.MaxSources = min(p.MaxSources, 60)
	p.ReadBatch = min(p.ReadBatch, min(p.MaxSources, maxPagesPerRun))
	p.MaxEvidence = min(p.MaxEvidence, 120)
	p.MaxSeconds = min(p.MaxSeconds, 600)
	p.TargetSources = min(p.TargetSources, p.MaxSources)
	p.TargetDomains = min(p.TargetDomains, p.TargetSources)
	p.TargetEvidence = min(p.TargetEvidence, p.MaxEvidence)
	return p
}

type CoverageDimension struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Sources     int      `json:"sources"`
	Domains     int      `json:"domains"`
	EvidenceIDs []string `json:"evidenceIds"`
	Status      string   `json:"status"`
}

type ResearchCoverage struct {
	Status          string              `json:"status"`
	Round           int                 `json:"round"`
	MaxRounds       int                 `json:"maxRounds"`
	SourcesRead     int                 `json:"sourcesRead"`
	EvidenceSources int                 `json:"evidenceSources"`
	Domains         int                 `json:"domains"`
	EvidenceCount   int                 `json:"evidenceCount"`
	TargetSources   int                 `json:"targetSources"`
	TargetDomains   int                 `json:"targetDomains"`
	TargetEvidence  int                 `json:"targetEvidence"`
	Dimensions      []CoverageDimension `json:"dimensions"`
	Gaps            []string            `json:"gaps"`
	StopReason      string              `json:"stopReason"`
}

type researchDimension struct {
	id, name string
	terms    []string
	queries  []string
}

var marketDimensions = []researchDimension{
	{"market", "市场与统计口径", []string{"市场", "market", "规模", "增长", "survey", "统计"}, []string{"市场规模 统计 数据 报告", "market size adoption survey methodology", "行业报告 样本 统计口径 filetype:pdf"}},
	{"users", "用户与实际场景", []string{"用户", "客户", "user", "customer", "案例", "workflow", "流程"}, []string{"用户 痛点 客户案例 工作流", "customer case study workflow limitations", "用户访谈 试点 客户反馈"}},
	{"competition", "竞品与替代方案", []string{"竞品", "竞争", "compet", "alternative", "产品", "platform", "平台"}, []string{"竞品 替代方案 官方 产品文档", "alternatives product documentation comparison", "竞争格局 官方 功能 差异"}},
	{"pricing", "定价与商业模式", []string{"价格", "定价", "收费", "套餐", "计费", "pricing", "price", "billing", "subscription", "免费"}, []string{"官网 定价 套餐 计费", "official pricing plans billing limits", "订阅 单价 按量计费 成本"}},
	{"technology", "技术与可行性", []string{"架构", "技术", "部署", "文档", "api", "documentation", "evaluation", "benchmark", "性能"}, []string{"官方文档 部署 架构 限制", "documentation deployment benchmark evaluation", "技术评估 性能 开源 论文"}},
	{"risk", "风险与反证", []string{"风险", "失败", "限制", "安全", "监管", "risk", "failure", "limitation", "security", "compliance"}, []string{"失败案例 风险 安全 限制", "failure cases limitations security compliance", "监管 风险 反证 项目失败"}},
}

type ResearchSearchPlan struct {
	Topic           string   `json:"topic"`
	PaperQuery      string   `json:"paperQuery"`
	Candidates      []string `json:"candidates"`
	ScenarioQueries []string `json:"scenarioQueries,omitempty"`
}

func fallbackSearchPlan(question string) ResearchSearchPlan {
	topic := strings.TrimSpace(question)
	for _, phrase := range []string{"现在呢", "如果我要做一个", "我应该怎么做", "我应该去怎么做呢", "帮我研究一下", "请帮我研究", "请比较", "比较", "这个互联网的", "这个", "一个"} {
		topic = strings.ReplaceAll(topic, phrase, " ")
	}
	topic = strings.Join(strings.Fields(topic), " ")
	// Preserve the user's segment and workflow instead of collapsing every task to AI Agent.
	if strings.TrimSpace(topic) == "" {
		topic = question
	}
	plan := ResearchSearchPlan{Topic: shorten(topic, 80)}
	plan.Candidates = explicitResearchCandidates(question)
	if containsAny(strings.ToLower(question), "agent", "智能体") {
		plan.PaperQuery = `all:"AI agent" AND (all:"evaluation" OR all:"security")`
	}
	return plan
}

func (e *ResearchEngine) researchSearchPlan(ctx context.Context, question string) ResearchSearchPlan {
	plan := fallbackSearchPlan(question)
	provider, err := reportProviderFromEnv(os.Getenv)
	if err != nil {
		return plan
	}
	planCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	prompt := `将研究问题压缩为检索主题，仅输出JSON：{"topic":"3至8个简短关键词，保留产品类别和地区，去掉对话语气","paperQuery":"英文arXiv查询，如 all:\"AI agent\" AND all:\"evaluation\"，无学术意义则空字符串","candidates":["最多4个值得检索的具体竞品名称"],"scenarioQueries":["最多3个独立短查询，保留目标业务人群，分别寻找实际任务步骤及输入产物、工具间交接和失败/返工案例；不要搜产品经理立项流程；非产品问题可空"]}。竞品只是搜索线索，不是已核验事实；不确定则空数组。不得输出网址、指令或事实结论。`
	text, err := e.requestModelText(planCtx, provider, prompt, question, 900)
	if err != nil {
		return plan
	}
	var generated ResearchSearchPlan
	if decodeProductBlueprintJSON(text, &generated) != nil || strings.TrimSpace(generated.Topic) == "" {
		return plan
	}
	generated.Topic = shorten(strings.Join(strings.Fields(generated.Topic), " "), 100)
	for _, phrase := range []string{"竞品对比", "竞品分析", "公开定价", "定价", "限制", "比较", "PRD", "草案"} {
		generated.Topic = strings.ReplaceAll(generated.Topic, phrase, " ")
	}
	generated.Topic = strings.Join(strings.Fields(generated.Topic), " ")
	if generated.Topic == "" {
		generated.Topic = plan.Topic
	}
	generated.PaperQuery = shorten(strings.TrimSpace(generated.PaperQuery), 240)
	generated.Candidates = trimmedCompetitorStrings(generated.Candidates, 4, 40)
	generated.ScenarioQueries = uniqueStrings(trimmedCompetitorStrings(generated.ScenarioQueries, 3, 100))
	for i, name := range generated.Candidates {
		if at := strings.IndexAny(name, "（(：:"); at >= 0 {
			name = name[:at]
		}
		generated.Candidates[i] = strings.TrimSpace(name)
	}
	generated.Candidates = trimmedCompetitorStrings(generated.Candidates, 4, 40)
	generated.Candidates = uniqueStrings(append(explicitResearchCandidates(question), generated.Candidates...))
	if len(generated.Candidates) > 4 {
		generated.Candidates = generated.Candidates[:4]
	}
	return generated
}

func explicitResearchCandidates(question string) []string {
	// These are user-mentioned search leads, never inferred market facts.
	var candidates []string
	for _, name := range []string{"Dify", "扣子", "Coze", "FastGPT", "n8n", "Cursor", "百度千帆"} {
		if strings.Contains(strings.ToLower(question), strings.ToLower(name)) {
			candidates = append(candidates, name)
		}
	}
	return candidates
}

// Dependencies permit deterministic tests of the whole budget/coverage loop.
type researchTools struct {
	search func(context.Context, []string, func(ProgressEvent), string) []searchHit
	papers func(context.Context, string) ([]searchHit, error)
	read   func(context.Context, []searchHit, func(ProgressEvent)) []Source
}

func (e *ResearchEngine) gatherResearch(ctx context.Context, result *ResearchResult, emit func(ProgressEvent)) error {
	policy := loadResearchPolicy()
	budgetCtx, cancel := context.WithTimeout(ctx, time.Duration(policy.MaxSeconds)*time.Second)
	defer cancel()
	emit(progress("plan", "scope", "正在拆分市场、用户、竞品、定价、技术与反证的检索主题。", 1, 5))
	plan := e.researchSearchPlan(budgetCtx, result.Question)
	emit(progress("plan", "scope", fmt.Sprintf("主题：%s；最多 %d 轮、%d 次查询、%d 个阅读尝试。", plan.Topic, policy.MaxRounds, policy.MaxQueries, policy.MaxSources), 1, 5))
	if len(plan.Candidates) > 0 {
		emit(progress("plan", "scope", "待核验竞品线索："+strings.Join(plan.Candidates, "、"), 1, 5))
	}
	runEvidenceLoop(budgetCtx, result, policy, plan, researchTools{e.collectHits, e.searchPapers, e.readSources}, emit)
	// A research budget timeout still permits a bounded report; user cancellation does not.
	return ctx.Err()
}

func runEvidenceLoop(ctx context.Context, result *ResearchResult, policy ResearchPolicy, plan ResearchSearchPlan, tools researchTools, emit func(ProgressEvent)) {
	var hits []searchHit
	seenQueries, attempted := map[string]bool{}, map[string]bool{}
	stagnant, previousEvidence := 0, 0
	for round := 1; round <= policy.MaxRounds; round++ {
		if ctx.Err() != nil {
			result.Coverage.StopReason = "time_budget"
			break
		}
		queries := adaptiveQueries(plan, result.Coverage, round)
		var fresh []string
		for _, q := range queries {
			if !seenQueries[q] && result.Stats.Queries+len(fresh) < policy.MaxQueries {
				seenQueries[q] = true
				fresh = append(fresh, q)
			}
		}
		if len(fresh) > 0 {
			emit(progress("plan", "verify", fmt.Sprintf("第 %d/%d 轮：按覆盖缺口检索 %d 个问题。", round, policy.MaxRounds, len(fresh)), round, policy.MaxRounds))
			result.Stats.Queries += len(fresh)
			hits = append(hits, tools.search(ctx, fresh, emit, "verify")...)
		}
		if round == 1 && plan.PaperQuery != "" && result.Stats.Queries < policy.MaxQueries && ctx.Err() == nil {
			result.Stats.Queries++
			emit(progress("search", "verify", "论文检索："+plan.PaperQuery, round, policy.MaxRounds))
			papers, err := tools.papers(ctx, plan.PaperQuery)
			if err == nil {
				hits = append(hits, papers...)
			} else {
				emit(progress("warning", "verify", "论文索引暂不可用，保留技术证据缺口。", round, policy.MaxRounds))
			}
		}
		hits = deduplicateHits(hits)
		result.Stats.Discovered = len(hits)
		relevanceTopic := plan.Topic + " " + strings.Join(plan.Candidates, " ")
		batch := selectResearchBatch(relevanceTopic, hits, attempted, result.Sources, min(policy.ReadBatch, policy.MaxSources-len(attempted)))
		for _, h := range batch {
			attempted[canonicalURL(h.URL)] = true
		}
		if len(batch) > 0 && ctx.Err() == nil {
			result.Sources = append(result.Sources, tools.read(ctx, batch, emit)...)
		}
		result.Evidence = extractResearchEvidence(relevanceTopic, result.Sources, policy.MaxEvidence)
		updateResearchStats(result)
		result.Coverage = assessResearchCoverage(*result, policy, round)
		if result.Coverage.Status == "sufficient" {
			result.Coverage.StopReason = "coverage_target"
		}
		if len(result.Evidence) <= previousEvidence {
			stagnant++
		} else {
			stagnant = 0
		}
		previousEvidence = len(result.Evidence)
		if result.Coverage.StopReason == "" {
			switch {
			case ctx.Err() != nil:
				result.Coverage.StopReason = "time_budget"
			case len(attempted) >= policy.MaxSources:
				result.Coverage.StopReason = "source_budget"
			case stagnant >= 2:
				result.Coverage.StopReason = "no_new_evidence"
			case round == policy.MaxRounds:
				result.Coverage.StopReason = "round_budget"
			case len(fresh) == 0 && len(batch) == 0:
				result.Coverage.StopReason = "query_budget"
			}
		}
		coverage := result.Coverage
		emit(ProgressEvent{Kind: "coverage", Step: "verify", Message: fmt.Sprintf("第 %d 轮：读取 %d 个来源，%d 个域名，%d 条证据，剩余 %d 项覆盖缺口。", round, coverage.SourcesRead, coverage.Domains, coverage.EvidenceCount, len(coverage.Gaps)), Discovered: result.Stats.Discovered, EvidenceCount: len(result.Evidence), Coverage: &coverage, At: time.Now().UTC()})
		if coverage.StopReason != "" {
			break
		}
	}
	if result.Coverage.MaxRounds == 0 {
		result.Coverage = assessResearchCoverage(*result, policy, 0)
		result.Coverage.StopReason = "time_budget"
	}
}

func adaptiveQueries(plan ResearchSearchPlan, coverage ResearchCoverage, round int) []string {
	queries := []string{}
	allCovered := len(coverage.Dimensions) > 0
	for _, item := range coverage.Dimensions {
		if item.Status != "covered" {
			allCovered = false
		}
	}
	for _, dim := range marketDimensions {
		covered := false
		for _, item := range coverage.Dimensions {
			if item.ID == dim.id && item.Status == "covered" {
				covered = true
			}
		}
		specificScenarioQuery := dim.id == "users" && round <= len(plan.ScenarioQueries)
		if covered && round > 1 && !allCovered && !specificScenarioQuery {
			continue
		}
		variant := (round - 1) % len(dim.queries)
		query := plan.Topic + " " + dim.queries[variant]
		if specificScenarioQuery {
			query = plan.ScenarioQueries[round-1]
		}
		if round > 3 {
			query += fmt.Sprintf(" %d %s", time.Now().Year(), []string{"原始资料 methodology", "最新案例 原始数据"}[(round-4)%2])
		}
		queries = append(queries, query)
	}
	if round > 1 {
		for i := 0; i < min(2, len(plan.Candidates)); i++ {
			name := plan.Candidates[(((round-2)/2)*2+i)%len(plan.Candidates)]
			suffix := []string{" documentation", " pricing", " documentation features", " pricing limits"}[(round-2)%4]
			if hasHan(name) {
				suffix = []string{" 官方文档", " 定价", " 官方文档 功能", " 定价 使用限制"}[(round-2)%4]
			}
			queries = append(queries, name+suffix)
		}
	}
	if len(queries) == 0 {
		queries = []string{plan.Topic + " 独立研究 数据集 失败案例 " + strconv.Itoa(time.Now().Year())}
	}
	return queries[:min(8, len(queries))]
}

func sourceDomain(raw string) string {
	domain := domainOf(raw)
	if parent, err := publicsuffix.EffectiveTLDPlusOne(domain); err == nil {
		return parent
	}
	return domain
}

func selectResearchBatch(topic string, hits []searchHit, attempted map[string]bool, sources []Source, limit int) []searchHit {
	if limit <= 0 {
		return nil
	}
	pool := append([]searchHit(nil), hits...)
	sort.SliceStable(pool, func(i, j int) bool {
		a, b := topicMatches(topic, pool[i].Title+" "+pool[i].Snippet+" "+pool[i].Abstract), topicMatches(topic, pool[j].Title+" "+pool[j].Snippet+" "+pool[j].Abstract)
		if a != b {
			return a > b
		}
		return pool[i].URL < pool[j].URL
	})
	domains := map[string]int{}
	for _, source := range sources {
		domains[sourceDomain(source.URL)]++
	}
	selected := []searchHit{}
	selectedURLs := map[string]bool{}
	// First pass broadens domains; second permits up to three pages per publisher.
	for _, domainLimit := range []int{1, 3} {
		for _, hit := range pool {
			url := canonicalURL(hit.URL)
			d := sourceDomain(hit.URL)
			if attempted[url] || selectedURLs[url] || domains[d] >= domainLimit || url == "" {
				continue
			}
			// Technical paper titles/abstracts must match the topic, not merely arXiv's rank.
			if hit.Abstract != "" && topicMatches(topic, hit.Title+" "+hit.Abstract) == 0 {
				continue
			}
			selected = append(selected, hit)
			selectedURLs[url] = true
			domains[d]++
			if len(selected) == limit {
				return selected
			}
		}
	}
	return selected
}

var englishTopicRE = regexp.MustCompile(`[a-z][a-z0-9_-]+`)

func researchTerms(topic string) []string {
	low := strings.ToLower(topic)
	terms := []string{}
	stop := map[string]bool{"the": true, "and": true, "for": true, "with": true, "what": true, "how": true, "china": true, "chinese": true, "enterprise": true, "market": true, "ai": true}
	for _, s := range englishTopicRE.FindAllString(low, -1) {
		if len(s) > 2 && !stop[s] {
			terms = append(terms, s)
		}
	}
	for _, s := range []string{"智能体", "工作台", "客服", "自媒体", "电商", "编程", "核能", "医疗", "教育", "物流", "金融", "agent"} {
		if strings.Contains(low, s) {
			terms = append(terms, s)
		}
	}
	// CJK bigrams capture the whole question rather than only its first six characters.
	runes := []rune(low)
	for i := 1; i < len(runes); i++ {
		if unicode.Is(unicode.Han, runes[i-1]) && unicode.Is(unicode.Han, runes[i]) {
			pair := string(runes[i-1 : i+1])
			if !containsAny("中国 企业 市场 比较 研究 如何 什么 一个 这个 现在 应该 进行 关于 以及", pair) {
				terms = append(terms, pair)
			}
		}
	}
	return uniqueStrings(terms)
}

func topicMatches(topic, text string) int {
	low := strings.ToLower(text)
	matches := 0
	for _, term := range researchTerms(topic) {
		if strings.Contains(low, term) {
			matches++
		}
	}
	return matches
}

func extractResearchEvidence(topic string, sources []Source, limit int) []Evidence {
	if limit <= 0 {
		return nil
	}
	ordered := append([]Source(nil), sources...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	items := []Evidence{}
	seen := map[string]bool{}
	for _, source := range ordered {
		if source.Status != "已阅读" || topicMatches(topic, source.Title+" "+source.Content) == 0 {
			continue
		}
		type candidate struct {
			quote string
			score int
		}
		candidates := []candidate{}
		for _, passage := range splitSentences(source.Content) {
			passage = strings.TrimSpace(passage)
			n := len([]rune(passage))
			if n < 35 {
				continue
			}
			// Longer English paragraphs are kept as meaningful chunks rather than discarded.
			for offset := 0; offset < n; offset += 650 {
				chunk := string([]rune(passage)[offset:min(n, offset+650)])
				if len([]rune(chunk)) < 35 {
					continue
				}
				matched := topicMatches(topic, chunk)
				// Pricing/limits paragraphs may omit the product name repeated in the page title.
				if matched == 0 && topicMatches(topic, source.Title) > 0 {
					for _, dim := range marketDimensions {
						if containsAny(strings.ToLower(chunk), dim.terms...) {
							matched = 1
							break
						}
					}
				}
				if matched == 0 {
					continue
				}
				score := min(85, source.Trust/2+min(matched, 4)*10)
				candidates = append(candidates, candidate{chunk, score})
			}
		}
		sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
		count := 0
		for _, candidate := range candidates {
			key := strings.ToLower(strings.Join(strings.Fields(candidate.quote), ""))
			if seen[key] {
				continue
			}
			seen[key] = true
			confidence := "低"
			if candidate.score >= 76 {
				confidence = "高"
			} else if candidate.score >= 59 {
				confidence = "中"
			}
			items = append(items, Evidence{ID: fmt.Sprintf("E%d", len(items)+1), SourceID: source.ID, SourceTitle: source.Title, URL: source.URL, Quote: candidate.quote, Score: candidate.score, Confidence: confidence, Tags: passageTags(candidate.quote), PublishedAt: source.PublishedAt, FetchedAt: source.FetchedAt})
			count++
			if count == 3 || len(items) == limit {
				break
			}
		}
		if len(items) == limit {
			break
		}
	}
	return items
}

func updateResearchStats(result *ResearchResult) {
	result.Stats.Read = 0
	result.Stats.HighConfidence = 0
	result.Stats.MediumConfidence = 0
	for _, source := range result.Sources {
		if source.Status == "已阅读" {
			result.Stats.Read++
		}
	}
	result.Stats.EvidenceCount = len(result.Evidence)
	for _, item := range result.Evidence {
		if item.Confidence == "高" {
			result.Stats.HighConfidence++
		}
		if item.Confidence == "中" {
			result.Stats.MediumConfidence++
		}
	}
}

func assessResearchCoverage(result ResearchResult, p ResearchPolicy, round int) ResearchCoverage {
	c := ResearchCoverage{Status: "insufficient", Round: round, MaxRounds: p.MaxRounds, SourcesRead: result.Stats.Read, EvidenceCount: len(result.Evidence), TargetSources: p.TargetSources, TargetDomains: p.TargetDomains, TargetEvidence: p.TargetEvidence, Gaps: []string{}}
	domains := map[string]bool{}
	sources := map[string]bool{}
	for _, item := range result.Evidence {
		domains[sourceDomain(item.URL)] = true
		sources[item.SourceID] = true
	}
	c.Domains = len(domains)
	c.EvidenceSources = len(sources)
	if len(sources) < p.TargetSources {
		c.Gaps = append(c.Gaps, fmt.Sprintf("产生相关证据的来源 %d/%d", len(sources), p.TargetSources))
	}
	if c.Domains < p.TargetDomains {
		c.Gaps = append(c.Gaps, fmt.Sprintf("证据来源域名 %d/%d（域名不等于独立原创来源）", c.Domains, p.TargetDomains))
	}
	if c.EvidenceCount < p.TargetEvidence {
		c.Gaps = append(c.Gaps, fmt.Sprintf("相关证据 %d/%d", c.EvidenceCount, p.TargetEvidence))
	}
	for _, dim := range marketDimensions {
		d := CoverageDimension{ID: dim.id, Name: dim.name, Status: "gap", EvidenceIDs: []string{}}
		ds, ss := map[string]bool{}, map[string]bool{}
		for _, item := range result.Evidence {
			if containsAny(strings.ToLower(item.Quote), dim.terms...) {
				ds[sourceDomain(item.URL)] = true
				ss[item.SourceID] = true
				d.EvidenceIDs = append(d.EvidenceIDs, item.ID)
			}
		}
		d.Domains = len(ds)
		d.Sources = len(ss)
		if len(ds) >= 2 {
			d.Status = "covered"
		} else {
			c.Gaps = append(c.Gaps, dim.name+"：尚缺两个不同域名的相关正文，需核验来源独立性")
		}
		c.Dimensions = append(c.Dimensions, d)
	}
	if len(c.Gaps) == 0 {
		c.Status = "sufficient"
	}
	return c
}

func coveragePrompt(c ResearchCoverage) string {
	data, _ := json.Marshal(c)
	return "\n研究覆盖检查（关键词与域名启发式，仅衡量覆盖，不是事实认证）：" + string(data) + "\n没有覆盖的维度必须在结论中保留为缺口；不得把数量达标称为完整市场全景，不得把同源转载视为独立交叉验证。\n"
}
