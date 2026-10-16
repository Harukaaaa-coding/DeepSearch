package main

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

func TestBuildReportDOCX(t *testing.T) {
	result := ResearchResult{
		Question:    "企业级 AI Agent 的创业机会在哪里",
		CompletedAt: time.Date(2026, 10, 4, 14, 30, 0, 0, time.UTC),
		Report:      Report{Title: "证据约束研究报告", Analysis: "## 执行摘要\n\n现有证据显示市场正在形成。 [E1]\n\n### 风险与反证\n\n- 需要继续核验市场口径。", Method: "两轮检索与交叉验证。", Limitations: []string{"样本有限。"}, Conflicts: []string{"数据口径存在差异。"}},
		Evidence:    []Evidence{{ID: "E1", SourceTitle: "官方来源", URL: "https://example.com", Quote: "市场正在形成。", Score: 86, Confidence: "高"}},
	}
	data, err := buildReportDOCX(result)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document string
	for _, file := range reader.File {
		if file.Name != "word/document.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		document = string(body)
	}
	if !strings.Contains(document, "深度研究报告") || !strings.Contains(document, "官方来源") || !strings.Contains(document, "证据清单") {
		t.Fatalf("DOCX document content is incomplete: %s", document)
	}
}

func TestBuildDeliveryPRDDOCX(t *testing.T) {
	store, err := NewKnowledgeStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.SeedAIAgentDemo(); err != nil {
		t.Fatal(err)
	}
	copilot, err := store.BuildDeliveryCopilot()
	if err != nil {
		t.Fatal(err)
	}
	data, err := buildDeliveryPRDDOCX(copilot)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document string
	for _, file := range reader.File {
		if file.Name != "word/document.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		document = string(body)
	}
	for _, expected := range []string{"产品需求文档", "低保真原型规格", "成功指标与衡量口径", "数据模型与数据字典", "接口契约与依赖", "工期与交付计划", "证据链、风险与待决项", "REQ-001"} {
		if !strings.Contains(document, expected) {
			t.Fatalf("delivery DOCX missing %q", expected)
		}
	}
}
