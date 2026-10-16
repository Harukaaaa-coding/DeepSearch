package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ResearchHistoryItem struct {
	RunID       string `json:"runId"`
	Question    string `json:"question"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	StartedAt   string `json:"startedAt"`
	CompletedAt string `json:"completedAt"`
	HasSnapshot bool   `json:"hasSnapshot"`
	HasDelivery bool   `json:"hasDelivery"`
	Summary     string `json:"summary"`
}

type ResearchHistoryPage struct {
	Items  []ResearchHistoryItem `json:"items"`
	Total  int                   `json:"total"`
	Limit  int                   `json:"limit"`
	Offset int                   `json:"offset"`
}

type ResearchHistoryDetail struct {
	Item     ResearchHistoryItem `json:"item"`
	Result   *ResearchResult     `json:"result"`
	Delivery *DeliveryCopilot    `json:"delivery"`
	Notice   string              `json:"notice,omitempty"`
}

var errResearchSavedDeliveryUnreadable = errors.New("已保存的 PRD 版本无法读取")

// SaveResearchArchive stores the complete client-visible result independently of
// the token-budgeted agent memory. Reopening it never requires a model request.
func (s *KnowledgeStore) SaveResearchArchive(result ResearchResult) error {
	if !researchDiagnosticID.MatchString(result.RunID) || strings.TrimSpace(result.Question) == "" || strings.TrimSpace(result.Report.Analysis) == "" {
		return errors.New("研究归档需要有效任务编号、问题和完整报告正文")
	}
	status := result.Status
	if status == "" {
		status = "completed"
	}
	if status != "completed" && status != "completed_with_warnings" {
		return errors.New("未完成的研究不能作为完整报告归档")
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("序列化研究归档失败：%w", err)
	}
	title := strings.TrimSpace(result.Report.Title)
	if title == "" {
		title = result.Question
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	updated, err := s.db.Exec(`INSERT INTO research_archives (run_id,question,title,status,started_at,completed_at,result_json,delivery_json,updated_at)
		VALUES (?,?,?,?,?,?,?,'',?) ON CONFLICT(run_id) DO UPDATE SET
		title=excluded.title,status=excluded.status,started_at=excluded.started_at,completed_at=excluded.completed_at,result_json=excluded.result_json,updated_at=excluded.updated_at
		WHERE research_archives.question=excluded.question`, result.RunID, result.Question, title, status,
		result.StartedAt.Format(time.RFC3339Nano), result.CompletedAt.Format(time.RFC3339Nano), string(payload), now)
	if err != nil {
		return fmt.Errorf("保存研究归档失败：%w", err)
	}
	if affected, err := updated.RowsAffected(); err != nil {
		return err
	} else if affected != 1 {
		return errors.New("研究任务编号已属于其他问题，不能覆盖")
	}
	return nil
}

func researchHistoryPRDID(runID, question string) string {
	return "RPRD-" + strings.ToUpper(shortID(runID+strings.TrimSpace(question)))
}

func researchHistoryDeliveryMatches(copilot DeliveryCopilot, runID, question string) bool {
	return copilot.PRD.Info.PRDID == researchHistoryPRDID(runID, question) &&
		(copilot.ResearchRunID == "" || copilot.ResearchRunID == runID)
}

// SaveResearchDelivery attaches only the PRD belonging to the archived run. It
// does not overwrite manually saved PRD versions or infer associations by topic.
func (s *KnowledgeStore) SaveResearchDelivery(runID string, copilot DeliveryCopilot) error {
	if !researchDiagnosticID.MatchString(runID) {
		return sql.ErrNoRows
	}
	var question string
	if err := s.db.QueryRow(`SELECT question FROM research_archives WHERE run_id=?`, runID).Scan(&question); err != nil {
		return err
	}
	if !researchHistoryDeliveryMatches(copilot, runID, question) {
		return errors.New("PRD 与研究任务不匹配，不能写入该研究归档")
	}
	copilot.ResearchRunID = runID
	payload, err := json.Marshal(copilot)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE research_archives SET delivery_json=?,updated_at=? WHERE run_id=?`, string(payload), time.Now().UTC().Format(time.RFC3339Nano), runID)
	return err
}

