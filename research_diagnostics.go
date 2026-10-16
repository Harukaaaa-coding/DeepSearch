package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type ResearchRunDiagnostic struct {
	RunID        string    `json:"runId"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"startedAt"`
	EndedAt      time.Time `json:"endedAt,omitempty"`
	DurationMS   int64     `json:"durationMs"`
	LastStep     string    `json:"lastStep"`
	EventCount   int       `json:"eventCount"`
	WarningCount int       `json:"warningCount"`
	ErrorCode    string    `json:"errorCode,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	Warnings     []string  `json:"warnings,omitempty"`
	Stats        Stats     `json:"stats"`
}

type researchRunIdentity struct {
	ID        string
	StartedAt time.Time
}
type researchRunIdentityKey struct{}

func newResearchRunIdentity() researchRunIdentity {
	started := time.Now().UTC()
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return researchRunIdentity{ID: "run-" + shortID(started.String()), StartedAt: started}
	}
	return researchRunIdentity{ID: "run-" + hex.EncodeToString(random[:]), StartedAt: started}
}

func researchRunIdentityFromContext(ctx context.Context) researchRunIdentity {
	if identity, ok := ctx.Value(researchRunIdentityKey{}).(researchRunIdentity); ok {
		return identity
	}
	return newResearchRunIdentity()
}

var researchDiagnosticID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,80}$`)
var researchHTTPCode = regexp.MustCompile(`(?i)HTTP\s+([1-5][0-9][0-9])`)

func (s *KnowledgeStore) BeginResearchDiagnostic(runID string, started time.Time) error {
	if !researchDiagnosticID.MatchString(runID) || started.IsZero() {
		return errors.New("研究诊断缺少有效任务标识或开始时间")
	}
	_, err := s.db.Exec(`INSERT INTO research_run_diagnostics (run_id,status,started_at,last_step) VALUES (?,?,?,?)`, runID, "running", started.UTC().Format(time.RFC3339Nano), "scope")
	return err
}

func (s *KnowledgeStore) FinishResearchDiagnostic(record ResearchRunDiagnostic) error {
	if !researchDiagnosticID.MatchString(record.RunID) || record.StartedAt.IsZero() {
		return errors.New("研究诊断缺少有效任务标识或开始时间")
	}
	switch record.Status {
	case "completed", "completed_with_warnings", "failed", "cancelled":
	default:
		return errors.New("无效的研究终态")
	}
	if record.EndedAt.IsZero() {
		record.EndedAt = time.Now().UTC()
	}
	record.DurationMS = researchDurationMS(record.StartedAt, record.EndedAt)
	// Diagnostics deliberately contain no question, document text or raw provider
	// error. Only allowlisted categories are persisted and exposed to the UI.
	record.LastStep = safeResearchStep(record.LastStep)
	record.ErrorMessage = researchDiagnosticMessage(record.ErrorCode)
	if record.ErrorMessage == "" {
		record.ErrorCode = ""
	}
	warnings := make([]string, 0, len(record.Warnings))
	for _, code := range record.Warnings {
		if researchDiagnosticMessage(code) != "" {
			warnings = append(warnings, code)
		}
	}
	statsJSON, _ := json.Marshal(record.Stats)
	warningsJSON, _ := json.Marshal(uniqueStrings(warnings))
	_, err := s.db.Exec(`INSERT INTO research_run_diagnostics
		(run_id,status,started_at,ended_at,duration_ms,last_step,event_count,warning_count,error_code,error_message,stats_json,warnings_json)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(run_id) DO UPDATE SET
		status=excluded.status,ended_at=excluded.ended_at,duration_ms=excluded.duration_ms,last_step=excluded.last_step,
		event_count=excluded.event_count,warning_count=excluded.warning_count,error_code=excluded.error_code,
		error_message=excluded.error_message,stats_json=excluded.stats_json,warnings_json=excluded.warnings_json
		WHERE research_run_diagnostics.status='running'`,
		record.RunID, record.Status, record.StartedAt.UTC().Format(time.RFC3339Nano), record.EndedAt.UTC().Format(time.RFC3339Nano), record.DurationMS,
		record.LastStep, max(0, record.EventCount), max(0, record.WarningCount), record.ErrorCode, record.ErrorMessage, string(statsJSON), string(warningsJSON))
	return err
}

