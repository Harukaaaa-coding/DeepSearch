package main

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/html/charset"
	_ "modernc.org/sqlite"
)

const (
	maxKnowledgeFileBytes = 10 << 20
	knowledgeChunkRunes   = 850
)

type KnowledgeStore struct {
	root   string
	rawDir string
	db     *sql.DB
}

type KnowledgeDocument struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	DocumentType string    `json:"documentType"`
	Source       string    `json:"source"`
	FilePath     string    `json:"filePath"`
	OriginalName string    `json:"originalName"`
	Author       string    `json:"author"`
	Department   string    `json:"department"`
	ProductArea  string    `json:"productArea"`
	Version      string    `json:"version"`
	Status       string    `json:"status"`
	AccessLevel  string    `json:"accessLevel"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	ChunkCount   int       `json:"chunkCount"`
}

type KnowledgeChunk struct {
	ID           string `json:"id"`
	DocumentID   string `json:"documentId"`
	ChunkIndex   int    `json:"chunkIndex"`
	SectionTitle string `json:"sectionTitle"`
	Content      string `json:"content"`
	TokenCount   int    `json:"tokenCount"`
}

type KnowledgeSearchHit struct {
	ChunkID       string `json:"chunkId"`
	DocumentID    string `json:"documentId"`
	DocumentTitle string `json:"documentTitle"`
	DocumentType  string `json:"documentType"`
	Department    string `json:"department"`
	Version       string `json:"version"`
	Status        string `json:"status"`
	SectionTitle  string `json:"sectionTitle"`
	Snippet       string `json:"snippet"`
	Score         int    `json:"score"`
	Retrieval     string `json:"retrieval"`
}

type KnowledgeStatus struct {
	Documents           int `json:"documents"`
	Chunks              int `json:"chunks"`
	SupportTickets      int `json:"supportTickets"`
	Interviews          int `json:"interviews"`
	Insights            int `json:"insights"`
	ProductRequirements int `json:"productRequirements"`
	ResearchNotes       int `json:"researchNotes"`
	StructuredDatasets  int `json:"structuredDatasets"`
	Initiatives         int `json:"initiatives"`
	PRDVersions         int `json:"prdVersions"`
	PRDRequirements     int `json:"prdRequirements"`
	EvidenceLinks       int `json:"evidenceLinks"`
	PRDReviews          int `json:"prdReviews"`
	PRDApprovals        int `json:"prdApprovals"`
	AgentRuns           int `json:"agentRuns"`
	AgentEvents         int `json:"agentEvents"`
	MemorySummaries     int `json:"memorySummaries"`
}

type KnowledgeMetadata struct {
	Source       string
	Title        string
	DocumentType string
	Author       string
	Department   string
	ProductArea  string
	Version      string
	Status       string
	AccessLevel  string
}

func NewKnowledgeStore(root string) (*KnowledgeStore, error) {
	root = filepath.Clean(root)
	rawDir := filepath.Join(root, "raw")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(root, "deepsearch.db")
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath)+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("打开本地知识库失败：%w", err)
	}
	store := &KnowledgeStore{root: root, rawDir: rawDir, db: db}
	if err := store.initialize(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *KnowledgeStore) Close() error { return s.db.Close() }

func (s *KnowledgeStore) initialize() error {
	const schema = `
