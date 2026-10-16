package main

import (
	"fmt"
	"sort"
	"strings"
)

type HistoryRecord struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Metadata  string `json:"metadata"`
	UpdatedAt string `json:"updatedAt"`
	Score     int    `json:"score"`
}

type HistorySearchResult struct {
	Records []HistoryRecord `json:"records"`
	Total   int             `json:"total"`
	Query   string          `json:"query"`
	Type    string          `json:"type"`
}

// SearchHistory searches every structured internal-knowledge table, not only document chunks.
func (s *KnowledgeStore) SearchHistory(query, recordType string, limit int) (HistorySearchResult, error) {
	query = strings.TrimSpace(query)
	recordType = strings.TrimSpace(recordType)
	if recordType == "" {
		recordType = "all"
	}
	if limit <= 0 || limit > 80 {
		limit = 30
	}
	terms := knowledgeTerms(query)
	records := make([]HistoryRecord, 0, limit)
	add := func(record HistoryRecord) {
		if recordType != "all" && record.Type != recordType {
			return
		}
		record.Score = historyScore(terms, record.Title, record.Summary, record.Metadata)
		if query != "" && record.Score == 0 {
			return
		}
		records = append(records, record)
	}
	if err := s.addDocumentHistory(add); err != nil {
		return HistorySearchResult{}, err
	}
	if err := s.addTicketHistory(add); err != nil {
		return HistorySearchResult{}, err
	}
	if err := s.addInterviewHistory(add); err != nil {
		return HistorySearchResult{}, err
	}
	if err := s.addInsightHistory(add); err != nil {
		return HistorySearchResult{}, err
	}
	if err := s.addRequirementHistory(add); err != nil {
		return HistorySearchResult{}, err
	}
	if err := s.addResearchNoteHistory(add); err != nil {
		return HistorySearchResult{}, err
	}
	if err := s.addDatasetHistory(add); err != nil {
		return HistorySearchResult{}, err
	}
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Score == records[j].Score {
			return records[i].UpdatedAt > records[j].UpdatedAt
		}
		return records[i].Score > records[j].Score
	})
	if len(records) > limit {
		records = records[:limit]
	}
	return HistorySearchResult{Records: records, Total: len(records), Query: query, Type: recordType}, nil
}

