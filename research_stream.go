package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type researchRunner func(context.Context, string, func(ProgressEvent)) (ResearchResult, error)

func streamResearchWithRunner(w http.ResponseWriter, r *http.Request, run researchRunner, store *KnowledgeStore, question string) {
	if _, ok := w.(http.Flusher); !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	identity := newResearchRunIdentity()
	ctx, cancel := context.WithCancel(context.WithValue(r.Context(), researchRunIdentityKey{}, identity))
	defer cancel()
	record := ResearchRunDiagnostic{RunID: identity.ID, StartedAt: identity.StartedAt, Status: "running", LastStep: "scope"}
	if err := store.BeginResearchDiagnostic(identity.ID, identity.StartedAt); err != nil {
		record.WarningCount++
		record.Warnings = append(record.Warnings, "diagnostic_save_failed")
		log.Printf("research_diagnostic_save_failed run_id=%s phase=begin", identity.ID)
	}
	log.Printf("research_started run_id=%s", identity.ID)
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Research-Run-ID", identity.ID)
	var mu sync.Mutex
	var streamErr error
	var runErr error
	var result ResearchResult
	emit := func(kind string, payload any) bool {
		mu.Lock()
		defer mu.Unlock()
		if event, ok := payload.(ProgressEvent); ok {
			event.RunID = identity.ID
			payload = event
			record.EventCount++
			if event.Step != "" {
				record.LastStep = safeResearchStep(event.Step)
			}
			if event.Kind == "warning" {
				record.WarningCount++
				record.Warnings = append(record.Warnings, researchDiagnosticCode(errors.New(event.Message)))
			}
			if event.Kind == "search" {
				record.Stats.Queries++
			}
			if event.Coverage != nil {
				record.Stats.Discovered = max(record.Stats.Discovered, event.Discovered)
				record.Stats.Read = max(record.Stats.Read, event.Coverage.SourcesRead)
				record.Stats.EvidenceCount = max(record.Stats.EvidenceCount, event.Coverage.EvidenceCount)
			} else {
				record.Stats.Discovered += max(0, event.Discovered)
				record.Stats.Read += max(0, event.ReadCount)
				record.Stats.EvidenceCount = max(record.Stats.EvidenceCount, event.EvidenceCount)
			}
		}
		if streamErr != nil {
			return false
		}
		encoded, err := json.Marshal(payload)
		if err == nil {
			_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, encoded)
		}
		if err == nil {
			err = http.NewResponseController(w).Flush()
		}
		if err != nil {
			streamErr = err
			cancel()
			return false
		}
		return true
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			runErr = errors.New("research panic")
			record.Status, record.ErrorCode = "failed", "panic"
			emit("failure", map[string]string{"error": researchDiagnosticMessage("panic"), "runId": identity.ID, "status": "failed"})
		}
		mu.Lock()
		if streamErr != nil {
			record.Status, record.ErrorCode = "cancelled", "stream_disconnected"
		}
		if record.Status == "running" {
			record.Status, record.ErrorCode = "failed", researchDiagnosticCode(runErr)
			if errors.Is(runErr, context.Canceled) {
				record.Status = "cancelled"
			}
			if record.ErrorCode == "" {
				record.ErrorCode = "internal_error"
			}
		}
		record.EndedAt = time.Now().UTC()
		record.DurationMS = researchDurationMS(record.StartedAt, record.EndedAt)
		record.Stats.Queries = max(record.Stats.Queries, result.Stats.Queries)
		record.Stats.Discovered = max(record.Stats.Discovered, result.Stats.Discovered)
		record.Stats.Read = max(record.Stats.Read, result.Stats.Read)
		record.Stats.EvidenceCount = max(record.Stats.EvidenceCount, result.Stats.EvidenceCount)
		record.Stats.HighConfidence = result.Stats.HighConfidence
		record.Stats.MediumConfidence = result.Stats.MediumConfidence
		record.Stats.ConflictCount = result.Stats.ConflictCount
		finalRecord := record
		mu.Unlock()
		// Do not use the cancelled request context: diagnostics must survive disconnects.
		if err := store.FinishResearchDiagnostic(finalRecord); err != nil {
			log.Printf("research_diagnostic_save_failed run_id=%s phase=finish", identity.ID)
		}
		log.Printf("research_finished run_id=%s status=%s duration_ms=%d stage=%s events=%d warnings=%d error_code=%s", identity.ID, finalRecord.Status, finalRecord.DurationMS, finalRecord.LastStep, finalRecord.EventCount, finalRecord.WarningCount, finalRecord.ErrorCode)
	}()
	emit("progress", ProgressEvent{RunID: identity.ID, Kind: "started", Step: "scope", Message: "正在建立研究任务…", At: identity.StartedAt})
	if ctx.Err() != nil {
		runErr = ctx.Err()
	} else {
		result, runErr = run(ctx, question, func(event ProgressEvent) { emit("progress", event) })
	}
	if ctx.Err() != nil {
		runErr = ctx.Err()
	}
	result.RunID, result.StartedAt = identity.ID, identity.StartedAt
	if runErr != nil {
		record.ErrorCode = researchDiagnosticCode(runErr)
		record.Status = "failed"
		if errors.Is(runErr, context.Canceled) {
			record.Status = "cancelled"
		}
		emit("failure", map[string]string{"error": researchDiagnosticMessage(record.ErrorCode), "runId": identity.ID, "status": record.Status})
		return
	}
	if result.CompletedAt.IsZero() {
		result.CompletedAt = time.Now().UTC()
	}
	result.Status = "completed"
	if record.WarningCount > 0 {
		result.Status = "completed_with_warnings"
	}
	// Persist the complete research before notifying the browser. A disconnect
	// while sending the terminal event must not discard an already produced report.
	result.ArchiveStatus = "saved"
	if err := store.SaveResearchArchive(result); err != nil {
		result.ArchiveStatus = "failed"
		result.Status = "completed_with_warnings"
		emit("progress", ProgressEvent{Kind: "warning", Step: "synthesize", Message: "历史归档保存失败：本次报告仍可查看，请先下载保存，不要直接关闭页面。", At: time.Now().UTC()})
		log.Printf("research_archive_failed run_id=%s", identity.ID)
	}
	record.Status = result.Status
	// Do not store a successful operational trace if even the terminal frame
	// cannot be written. Once it is flushed, a browser may cancel its reader;
	// that must not retroactively turn this completed result into cancellation.
	if !emit("complete", result) {
		return
	}
	if err := store.RecordResearchRun(result); err != nil {
		record.WarningCount++
		record.Warnings = append(record.Warnings, "memory_save_failed")
		record.Status = "completed_with_warnings"
		log.Printf("research_memory_save_failed run_id=%s", identity.ID)
	}
}