const researchHistoryUnion = `WITH history AS (
	SELECT run_id,question,title,status,started_at,completed_at,1 AS has_snapshot,
		CASE WHEN delivery_json<>'' THEN 1 ELSE 0 END AS has_delivery,
		COALESCE(substr(json_extract(result_json,'$.report.analysis'),1,240),'') AS summary
	FROM research_archives
	UNION ALL
	SELECT id AS run_id,question,question AS title,status,started_at,completed_at,0 AS has_snapshot,0 AS has_delivery,result_summary AS summary
	FROM agent_runs r WHERE NOT EXISTS (SELECT 1 FROM research_archives a WHERE a.run_id=r.id)
) `

// Search uses literal, parameterized substring matching; % and _ in a question
// are not wildcard instructions. Paging never reads report bodies from the web.
func (s *KnowledgeStore) ListResearchHistory(query string, limit, offset int) (ResearchHistoryPage, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	page := ResearchHistoryPage{Items: []ResearchHistoryItem{}, Limit: limit, Offset: offset}
	pattern := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.TrimSpace(query)) + "%"
	const where = ` WHERE (question LIKE ? ESCAPE '\' OR title LIKE ? ESCAPE '\' OR summary LIKE ? ESCAPE '\') `
	if err := s.db.QueryRow(researchHistoryUnion+`SELECT COUNT(*) FROM history`+where, pattern, pattern, pattern).Scan(&page.Total); err != nil {
		return page, err
	}
	rows, err := s.db.Query(researchHistoryUnion+`SELECT run_id,question,title,status,started_at,completed_at,has_snapshot,has_delivery,summary FROM history`+where+`ORDER BY completed_at DESC,run_id DESC LIMIT ? OFFSET ?`, pattern, pattern, pattern, limit, offset)
	if err != nil {
		return page, err
	}
	for rows.Next() {
		var item ResearchHistoryItem
		if err := rows.Scan(&item.RunID, &item.Question, &item.Title, &item.Status, &item.StartedAt, &item.CompletedAt, &item.HasSnapshot, &item.HasDelivery, &item.Summary); err != nil {
			_ = rows.Close()
			return page, err
		}
		item.Summary = shorten(strings.Join(strings.Fields(item.Summary), " "), 240)
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return page, err
	}
	if err := rows.Close(); err != nil {
		return page, err
	}
	// Check exact saved initiatives after closing rows, also supporting stores with
	// a single DB connection and legacy runs that predate complete snapshots.
	for i := range page.Items {
		copilot, err := s.savedResearchDelivery(page.Items[i].RunID, page.Items[i].Question)
		if errors.Is(err, errResearchSavedDeliveryUnreadable) {
			// One damaged PRD must not hide the report or the entire history list.
			page.Items[i].HasDelivery = false
			continue
		}
		if err != nil {
			return page, err
		}
		if copilot != nil {
			page.Items[i].HasDelivery = true
		}
	}
	return page, nil
}

// savedResearchDelivery reads the latest saved version of the one exact PRD.
// Similar titles, questions and the globally latest draft are never substitutes.
func (s *KnowledgeStore) savedResearchDelivery(runID, question string) (*DeliveryCopilot, error) {
	prdID := researchHistoryPRDID(runID, question)
	key := deliveryInitiativeKey + "-" + strings.ToUpper(shortID(prdID))
	var raw string
	err := s.db.QueryRow(`SELECT v.snapshot_json FROM prd_versions v JOIN product_initiatives i ON i.id=v.initiative_id
		WHERE i.initiative_key=? ORDER BY v.updated_at DESC,v.id DESC LIMIT 1`, key).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var copilot DeliveryCopilot
	if err := json.Unmarshal([]byte(raw), &copilot); err != nil {
		return nil, errResearchSavedDeliveryUnreadable
	}
	if !researchHistoryDeliveryMatches(copilot, runID, question) {
		return nil, errResearchSavedDeliveryUnreadable
	}
	copilot.ResearchRunID = runID
	return &copilot, nil
}

