package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Opt-in only: this calls public search and the configured report model, never image APIs.
func TestLiveAdaptiveResearch(t *testing.T) {
	if os.Getenv("DEEPSEARCH_LIVE_TEST") != "1" {
		t.Skip("set DEEPSEARCH_LIVE_TEST=1 to run paid model/public search integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	question := "为中国自媒体内容团队设计 AI Agent 工作台：比较扣子、Dify 等替代方案的用户场景、功能、公开定价与限制，分析市场机会和反证，给出有证据的 MVP、验收指标、数据结构、测试和工期 PRD 草案。"
	result, err := NewResearchEngine().Run(ctx, question, func(event ProgressEvent) {
		if event.Kind == "coverage" || event.Kind == "warning" || event.Kind == "plan" {
			fmt.Printf("%s %s\n", time.Now().Format("15:04:05"), event.Message)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("data", "test-runs")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "adaptive-"+time.Now().Format("20060102-150405")+".json")
	data, _ := json.MarshalIndent(result, "", "  ")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("saved=%s queries=%d discovered=%d read=%d evidence=%d effectiveSources=%d domains=%d coverage=%s stop=%s", path, result.Stats.Queries, result.Stats.Discovered, result.Stats.Read, len(result.Evidence), result.Coverage.EvidenceSources, result.Coverage.Domains, result.Coverage.Status, result.Coverage.StopReason)
	policy := loadResearchPolicy()
	if result.Stats.Queries > policy.MaxQueries || len(result.Sources) > policy.MaxSources {
		t.Error("research exceeded configured budget")
	}
	if len(result.Evidence) == 0 {
		t.Error("no relevant evidence from live search")
	}
	if result.Report.ModelUsed == "" {
		t.Error("model report was not generated")
	}
	if result.ProductBlueprint == nil {
		t.Errorf("PRD blueprint unavailable: %s", result.BlueprintStatus)
	}
	if result.Coverage.Status == "insufficient" && len(result.Coverage.Gaps) == 0 {
		t.Error("insufficient coverage lost its gaps")
	}
}

func TestLiveSearchDiagnostics(t *testing.T) {
	if os.Getenv("DEEPSEARCH_LIVE_TEST") != "1" {
		t.Skip("explicit live test only")
	}
	engine := NewResearchEngine()
	for _, q := range []string{"Dify pricing", "扣子 官方 产品文档 功能", "中国 自媒体 AI Agent 工作台 竞品对比 定价 限制 官方 定价 套餐 计费"} {
		hits, err := engine.search(context.Background(), q)
		t.Logf("query=%s hits=%d err=%v", q, len(hits), err)
		for _, h := range hits {
			t.Logf("  %s %s", h.Title, h.URL)
		}
	}
}

func TestLiveEvidenceLoop(t *testing.T) {
	if os.Getenv("DEEPSEARCH_LIVE_TEST") != "1" {
		t.Skip("explicit live test only")
	}
	engine := NewResearchEngine()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	result := ResearchResult{Question: "中国自媒体 AI Agent 工作台"}
	runEvidenceLoop(ctx, &result, loadResearchPolicy(), ResearchSearchPlan{Topic: result.Question, Candidates: []string{"Dify", "扣子", "FastGPT", "百度千帆"}}, researchTools{engine.collectHits, engine.searchPapers, engine.readSources}, func(e ProgressEvent) {
		if e.Kind == "coverage" || e.Kind == "search" || e.Kind == "warning" {
			fmt.Println(e.Message)
		}
	})
	for _, s := range result.Sources {
		t.Logf("%s %s %s %s", s.Status, s.Error, s.Title, s.URL)
	}
	t.Logf("queries=%d discovered=%d attempted=%d read=%d evidence=%d", result.Stats.Queries, result.Stats.Discovered, len(result.Sources), result.Stats.Read, len(result.Evidence))
}