func (s *KnowledgeStore) addDocumentHistory(add func(HistoryRecord)) error {
	rows, err := s.db.Query(`SELECT id,title,document_type,department,status,source,updated_at FROM documents`)
	if err != nil {
		return fmt.Errorf("读取历史文档失败：%w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, title, docType, department, status, source, updatedAt string
		if err := rows.Scan(&id, &title, &docType, &department, &status, &source, &updatedAt); err != nil {
			return err
		}
		add(HistoryRecord{ID: id, Type: "document", Title: title, Summary: "已入库文档；可通过 document_chunks 检索正文。", Metadata: strings.Join([]string{docType, department, status, source}, " · "), UpdatedAt: updatedAt})
	}
	return rows.Err()
}
func (s *KnowledgeStore) addTicketHistory(add func(HistoryRecord)) error {
	rows, err := s.db.Query(`SELECT id,external_ticket_id,user_message,category,resolution,created_at FROM support_tickets`)
	if err != nil {
		return fmt.Errorf("读取历史工单失败：%w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, externalID, message, category, resolution, createdAt string
		if err := rows.Scan(&id, &externalID, &message, &category, &resolution, &createdAt); err != nil {
			return err
		}
		title := "客服工单"
		if externalID != "" {
			title += " · " + externalID
		}
		add(HistoryRecord{ID: id, Type: "ticket", Title: title, Summary: message, Metadata: strings.Join([]string{category, "处理：" + resolution}, " · "), UpdatedAt: createdAt})
	}
	return rows.Err()
}
func (s *KnowledgeStore) addInterviewHistory(add func(HistoryRecord)) error {
	rows, err := s.db.Query(`SELECT id,persona,scenario,transcript,created_at FROM interviews`)
	if err != nil {
		return fmt.Errorf("读取历史访谈失败：%w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, persona, scenario, transcript, createdAt string
		if err := rows.Scan(&id, &persona, &scenario, &transcript, &createdAt); err != nil {
			return err
		}
		add(HistoryRecord{ID: id, Type: "interview", Title: "客户访谈 · " + persona, Summary: transcript, Metadata: scenario, UpdatedAt: createdAt})
	}
	return rows.Err()
}
func (s *KnowledgeStore) addInsightHistory(add func(HistoryRecord)) error {
	rows, err := s.db.Query(`SELECT id,pain_points_json,behaviors_json,needs_json,confidence,created_at FROM insights`)
	if err != nil {
		return fmt.Errorf("读取历史洞察失败：%w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, painPoints, behaviors, needs, createdAt string
		var confidence float64
		if err := rows.Scan(&id, &painPoints, &behaviors, &needs, &confidence, &createdAt); err != nil {
			return err
		}
		add(HistoryRecord{ID: id, Type: "insight", Title: "需求洞察", Summary: humanizeJSONList(painPoints), Metadata: fmt.Sprintf("行为：%s · 需求：%s · 置信度 %.2f", humanizeJSONList(behaviors), humanizeJSONList(needs), confidence), UpdatedAt: createdAt})
	}
	return rows.Err()
}
func (s *KnowledgeStore) addRequirementHistory(add func(HistoryRecord)) error {
	rows, err := s.db.Query(`SELECT id,product_area,version,status,summary,created_at FROM product_requirements`)
	if err != nil {
		return fmt.Errorf("读取历史 PRD 失败：%w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, area, version, status, summary, createdAt string
		if err := rows.Scan(&id, &area, &version, &status, &summary, &createdAt); err != nil {
			return err
		}
		add(HistoryRecord{ID: id, Type: "prd", Title: "产品需求 · " + area, Summary: summary, Metadata: strings.Join([]string{"v" + version, status}, " · "), UpdatedAt: createdAt})
	}
	return rows.Err()
}
func (s *KnowledgeStore) addResearchNoteHistory(add func(HistoryRecord)) error {
	rows, err := s.db.Query(`SELECT id,title,topic,key_findings_json,open_questions_json,created_at FROM research_notes`)
	if err != nil {
		return fmt.Errorf("读取历史研究笔记失败：%w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, title, topic, findings, questions, createdAt string
		if err := rows.Scan(&id, &title, &topic, &findings, &questions, &createdAt); err != nil {
			return err
		}
		add(HistoryRecord{ID: id, Type: "research_note", Title: title, Summary: humanizeJSONList(findings), Metadata: "待验证：" + humanizeJSONList(questions) + " · " + topic, UpdatedAt: createdAt})
	}
	return rows.Err()
}
func (s *KnowledgeStore) addDatasetHistory(add func(HistoryRecord)) error {
	rows, err := s.db.Query(`SELECT id,name,schema_json,row_count,freshness,created_at FROM structured_datasets`)
	if err != nil {
		return fmt.Errorf("读取历史数据集失败：%w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, name, schema, freshness, createdAt string
		var rowCount int
		if err := rows.Scan(&id, &name, &schema, &rowCount, &freshness, &createdAt); err != nil {
			return err
		}
		add(HistoryRecord{ID: id, Type: "dataset", Title: name, Summary: fmt.Sprintf("%d 行结构化数据", rowCount), Metadata: "字段：" + humanizeJSONList(schema) + " · " + freshness, UpdatedAt: createdAt})
	}
	return rows.Err()
}
func historyScore(terms []string, values ...string) int {
	return keywordScore(terms, values[0], strings.Join(values[1:], " "))
}
func humanizeJSONList(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	value = strings.TrimPrefix(value, "{")
	value = strings.TrimSuffix(value, "}")
	value = strings.ReplaceAll(value, `"`, "")
	value = strings.ReplaceAll(value, ",", " · ")
	return shorten(strings.TrimSpace(value), 220)
}
