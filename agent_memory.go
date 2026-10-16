package main

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultContextWindowTokens = 16000
	defaultCompressAtPercent   = 78
	defaultRecentEvents        = 12
	defaultSummaryTokens       = 1800
)

type AgentMemoryPolicy struct {
	ContextWindowTokens int `json:"contextWindowTokens"`
	CompressAtPercent   int `json:"compressAtPercent"`
	RetainRecentEvents  int `json:"retainRecentEvents"`
	SummaryTokenBudget  int `json:"summaryTokenBudget"`
}

type AgentMemoryEvent struct {
	ID            string `json:"id"`
	RunID         string `json:"runId"`
	Sequence      int    `json:"sequence"`
	Category      string `json:"category"`
	Tool          string `json:"tool"`
	InputSummary  string `json:"inputSummary"`
	OutputSummary string `json:"outputSummary"`
	Importance    int    `json:"importance"`
	CreatedAt     string `json:"createdAt"`
}

type AgentContextPack struct {
	Question              string             `json:"question"`
	Policy                AgentMemoryPolicy  `json:"policy"`
	EstimatedTokens       int                `json:"estimatedTokens"`
	RawEventTokens        int                `json:"rawEventTokens"`
	CompressionTriggered  bool               `json:"compressionTriggered"`
	PinnedFacts           []string           `json:"pinnedFacts"`
	RecentEvents          []AgentMemoryEvent `json:"recentEvents"`
	CompressedToolSummary string             `json:"compressedToolSummary"`
	SafetyNote            string             `json:"safetyNote"`
}

type AgentMemoryOverview struct {
	Runs      int              `json:"runs"`
	Events    int              `json:"events"`
	Summaries int              `json:"summaries"`
	Context   AgentContextPack `json:"context"`
}

func defaultAgentMemoryPolicy() AgentMemoryPolicy {
	return AgentMemoryPolicy{
		ContextWindowTokens: defaultContextWindowTokens,
		CompressAtPercent:   defaultCompressAtPercent,
		RetainRecentEvents:  defaultRecentEvents,
		SummaryTokenBudget:  defaultSummaryTokens,
	}
}

// loadAgentMemoryPolicy reads non-secret runtime tuning from agent.yaml.
// Missing or malformed values fall back to safe local defaults.
func loadAgentMemoryPolicy(path string) AgentMemoryPolicy {
	policy := defaultAgentMemoryPolicy()
	data, err := os.ReadFile(path)
	if err != nil {
		return policy
	}
	inMemory := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
		if indent == 0 {
			inMemory = strings.HasPrefix(line, "memory:")
			continue
		}
		if !inMemory {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(strings.Split(value, "#")[0]))
		if err != nil || parsed <= 0 {
			continue
		}
		switch strings.TrimSpace(key) {
		case "context_window_tokens":
			policy.ContextWindowTokens = clampInt(parsed, 2000, 128000)
		case "compress_at_percent":
			policy.CompressAtPercent = clampInt(parsed, 55, 90)
		case "retain_recent_events":
			policy.RetainRecentEvents = clampInt(parsed, 4, 40)
		case "summary_token_budget":
			policy.SummaryTokenBudget = clampInt(parsed, 300, 12000)
		}
	}
	return policy
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

