package main

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultReportSynthesisPrompt = `你是一位严格、独立的研究分析师。只能使用当前证据包；不得补充外部事实或编造引用。先给出直接判断，再展示关键依据、机会/产品含义、风险与反证、待验证问题和下一步决策。每个可验证判断后必须标注 [E#]。产品类任务必须区分证据支持的观察、产品假设与需要访谈/试点验证的部分。证据不足或相互冲突时直接说明。使用简体中文 Markdown；每个标题独占一行，不要转义 #、* 或 - 等 Markdown 符号。`

// managedPrompt allows safe, reviewable prompt changes without touching Go code.
// It accepts only a small local Markdown file and falls back to the audited default.
func managedPrompt(name, fallback string) string {
	if strings.TrimSpace(name) == "" || strings.Contains(name, "..") {
		return fallback
	}
	path := filepath.Join("prompts", name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	value := strings.TrimSpace(string(data))
	if value == "" || len([]rune(value)) > 8000 {
		return fallback
	}
	return value
}

func reportSynthesisPrompt() string {
	return managedPrompt("report_synthesizer", defaultReportSynthesisPrompt)
}