func (s *KnowledgeStore) GetResearchArchive(runID string) (ResearchHistoryDetail, error) {
	detail := ResearchHistoryDetail{}
	if !researchDiagnosticID.MatchString(runID) {
		return detail, sql.ErrNoRows
	}
	var resultJSON, deliveryJSON string
	err := s.db.QueryRow(`SELECT run_id,question,title,status,started_at,completed_at,result_json,delivery_json FROM research_archives WHERE run_id=?`, runID).
		Scan(&detail.Item.RunID, &detail.Item.Question, &detail.Item.Title, &detail.Item.Status, &detail.Item.StartedAt, &detail.Item.CompletedAt, &resultJSON, &deliveryJSON)
	if errors.Is(err, sql.ErrNoRows) {
		err = s.db.QueryRow(`SELECT id,question,question,status,started_at,completed_at,result_summary FROM agent_runs WHERE id=?`, runID).
			Scan(&detail.Item.RunID, &detail.Item.Question, &detail.Item.Title, &detail.Item.Status, &detail.Item.StartedAt, &detail.Item.CompletedAt, &detail.Item.Summary)
		if err != nil {
			return detail, err
		}
		detail.Notice = "该记录来自升级前的研究摘要，未保存完整报告，不能还原原始正文。可查看已有摘要及对应的已保存 PRD；查看不会重新搜索或消耗模型 token。"
	} else if err != nil {
		return detail, err
	} else {
		var result ResearchResult
		if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
			return detail, errors.New("研究归档内容无法读取")
		}
		if result.RunID != runID || result.Question != detail.Item.Question || strings.TrimSpace(result.Report.Analysis) == "" {
			return detail, errors.New("研究归档内容与任务不一致")
		}
		detail.Result, detail.Item.HasSnapshot = &result, true
		detail.Item.Summary = shorten(strings.Join(strings.Fields(result.Report.Analysis), " "), 240)
	}
	// Human edits saved as a PRD version take precedence over initial generation.
	detail.Delivery, err = s.savedResearchDelivery(runID, detail.Item.Question)
	if errors.Is(err, errResearchSavedDeliveryUnreadable) {
		// Never replace unreadable human edits with an older generated draft.
		detail.Notice += "对应已保存的 PRD 无法读取；已保留可用的研究报告或摘要，未自动覆盖或重新生成。"
		return detail, nil
	}
	if err != nil {
		return detail, err
	}
	if detail.Delivery == nil && deliveryJSON != "" {
		var copilot DeliveryCopilot
		if err := json.Unmarshal([]byte(deliveryJSON), &copilot); err != nil {
			detail.Notice += "研究报告已恢复，但归档中的 PRD 无法读取；未自动覆盖或重新生成。"
			return detail, nil
		}
		if researchHistoryDeliveryMatches(copilot, runID, detail.Item.Question) {
			copilot.ResearchRunID = runID
			detail.Delivery = &copilot
		}
	}
	detail.Item.HasDelivery = detail.Delivery != nil
	return detail, nil
}

func registerResearchHistoryRoutes(mux *http.ServeMux, store *KnowledgeStore) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "历史记录仅支持读取"})
			return
		}
		if r.URL.Path == "/api/research/history" {
			limit, offset := 20, 0
			for name, target := range map[string]*int{"limit": &limit, "offset": &offset} {
				if value := r.URL.Query().Get(name); value != "" {
					parsed, err := strconv.Atoi(value)
					if err != nil || parsed < 0 || (name == "limit" && parsed == 0) {
						writeJSON(w, http.StatusBadRequest, map[string]string{"error": "历史记录分页参数无效"})
						return
					}
					*target = parsed
				}
			}
			page, err := store.ListResearchHistory(r.URL.Query().Get("q"), limit, offset)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "暂时无法读取研究历史"})
				return
			}
			writeJSON(w, http.StatusOK, page)
			return
		}
		runID := strings.TrimPrefix(r.URL.Path, "/api/research/history/")
		detail, err := store.GetResearchArchive(runID)
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "未找到该研究历史记录"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "暂时无法读取该研究归档"})
			return
		}
		writeJSON(w, http.StatusOK, detail)
	}
	mux.HandleFunc("/api/research/history", handler)
	mux.HandleFunc("/api/research/history/", handler)
}