CREATE TABLE IF NOT EXISTS documents (
  id TEXT PRIMARY KEY, title TEXT NOT NULL, document_type TEXT NOT NULL, source TEXT NOT NULL,
  file_path TEXT NOT NULL, original_name TEXT NOT NULL, author TEXT NOT NULL DEFAULT '',
  department TEXT NOT NULL DEFAULT '', product_area TEXT NOT NULL DEFAULT '', version TEXT NOT NULL,
  status TEXT NOT NULL, access_level TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_type ON documents(document_type);
CREATE INDEX IF NOT EXISTS idx_documents_department ON documents(department);
CREATE TABLE IF NOT EXISTS document_chunks (
  id TEXT PRIMARY KEY, document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  chunk_index INTEGER NOT NULL, content TEXT NOT NULL, section_title TEXT NOT NULL DEFAULT '',
  token_count INTEGER NOT NULL, metadata_json TEXT NOT NULL DEFAULT '{}', UNIQUE(document_id, chunk_index)
);
CREATE INDEX IF NOT EXISTS idx_chunks_document ON document_chunks(document_id, chunk_index);
CREATE VIRTUAL TABLE IF NOT EXISTS document_chunks_fts USING fts5(
  content, section_title, document_id UNINDEXED, chunk_id UNINDEXED, tokenize='unicode61'
);
CREATE TABLE IF NOT EXISTS support_tickets (
  id TEXT PRIMARY KEY, document_id TEXT REFERENCES documents(id) ON DELETE SET NULL,
  external_ticket_id TEXT NOT NULL DEFAULT '', user_message TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL DEFAULT '', sub_category TEXT NOT NULL DEFAULT '', sentiment TEXT NOT NULL DEFAULT '',
  product_area TEXT NOT NULL DEFAULT '', resolution TEXT NOT NULL DEFAULT '', occurred_at TEXT, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_support_tickets_document ON support_tickets(document_id);
CREATE TABLE IF NOT EXISTS interviews (
  id TEXT PRIMARY KEY, document_id TEXT REFERENCES documents(id) ON DELETE SET NULL,
  persona TEXT NOT NULL DEFAULT '', scenario TEXT NOT NULL DEFAULT '', transcript TEXT NOT NULL DEFAULT '',
  occurred_at TEXT, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_interviews_document ON interviews(document_id);
CREATE TABLE IF NOT EXISTS insights (
  id TEXT PRIMARY KEY, interview_id TEXT REFERENCES interviews(id) ON DELETE SET NULL,
  pain_points_json TEXT NOT NULL DEFAULT '[]', behaviors_json TEXT NOT NULL DEFAULT '[]', needs_json TEXT NOT NULL DEFAULT '[]',
  evidence_chunk_ids_json TEXT NOT NULL DEFAULT '[]', confidence REAL NOT NULL DEFAULT 0, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_insights_interview ON insights(interview_id);
CREATE TABLE IF NOT EXISTS research_notes (
  id TEXT PRIMARY KEY, document_id TEXT REFERENCES documents(id) ON DELETE SET NULL,
  title TEXT NOT NULL, topic TEXT NOT NULL DEFAULT '', key_findings_json TEXT NOT NULL DEFAULT '[]',
  open_questions_json TEXT NOT NULL DEFAULT '[]', confidence REAL NOT NULL DEFAULT 0, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_research_notes_document ON research_notes(document_id);
CREATE TABLE IF NOT EXISTS structured_datasets (
  id TEXT PRIMARY KEY, document_id TEXT REFERENCES documents(id) ON DELETE SET NULL,
  name TEXT NOT NULL, schema_json TEXT NOT NULL DEFAULT '{}', row_count INTEGER NOT NULL DEFAULT 0,
  freshness TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_structured_datasets_document ON structured_datasets(document_id);CREATE TABLE IF NOT EXISTS product_requirements (
  id TEXT PRIMARY KEY, document_id TEXT REFERENCES documents(id) ON DELETE SET NULL,
  product_area TEXT NOT NULL DEFAULT '', version TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'active',
  summary TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_product_requirements_document ON product_requirements(document_id);
CREATE TABLE IF NOT EXISTS product_initiatives (
  id TEXT PRIMARY KEY, initiative_key TEXT NOT NULL UNIQUE, title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft', owner TEXT NOT NULL DEFAULT '', current_version TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_product_initiatives_status ON product_initiatives(status, updated_at DESC);
CREATE TABLE IF NOT EXISTS prd_versions (
  id TEXT PRIMARY KEY, initiative_id TEXT NOT NULL REFERENCES product_initiatives(id) ON DELETE CASCADE,
  version TEXT NOT NULL, status TEXT NOT NULL, snapshot_json TEXT NOT NULL, change_summary TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(initiative_id, version)
);
CREATE INDEX IF NOT EXISTS idx_prd_versions_initiative ON prd_versions(initiative_id, updated_at DESC);
CREATE TABLE IF NOT EXISTS prd_requirements (
  id TEXT PRIMARY KEY, prd_version_id TEXT NOT NULL REFERENCES prd_versions(id) ON DELETE CASCADE,
  requirement_key TEXT NOT NULL, title TEXT NOT NULL, priority TEXT NOT NULL, user_story TEXT NOT NULL,
  preconditions_json TEXT NOT NULL DEFAULT '[]', main_flow_json TEXT NOT NULL DEFAULT '[]', business_rules_json TEXT NOT NULL DEFAULT '[]',
  exceptions_json TEXT NOT NULL DEFAULT '[]', permissions_json TEXT NOT NULL DEFAULT '[]', acceptance_json TEXT NOT NULL DEFAULT '[]',
  tracking_json TEXT NOT NULL DEFAULT '[]', status TEXT NOT NULL DEFAULT 'draft', created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
  UNIQUE(prd_version_id, requirement_key)
);
CREATE INDEX IF NOT EXISTS idx_prd_requirements_version ON prd_requirements(prd_version_id, priority);
CREATE TABLE IF NOT EXISTS evidence_links (
  id TEXT PRIMARY KEY, prd_requirement_id TEXT NOT NULL REFERENCES prd_requirements(id) ON DELETE CASCADE,
  source_type TEXT NOT NULL, source_id TEXT NOT NULL DEFAULT '', relation TEXT NOT NULL DEFAULT 'supports', note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL, UNIQUE(prd_requirement_id, source_type, source_id, relation)
);
CREATE INDEX IF NOT EXISTS idx_evidence_links_requirement ON evidence_links(prd_requirement_id);
CREATE TABLE IF NOT EXISTS prd_reviews (
  id TEXT PRIMARY KEY, prd_version_id TEXT NOT NULL REFERENCES prd_versions(id) ON DELETE CASCADE,
  reviewer_type TEXT NOT NULL, reviewer_name TEXT NOT NULL DEFAULT '', readiness_score INTEGER NOT NULL,
  verdict TEXT NOT NULL, status TEXT NOT NULL, checks_json TEXT NOT NULL DEFAULT '[]', created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_prd_reviews_version ON prd_reviews(prd_version_id, created_at DESC);
CREATE TABLE IF NOT EXISTS prd_approvals (
  id TEXT PRIMARY KEY, prd_version_id TEXT NOT NULL REFERENCES prd_versions(id) ON DELETE CASCADE,
  role TEXT NOT NULL, reviewer TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'pending', comment TEXT NOT NULL DEFAULT '',
  decided_at TEXT, created_at TEXT NOT NULL, UNIQUE(prd_version_id, role)
);
CREATE INDEX IF NOT EXISTS idx_prd_approvals_version ON prd_approvals(prd_version_id, status);
CREATE TABLE IF NOT EXISTS agent_runs (
  id TEXT PRIMARY KEY, question TEXT NOT NULL, status TEXT NOT NULL, result_summary TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL, completed_at TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_agent_runs_completed ON agent_runs(completed_at DESC);
CREATE TABLE IF NOT EXISTS research_archives (
  run_id TEXT PRIMARY KEY, question TEXT NOT NULL, title TEXT NOT NULL, status TEXT NOT NULL,
  started_at TEXT NOT NULL, completed_at TEXT NOT NULL, result_json TEXT NOT NULL,
  delivery_json TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_research_archives_completed ON research_archives(completed_at DESC, run_id);
CREATE TABLE IF NOT EXISTS research_run_diagnostics (
  run_id TEXT PRIMARY KEY, status TEXT NOT NULL, started_at TEXT NOT NULL, ended_at TEXT NOT NULL DEFAULT '',
  duration_ms INTEGER NOT NULL DEFAULT 0, last_step TEXT NOT NULL DEFAULT '', event_count INTEGER NOT NULL DEFAULT 0,
  warning_count INTEGER NOT NULL DEFAULT 0, error_code TEXT NOT NULL DEFAULT '', error_message TEXT NOT NULL DEFAULT '',
  stats_json TEXT NOT NULL DEFAULT '{}', warnings_json TEXT NOT NULL DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS idx_research_diagnostics_started ON research_run_diagnostics(started_at DESC);
CREATE TABLE IF NOT EXISTS agent_events (
  id TEXT PRIMARY KEY, run_id TEXT NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
  sequence INTEGER NOT NULL, category TEXT NOT NULL, tool TEXT NOT NULL,
  input_summary TEXT NOT NULL DEFAULT '', output_summary TEXT NOT NULL DEFAULT '', importance INTEGER NOT NULL DEFAULT 50,
  created_at TEXT NOT NULL, UNIQUE(run_id, sequence)
);
CREATE INDEX IF NOT EXISTS idx_agent_events_recent ON agent_events(created_at DESC, importance DESC);
CREATE TABLE IF NOT EXISTS agent_memory_summaries (
  scope TEXT PRIMARY KEY, summary TEXT NOT NULL, covered_event_count INTEGER NOT NULL DEFAULT 0, updated_at TEXT NOT NULL
);`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("初始化本地知识库表失败：%w", err)
	}
	return s.db.Ping()
}

func (s *KnowledgeStore) Status() KnowledgeStatus {
	status := KnowledgeStatus{}
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM documents`).Scan(&status.Documents)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM document_chunks`).Scan(&status.Chunks)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM support_tickets`).Scan(&status.SupportTickets)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM interviews`).Scan(&status.Interviews)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM insights`).Scan(&status.Insights)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM product_requirements`).Scan(&status.ProductRequirements)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM research_notes`).Scan(&status.ResearchNotes)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM structured_datasets`).Scan(&status.StructuredDatasets)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM product_initiatives`).Scan(&status.Initiatives)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM prd_versions`).Scan(&status.PRDVersions)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM prd_requirements`).Scan(&status.PRDRequirements)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM evidence_links`).Scan(&status.EvidenceLinks)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM prd_reviews`).Scan(&status.PRDReviews)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM prd_approvals`).Scan(&status.PRDApprovals)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM agent_runs`).Scan(&status.AgentRuns)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM agent_events`).Scan(&status.AgentEvents)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM agent_memory_summaries`).Scan(&status.MemorySummaries)
	return status
}

func (s *KnowledgeStore) Ingest(filename string, raw []byte, metadata KnowledgeMetadata) (KnowledgeDocument, error) {
	if len(raw) == 0 {
		return KnowledgeDocument{}, fmt.Errorf("上传文件为空")
	}
	if len(raw) > maxKnowledgeFileBytes {
		return KnowledgeDocument{}, fmt.Errorf("文件不能超过 10 MB")
	}
	content, err := extractKnowledgeText(filename, raw)
	if err != nil {
		return KnowledgeDocument{}, err
	}
	if len([]rune(content)) < 20 {
		return KnowledgeDocument{}, fmt.Errorf("文件中没有可用于检索的文本")
	}
	id := shortID(fmt.Sprintf("%s:%d:%d", filename, time.Now().UnixNano(), len(raw)))
	cleanName := sanitizeKnowledgeFilename(filename)
	relativePath := filepath.ToSlash(filepath.Join("raw", id+"_"+cleanName))
	title := firstNonEmpty(shorten(strings.TrimSpace(metadata.Title), 160), titleFromFilename(cleanName))
	chunks := makeKnowledgeChunks(id, title, content)
	if len(chunks) == 0 {
		return KnowledgeDocument{}, fmt.Errorf("文件未生成可检索的文本分块")
	}
	now := time.Now().UTC()
	document := KnowledgeDocument{
		ID: id, Title: title, DocumentType: firstNonEmpty(shorten(strings.TrimSpace(metadata.DocumentType), 80), documentTypeFromFilename(cleanName)),
		Source: firstNonEmpty(shorten(strings.TrimSpace(metadata.Source), 80), "internal_upload"), FilePath: relativePath, OriginalName: shorten(cleanName, 180), Author: shorten(strings.TrimSpace(metadata.Author), 100),
		Department: firstNonEmpty(shorten(strings.TrimSpace(metadata.Department), 80), "未分类"), ProductArea: shorten(strings.TrimSpace(metadata.ProductArea), 100),
		Version: firstNonEmpty(shorten(strings.TrimSpace(metadata.Version), 40), "1"), Status: firstNonEmpty(shorten(strings.TrimSpace(metadata.Status), 40), "active"),
		AccessLevel: firstNonEmpty(shorten(strings.TrimSpace(metadata.AccessLevel), 80), "team"), CreatedAt: now, UpdatedAt: now, ChunkCount: len(chunks),
	}
	storedFile := filepath.Join(s.rawDir, id+"_"+cleanName)
	if err := os.WriteFile(storedFile, raw, 0o600); err != nil {
		return KnowledgeDocument{}, fmt.Errorf("保存原始文件失败：%w", err)
	}
	tx, err := s.db.Begin()
	if err != nil {
		_ = os.Remove(storedFile)
		return KnowledgeDocument{}, err
	}
	rollback := func(cause error) (KnowledgeDocument, error) {
		_ = tx.Rollback()
		_ = os.Remove(storedFile)
		return KnowledgeDocument{}, cause
	}
	_, err = tx.Exec(`INSERT INTO documents (id,title,document_type,source,file_path,original_name,author,department,product_area,version,status,access_level,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		document.ID, document.Title, document.DocumentType, document.Source, document.FilePath, document.OriginalName, document.Author, document.Department,
		document.ProductArea, document.Version, document.Status, document.AccessLevel, document.CreatedAt.Format(time.RFC3339Nano), document.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return rollback(fmt.Errorf("写入 documents 表失败：%w", err))
	}
	for _, chunk := range chunks {
		if _, err := tx.Exec(`INSERT INTO document_chunks (id,document_id,chunk_index,content,section_title,token_count,metadata_json) VALUES (?,?,?,?,?,?,?)`, chunk.ID, chunk.DocumentID, chunk.ChunkIndex, chunk.Content, chunk.SectionTitle, chunk.TokenCount, `{}`); err != nil {
			return rollback(fmt.Errorf("写入 document_chunks 表失败：%w", err))
		}
		if _, err := tx.Exec(`INSERT INTO document_chunks_fts (content,section_title,document_id,chunk_id) VALUES (?,?,?,?)`, chunk.Content, chunk.SectionTitle, chunk.DocumentID, chunk.ID); err != nil {
			return rollback(fmt.Errorf("写入本地检索索引失败：%w", err))
		}
	}
	if err := tx.Commit(); err != nil {
		_ = os.Remove(storedFile)
		return KnowledgeDocument{}, fmt.Errorf("提交本地知识库事务失败：%w", err)
	}
	return document, nil
}

func (s *KnowledgeStore) Search(query, status string, limit int) []KnowledgeSearchHit {
	query = strings.TrimSpace(query)
	if query == "" {
		return []KnowledgeSearchHit{}
	}
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	terms := knowledgeTerms(query)
	status = strings.TrimSpace(status)
	rows, err := s.db.Query(`SELECT c.id,c.document_id,c.section_title,c.content,d.title,d.document_type,d.department,d.version,d.status FROM document_chunks c JOIN documents d ON d.id=c.document_id WHERE (? = '' OR d.status = ?)`, status, status)
	if err != nil {
		return []KnowledgeSearchHit{}
	}
	defer rows.Close()
	results := make([]KnowledgeSearchHit, 0, limit)
	for rows.Next() {
		var hit KnowledgeSearchHit
		var content string
		if err := rows.Scan(&hit.ChunkID, &hit.DocumentID, &hit.SectionTitle, &content, &hit.DocumentTitle, &hit.DocumentType, &hit.Department, &hit.Version, &hit.Status); err != nil {
			continue
		}
		score := keywordScore(terms, hit.DocumentTitle+" "+hit.SectionTitle, content)
		if score == 0 {
			continue
		}
		hit.Snippet, hit.Score, hit.Retrieval = shorten(content, 420), score, "keyword + metadata (FTS index ready)"
		results = append(results, hit)
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].DocumentTitle < results[j].DocumentTitle
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}
func extractKnowledgeText(filename string, raw []byte) (string, error) {
	extension := strings.ToLower(filepath.Ext(filename))
	switch extension {
	case ".txt", ".md", ".csv", ".html", ".htm":
		reader, err := charset.NewReader(bytes.NewReader(raw), contentTypeForKnowledge(extension))
		if err != nil {
			return "", fmt.Errorf("文件编码无法识别")
		}
		decoded, err := io.ReadAll(io.LimitReader(reader, maxKnowledgeFileBytes))
		if err != nil {
			return "", err
		}
		text := string(decoded)
		if extension == ".csv" {
			return extractCSVKnowledge(text), nil
		}
		if extension == ".html" || extension == ".htm" {
			text, _ = extractReadableText(text)
		}
		return cleanText(text), nil
	case ".docx":
		return extractDOCXKnowledge(raw)
	default:
		return "", fmt.Errorf("B1 当前支持 txt、md、csv、html 和 docx；PDF 与 xlsx 将在后续解析器中加入")
	}
}

func extractDOCXKnowledge(raw []byte) (string, error) {
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return "", fmt.Errorf("DOCX 文件无法读取")
	}
	for _, file := range archive.File {
		if file.Name != "word/document.xml" {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return "", err
		}
		content, readErr := io.ReadAll(io.LimitReader(reader, maxKnowledgeFileBytes))
		_ = reader.Close()
		if readErr != nil {
			return "", readErr
		}
		xmlText := strings.ReplaceAll(string(content), "</w:p>", "\n")
		xmlText = strings.ReplaceAll(xmlText, "</w:tr>", "\n")
		return cleanText(xmlText), nil
	}
	return "", fmt.Errorf("DOCX 中没有找到正文")
}

func extractCSVKnowledge(text string) string {
	reader := csv.NewReader(strings.NewReader(text))
	records, err := reader.ReadAll()
	if err != nil || len(records) == 0 {
		return cleanText(text)
	}
	headers := records[0]
	var rows []string
	for _, record := range records[1:] {
		var cells []string
		for index, value := range record {
			name := fmt.Sprintf("字段%d", index+1)
			if index < len(headers) && strings.TrimSpace(headers[index]) != "" {
				name = strings.TrimSpace(headers[index])
			}
			cells = append(cells, name+"："+strings.TrimSpace(value))
		}
		if len(cells) > 0 {
			rows = append(rows, strings.Join(cells, "；"))
		}
	}
	return strings.Join(rows, "\n")
}

func makeKnowledgeChunks(documentID, title, text string) []KnowledgeChunk {
	paragraphs := strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\r' })
	chunks := make([]KnowledgeChunk, 0, len(paragraphs)/2+1)
	var buffer strings.Builder
	section := title
	flush := func() {
		content := strings.TrimSpace(buffer.String())
		if content == "" {
			return
		}
		index := len(chunks) + 1
		chunks = append(chunks, KnowledgeChunk{ID: fmt.Sprintf("%s-c%03d", documentID, index), DocumentID: documentID, ChunkIndex: index, SectionTitle: section, Content: content, TokenCount: max(1, len([]rune(content))/2)})
		buffer.Reset()
	}
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		if strings.HasPrefix(paragraph, "#") {
			section = strings.TrimSpace(strings.TrimLeft(paragraph, "#"))
		}
		if buffer.Len() > 0 && len([]rune(buffer.String()))+len([]rune(paragraph))+1 > knowledgeChunkRunes {
			flush()
		}
		if len([]rune(paragraph)) > knowledgeChunkRunes {
			runes := []rune(paragraph)
			for start := 0; start < len(runes); start += knowledgeChunkRunes {
				end := min(len(runes), start+knowledgeChunkRunes)
				buffer.WriteString(string(runes[start:end]))
				flush()
			}
			continue
		}
		if buffer.Len() > 0 {
			buffer.WriteByte('\n')
		}
		buffer.WriteString(paragraph)
	}
	flush()
	return chunks
}

func knowledgeTerms(query string) []string {
	terms := queryTerms(query)
	for _, r := range []rune(strings.ToLower(query)) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			candidate := string(r)
			if !containsString(terms, candidate) {
				terms = append(terms, candidate)
			}
		}
	}
	return terms
}

func keywordScore(terms []string, title, content string) int {
	title, content = strings.ToLower(title), strings.ToLower(content)
	score := 0
	for _, term := range terms {
		term = strings.ToLower(strings.TrimSpace(term))
		if len([]rune(term)) < 1 {
			continue
		}
		if strings.Contains(title, term) {
			score += 8
		}
		if strings.Contains(content, term) {
			score += 3 + strings.Count(content, term)
		}
	}
	return score
}

func sanitizeKnowledgeFilename(name string) string {
	name = filepath.Base(name)
	var output strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '.' || r == '-' || r == '_' {
			output.WriteRune(r)
		} else {
			output.WriteRune('_')
		}
	}
	value := strings.Trim(output.String(), "._")
	if value == "" {
		return "upload.txt"
	}
	return shorten(value, 140)
}

func titleFromFilename(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}
func documentTypeFromFilename(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".csv":
		return "dataset"
	case ".docx":
		return "document"
	default:
		return "note"
	}
}
func contentTypeForKnowledge(extension string) string {
	if extension == ".html" || extension == ".htm" {
		return "text/html"
	}
	return "text/plain"
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