func (s *KnowledgeStore) ResearchDiagnostic(runID string) (ResearchRunDiagnostic, error) {
	if !researchDiagnosticID.MatchString(runID) {
		return ResearchRunDiagnostic{}, sql.ErrNoRows
	}
	var record ResearchRunDiagnostic
	var started, ended, statsJSON, warningsJSON string
	err := s.db.QueryRow(`SELECT run_id,status,started_at,ended_at,duration_ms,last_step,event_count,warning_count,error_code,error_message,stats_json,warnings_json FROM research_run_diagnostics WHERE run_id=?`, runID).Scan(
		&record.RunID, &record.Status, &started, &ended, &record.DurationMS, &record.LastStep, &record.EventCount, &record.WarningCount, &record.ErrorCode, &record.ErrorMessage, &statsJSON, &warningsJSON)
	if err != nil {
		return record, err
	}
	record.StartedAt, err = time.Parse(time.RFC3339Nano, started)
	if err != nil {
		return record, err
	}
	if ended != "" {
		record.EndedAt, err = time.Parse(time.RFC3339Nano, ended)
		if err != nil {
			return record, err
		}
	}
	if err := json.Unmarshal([]byte(statsJSON), &record.Stats); err != nil {
		return record, err
	}
	if err := json.Unmarshal([]byte(warningsJSON), &record.Warnings); err != nil {
		return record, err
	}
	if record.Status == "running" {
		record.DurationMS = researchDurationMS(record.StartedAt, time.Now())
	}
	return record, nil
}

func safeResearchStep(step string) string {
	switch step {
	case "scope", "discover", "verify", "synthesize", "read", "extract", "plan", "search", "report", "competition", "scenarios", "blueprint":
		return step
	}
	return "unknown"
}

func researchDurationMS(started, ended time.Time) int64 {
	duration := ended.Sub(started).Milliseconds()
	if duration < 0 {
		return 0
	}
	return duration
}

func researchDiagnosticCode(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, errIncompleteModelOutput) {
		return "model_output_incomplete"
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline_exceeded"
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return "unexpected_eof"
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "历史归档保存失败") {
		return "history_save_failed"
	}
	if strings.Contains(message, "截断") || strings.Contains(message, "不完整") || strings.Contains(message, "长度上限") || strings.Contains(message, "finish_reason") {
		return "model_output_incomplete"
	}
	if match := researchHTTPCode.FindStringSubmatch(message); len(match) > 1 {
		return "upstream_http_" + match[1]
	}
	if strings.Contains(message, "deadline exceeded") || strings.Contains(message, "timeout") {
		return "deadline_exceeded"
	}
	if strings.Contains(message, "eof") {
		return "unexpected_eof"
	}
	return "internal_error"
}

func researchDiagnosticMessage(code string) string {
	if regexp.MustCompile(`^upstream_http_[1-5][0-9][0-9]$`).MatchString(code) {
		return "上游服务返回 HTTP " + strings.TrimPrefix(code, "upstream_http_")
	}
	return map[string]string{
		"history_save_failed": "完整研究报告未能保存到历史归档，请手动下载保留。",
		"cancelled":           "研究已取消或客户端连接已关闭。", "stream_disconnected": "研究数据流连接中断，客户端未确认接收完整结果。",
		"deadline_exceeded": "研究或上游请求超过等待时限。", "unexpected_eof": "上游连接提前结束，未收到完整响应。",
		"model_output_incomplete": "模型输出被截断或未完整结束，未作为成功报告。", "internal_error": "研究处理失败；请根据任务编号与最后阶段定位。",
		"panic": "研究处理发生未预期异常，已停止任务。", "memory_save_failed": "研究结果的记忆记录未保存。", "diagnostic_save_failed": "研究诊断记录未能保存到数据库。",
	}[code]
}

func registerResearchDiagnosticRoutes(mux *http.ServeMux, store *KnowledgeStore) {
	mux.HandleFunc("/api/research/runs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		record, err := store.ResearchDiagnostic(strings.TrimPrefix(r.URL.Path, "/api/research/runs/"))
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "未找到该研究任务的诊断记录"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "暂时无法读取研究诊断记录"})
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, record)
	})
}