// RecordResearchRun appends the agent's operational trace. It intentionally records
// tool inputs/results rather than an opaque chat transcript, so later compression is auditable.
func (s *KnowledgeStore) RecordResearchRun(result ResearchResult) error {
	if strings.TrimSpace(result.RunID) == "" {
		return fmt.Errorf("研究运行缺少 run ID")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	completed := result.CompletedAt
	if completed.IsZero() {
		completed = time.Now().UTC()
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	rollback := func(cause error) error {
		_ = tx.Rollback()
		return cause
	}
	resultSummary := shorten(result.Report.Title+"；"+result.Report.Method, 900)
	status := "completed"
	if result.Status == "completed_with_warnings" {
		status = result.Status
	}
	if _, err := tx.Exec(`INSERT INTO agent_runs (id,question,status,result_summary,started_at,completed_at,created_at)
		VALUES (?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET question=excluded.question,status=excluded.status,result_summary=excluded.result_summary,completed_at=excluded.completed_at`,
		result.RunID, result.Question, status, resultSummary, result.StartedAt.Format(time.RFC3339Nano), completed.Format(time.RFC3339Nano), now); err != nil {
		return rollback(fmt.Errorf("写入 Agent run 失败：%w", err))
	}
	if _, err := tx.Exec(`DELETE FROM agent_events WHERE run_id=?`, result.RunID); err != nil {
		return rollback(fmt.Errorf("刷新 Agent 事件失败：%w", err))
	}

	plan := make([]string, 0, len(result.Plan))
	for _, step := range result.Plan {
		plan = append(plan, step.Title+"（"+step.Status+"）")
	}
	queries, readTitles, evidenceIDs := []string{}, []string{}, []string{}
	for _, source := range result.Sources {
		if source.Query != "" {
			queries = append(queries, source.Query)
		}
		if source.Status == "已阅读" {
			readTitles = append(readTitles, source.Title)
		}
	}
	for _, evidence := range result.Evidence {
		evidenceIDs = append(evidenceIDs, evidence.ID+"·"+evidence.SourceTitle)
	}
	events := []AgentMemoryEvent{
		{Category: "plan", Tool: "research_planner", InputSummary: "问题：" + shorten(result.Question, 220), OutputSummary: "计划：" + strings.Join(plan, " → "), Importance: 95},
		{Category: "search", Tool: "web_search", InputSummary: "查询：" + strings.Join(uniqueStrings(queries), " | "), OutputSummary: fmt.Sprintf("发现 %d 个候选来源，去重后进入阅读队列。", result.Stats.Discovered), Importance: 75},
		{Category: "read", Tool: "page_reader", InputSummary: fmt.Sprintf("读取上限 %d 页。", maxPagesPerRun), OutputSummary: fmt.Sprintf("已阅读 %d 页：%s", result.Stats.Read, strings.Join(uniqueStrings(readTitles), " | ")), Importance: 70},
		{Category: "evidence", Tool: "evidence_extractor", InputSummary: "按问题相关性、来源类型、时间和正文片段提取。", OutputSummary: fmt.Sprintf("提取 %d 条证据：%s", result.Stats.EvidenceCount, strings.Join(evidenceIDs, " | ")), Importance: 92},
		{Category: "verify", Tool: "conflict_checker", InputSummary: "比较同主题的方向性表述。", OutputSummary: fmt.Sprintf("发现 %d 个潜在冲突；冲突只提示继续核验，不替代事实判断。", result.Stats.ConflictCount), Importance: 84},
		{Category: "synthesis", Tool: "report_synthesizer", InputSummary: "输入仅包含当前研究的证据包。", OutputSummary: "输出：" + shorten(result.Report.Title+"；"+result.Report.ModelUsed, 400), Importance: 88},
	}
	for index, event := range events {
		event.ID = "agent-event-" + shortID(result.RunID+":"+event.Tool)
		event.RunID, event.Sequence, event.CreatedAt = result.RunID, index+1, completed.Format(time.RFC3339Nano)
		if _, err := tx.Exec(`INSERT INTO agent_events (id,run_id,sequence,category,tool,input_summary,output_summary,importance,created_at) VALUES (?,?,?,?,?,?,?,?,?)`,
			event.ID, event.RunID, event.Sequence, event.Category, event.Tool, shorten(event.InputSummary, 1400), shorten(event.OutputSummary, 1600), event.Importance, event.CreatedAt); err != nil {
			return rollback(fmt.Errorf("写入 Agent 工具轨迹失败：%w", err))
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交 Agent 运行记录失败：%w", err)
	}
	_, err = s.BuildAgentContext(result.Question)
	return err
}

func (s *KnowledgeStore) BuildAgentContext(question string) (AgentContextPack, error) {
	return s.buildAgentContext(question, loadAgentMemoryPolicy("agent.yaml"))
}

func (s *KnowledgeStore) buildAgentContext(question string, policy AgentMemoryPolicy) (AgentContextPack, error) {
	pack := AgentContextPack{
		Question:    strings.TrimSpace(question),
		Policy:      policy,
		PinnedFacts: s.agentPinnedFacts(),
		SafetyNote:  "压缩摘要只保存 Agent 使用过的工具与结果概览；它不是原始证据。涉及事实判断时，必须回到 Evidence ID、来源和原文。",
	}
	rows, err := s.db.Query(`SELECT id,run_id,sequence,category,tool,input_summary,output_summary,importance,created_at FROM agent_events ORDER BY created_at DESC,sequence DESC LIMIT 160`)
	if err != nil {
		return pack, fmt.Errorf("读取 Agent 记忆失败：%w", err)
	}
	defer rows.Close()
	events := make([]AgentMemoryEvent, 0, 80)
	for rows.Next() {
		var event AgentMemoryEvent
		if err := rows.Scan(&event.ID, &event.RunID, &event.Sequence, &event.Category, &event.Tool, &event.InputSummary, &event.OutputSummary, &event.Importance, &event.CreatedAt); err != nil {
			return pack, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return pack, err
	}

	terms := knowledgeTerms(pack.Question)
	sort.SliceStable(events, func(i, j int) bool {
		left := events[i].Importance*10 + keywordScore(terms, events[i].Tool+" "+events[i].InputSummary, events[i].OutputSummary)
		right := events[j].Importance*10 + keywordScore(terms, events[j].Tool+" "+events[j].InputSummary, events[j].OutputSummary)
		if left == right {
			return events[i].CreatedAt > events[j].CreatedAt
		}
		return left > right
	})

	pinnedTokens := estimateContextTokens(strings.Join(pack.PinnedFacts, "\n"))
	rawLimit := policy.ContextWindowTokens*policy.CompressAtPercent/100 - pinnedTokens
	if rawLimit < 500 {
		rawLimit = 500
	}
	remaining := make([]AgentMemoryEvent, 0, len(events))
	for _, event := range events {
		eventTokens := estimateContextTokens(event.Tool + event.InputSummary + event.OutputSummary)
		mustKeepRecent := len(pack.RecentEvents) < policy.RetainRecentEvents
		if mustKeepRecent || pack.RawEventTokens+eventTokens <= rawLimit {
			pack.RecentEvents = append(pack.RecentEvents, event)
			pack.RawEventTokens += eventTokens
		} else {
			remaining = append(remaining, event)
		}
	}
	if len(remaining) > 0 {
		pack.CompressionTriggered = true
		pack.CompressedToolSummary = compactAgentEvents(remaining, policy.SummaryTokenBudget)
	}
	pack.EstimatedTokens = pinnedTokens + pack.RawEventTokens + estimateContextTokens(pack.CompressedToolSummary) + estimateContextTokens(pack.SafetyNote)
	if pack.CompressionTriggered {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		_, _ = s.db.Exec(`INSERT INTO agent_memory_summaries (scope,summary,covered_event_count,updated_at) VALUES (?,?,?,?)
			ON CONFLICT(scope) DO UPDATE SET summary=excluded.summary,covered_event_count=excluded.covered_event_count,updated_at=excluded.updated_at`,
			"rolling_tool_context", pack.CompressedToolSummary, len(remaining), now)
	}
	return pack, nil
}

func (s *KnowledgeStore) agentPinnedFacts() []string {
	facts := []string{"证据规则：压缩摘要不能替代来源、原文与证据 ID。"}
	state, err := s.GetDeliveryWorkflowState()
	if err == nil && state.Persisted {
		facts = append(facts, fmt.Sprintf("当前产品交付：PRD v%s，状态 %s，%d 条需求、%d 个证据链接、Readiness %d/100。", state.Version, state.Status, state.Requirements, state.EvidenceLinks, state.ReadinessScore))
	}
	return facts
}

func compactAgentEvents(events []AgentMemoryEvent, maxTokens int) string {
	if len(events) == 0 {
		return ""
	}
	type aggregate struct {
		count int
		last  AgentMemoryEvent
	}
	groups := map[string]aggregate{}
	for _, event := range events {
		item := groups[event.Tool]
		item.count++
		if item.last.CreatedAt == "" || event.CreatedAt > item.last.CreatedAt {
			item.last = event
		}
		groups[event.Tool] = item
	}
	tools := make([]string, 0, len(groups))
	for tool := range groups {
		tools = append(tools, tool)
	}
	sort.Strings(tools)
	parts := []string{"已压缩的较早 Agent 工具轨迹（仅作操作上下文，不是事实证据）："}
	for _, tool := range tools {
		item := groups[tool]
		parts = append(parts, fmt.Sprintf("- %s：%d 次；最近结果：%s", tool, item.count, shorten(item.last.OutputSummary, 240)))
	}
	text := strings.Join(parts, "\n")
	maxRunes := max(200, maxTokens*2)
	return shorten(text, maxRunes)
}

func estimateContextTokens(value string) int {
	runes := len([]rune(strings.TrimSpace(value)))
	if runes == 0 {
		return 0
	}
	return int(math.Ceil(float64(runes) / 2.0))
}

func (s *KnowledgeStore) AgentMemoryOverview(question string) (AgentMemoryOverview, error) {
	context, err := s.BuildAgentContext(question)
	if err != nil {
		return AgentMemoryOverview{}, err
	}
	overview := AgentMemoryOverview{Context: context}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM agent_runs`).Scan(&overview.Runs); err != nil {
		return overview, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM agent_events`).Scan(&overview.Events); err != nil {
		return overview, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM agent_memory_summaries`).Scan(&overview.Summaries); err != nil && err != sql.ErrNoRows {
		return overview, err
	}
	return overview, nil
}
