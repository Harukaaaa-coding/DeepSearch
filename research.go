package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	stdhtml "html"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/ledongthuc/pdf"
	"golang.org/x/net/html/charset"
)

const (
	maxResultsPerQuery = 6
	maxPapersPerQuery  = 3
	maxPagesPerRun     = 36
	maxPDFBytes        = 14 << 20
)

var (
	errNoReportModel = errors.New("no report model is configured: set DEEPSEEK_API_KEY or OPENAI_API_KEY")
	anchorRE         = regexp.MustCompile(`(?is)<a\s+([^>]+)>(.*?)</a>`)
	hrefRE           = regexp.MustCompile(`(?is)\bhref\s*=\s*["']([^"']+)["']`)
	snippetRE        = regexp.MustCompile(`(?is)<[^>]*class\s*=\s*["'][^"']*result__snippet[^"']*["'][^>]*>(.*?)</[^>]+>`)
	titleRE          = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	articleRE        = regexp.MustCompile(`(?is)<article[^>]*>(.*?)</article>`)
	mainRE           = regexp.MustCompile(`(?is)<main[^>]*>(.*?)</main>`)
	paragraphRE      = regexp.MustCompile(`(?is)<(?:p|h[1-3]|li|blockquote)[^>]*>(.*?)</(?:p|h[1-3]|li|blockquote)>`)
	noiseRE          = regexp.MustCompile(`(?is)<(?:script|style|noscript|svg|canvas|iframe)[^>]*>.*?</(?:script|style|noscript|svg|canvas|iframe)>`)
	tagRE            = regexp.MustCompile(`(?is)<[^>]+>`)
	spaceRE          = regexp.MustCompile(`\s+`)
	publishedRE      = regexp.MustCompile(`(?is)<meta[^>]+(?:property|name)\s*=\s*["'](?:article:published_time|date|publishdate|pubdate)["'][^>]+content\s*=\s*["']([^"']+)`)
	publishedRE2     = regexp.MustCompile(`(?is)<meta[^>]+content\s*=\s*["']([^"']+)["'][^>]+(?:property|name)\s*=\s*["'](?:article:published_time|date|publishdate|pubdate)["']`)
)

type ResearchEngine struct {
	client *http.Client
}

type reportProvider struct {
	APIKey        string
	Endpoint      string
	Model         string
	Label         string
	Protocol      string
	SupportsStore bool
}

type ProgressEvent struct {
	RunID         string            `json:"runId,omitempty"`
	Kind          string            `json:"kind"`
	Step          string            `json:"step,omitempty"`
	Message       string            `json:"message"`
	Current       int               `json:"current,omitempty"`
	Total         int               `json:"total,omitempty"`
	Discovered    int               `json:"discovered,omitempty"`
	ReadCount     int               `json:"readCount,omitempty"`
	EvidenceCount int               `json:"evidenceCount,omitempty"`
	Coverage      *ResearchCoverage `json:"coverage,omitempty"`
	At            time.Time         `json:"at"`
}

type ResearchResult struct {
	ArchiveStatus    string              `json:"archiveStatus,omitempty"`
	Status           string              `json:"status,omitempty"`
	RunID            string              `json:"runId"`
	Question         string              `json:"question"`
	StartedAt        time.Time           `json:"startedAt"`
	CompletedAt      time.Time           `json:"completedAt"`
	Plan             []PlanStep          `json:"plan"`
	Stats            Stats               `json:"stats"`
	Report           Report              `json:"report"`
	Sources          []Source            `json:"sources"`
	Evidence         []Evidence          `json:"evidence"`
	Coverage         ResearchCoverage    `json:"coverage"`
	Competition      *CompetitorAnalysis `json:"competition,omitempty"`
	ProductBlueprint *ProductBlueprint   `json:"productBlueprint,omitempty"`
	Scenarios        []ProductScenario   `json:"scenarios,omitempty"`
	ScenarioStatus   string              `json:"scenarioStatus,omitempty"`
	BlueprintStatus  string              `json:"blueprintStatus,omitempty"`
}

type PlanStep struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type Stats struct {
	Queries          int `json:"queries"`
	Discovered       int `json:"discovered"`
	Read             int `json:"read"`
	EvidenceCount    int `json:"evidenceCount"`
	HighConfidence   int `json:"highConfidence"`
	MediumConfidence int `json:"mediumConfidence"`
	ConflictCount    int `json:"conflictCount"`
}

type Report struct {
	Title       string   `json:"title"`
	Analysis    string   `json:"analysis"`
	Method      string   `json:"method"`
	Limitations []string `json:"limitations"`
	Conflicts   []string `json:"conflicts"`
	ModelUsed   string   `json:"modelUsed"`
}

type Source struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Domain      string    `json:"domain"`
	Kind        string    `json:"kind"`
	Query       string    `json:"query"`
	Authors     string    `json:"authors,omitempty"`
	PublishedAt string    `json:"publishedAt,omitempty"`
	FetchedAt   time.Time `json:"fetchedAt"`
	Trust       int       `json:"trust"`
	Status      string    `json:"status"`
	Extractor   string    `json:"extractor,omitempty"`
	Error       string    `json:"error,omitempty"`
	Content     string    `json:"-"`
}

type Evidence struct {
	ID          string    `json:"id"`
	SourceID    string    `json:"sourceId"`
	SourceTitle string    `json:"sourceTitle"`
	URL         string    `json:"url"`
	Quote       string    `json:"quote"`
	Score       int       `json:"score"`
	Confidence  string    `json:"confidence"`
	Tags        []string  `json:"tags"`
	PublishedAt string    `json:"publishedAt,omitempty"`
	FetchedAt   time.Time `json:"fetchedAt,omitempty"`
}

type searchHit struct {
	Title, URL, Snippet, Query string
	Kind, Authors, PublishedAt string
	PDFURL, Abstract           string
}

type arxivFeed struct {
	Entries []arxivEntry `xml:"entry"`
}

type arxivEntry struct {
	ID        string `xml:"id"`
	Title     string `xml:"title"`
	Summary   string `xml:"summary"`
	Published string `xml:"published"`
	Authors   []struct {
		Name string `xml:"name"`
	} `xml:"author"`
}

func NewResearchEngine() *ResearchEngine {
	return &ResearchEngine{client: &http.Client{Timeout: 14 * time.Second}}
}

func (e *ResearchEngine) Run(ctx context.Context, question string, emit func(ProgressEvent)) (ResearchResult, error) {
	identity := researchRunIdentityFromContext(ctx)
	result := ResearchResult{RunID: identity.ID, Question: question, StartedAt: identity.StartedAt, Plan: buildPlan()}
	if err := e.gatherResearch(ctx, &result, emit); err != nil {
		return result, err
	}
	conflicts := findConflicts(result.Evidence)
	result.Stats.ConflictCount = len(conflicts)

	emit(progress("plan", "synthesize", "正在把证据、相关性与潜在冲突连接为报告。", 5, 5))
	report, err := e.buildReport(ctx, question, result.Evidence, result.Stats, conflicts, result.Coverage)
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if err != nil {
		message := "模型合成请求失败，已输出保守的证据报告：" + shorten(err.Error(), 96)
		if errors.Is(err, errNoReportModel) {
			message = "未配置报告模型：请在项目根目录 config.yaml 的 report_model.api_key 填写有效密钥。"
		} else if strings.Contains(err.Error(), "HTTP 401") {
			message = "模型服务鉴权失败（HTTP 401）：请检查 config.yaml 的 provider 与 api_key 是否来自同一服务。"
		}
		emit(progress("warning", "synthesize", message, 5, 5))
		report = fallbackReport(question, result.Evidence, result.Stats, conflicts)
	}
	report.Limitations = append(report.Limitations, result.Coverage.Gaps...)
	result.Report = report
	if wantsCompetitorAnalysis(question) {
		if err != nil || len(result.Evidence) == 0 {
			competition := competitorAnalysisFallback(result)
			result.Competition = &competition
			emit(progress("warning", "synthesize", "报告模型未完成，竞品区仅保留证据缺口，不额外请求模型。", 5, 5))
		} else {
			emit(progress("plan", "synthesize", "正在从已读公开资料构建竞品功能、定位与机会矩阵。", 5, 5))
			competition, competitionErr := e.buildCompetitorAnalysis(ctx, result)
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			result.Competition = &competition
			if competitionErr != nil {
				emit(progress("warning", "synthesize", "竞品分析证据不足："+shorten(competitionErr.Error(), 110), 5, 5))
			} else {
				emit(progress("sources", "synthesize", fmt.Sprintf("已生成 %d 个带证据的竞品画像与 %d 条待验证机会。", len(competition.Competitors), len(competition.Opportunities)), 5, 5))
			}
		}
	}
	if isProductDiscoveryTask(question) {
		if err != nil || len(result.Evidence) == 0 {
			result.BlueprintStatus = "报告模型未完成，未生成产品蓝图；当前仅保留证据底稿。"
		} else {
			emit(progress("plan", "synthesize", "正在从本轮证据拆解目标用户的业务触发、操作障碍、拟议步骤和验收结果。", 5, 5))
			scenarios, scenarioErr := e.buildResearchScenarios(ctx, result)
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			if scenarioErr != nil {
				result.ScenarioStatus = "场景分析尚未完成：" + shorten(scenarioErr.Error(), 160)
				emit(progress("warning", "synthesize", result.ScenarioStatus, 5, 5))
			} else {
				result.Scenarios = scenarios
				result.ScenarioStatus = "已生成研究相关的业务场景假设；引用原文及适用范围仍需核验，不能当作已验证客户行为。"
			}
			emit(progress("plan", "synthesize", "正在把研究结论转成产品蓝图、需求、原型、数据与交付计划。", 5, 5))
			blueprint, blueprintErr := e.buildProductBlueprint(ctx, result)
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			if blueprintErr != nil {
				result.BlueprintStatus = "产品蓝图模型生成失败：" + shorten(blueprintErr.Error(), 120)
				emit(progress("warning", "synthesize", result.BlueprintStatus, 5, 5))
			} else {
				result.ProductBlueprint = &blueprint
				result.BlueprintStatus = "已生成研究专属产品蓝图：用户、方案、MVP、流程、原型、数据、测试与交付计划均待人工评审。"
			}
		}
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	result.Plan, result.CompletedAt = updatePlan(result.Plan, "done"), time.Now().UTC()
	emit(progress("finished", "synthesize", "研究完成：每条证据都可回到原始网址核验。", 5, 5))
	return result, nil
}

func (e *ResearchEngine) collectHits(ctx context.Context, queries []string, emit func(ProgressEvent), step string) []searchHit {
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)
	hits := make([]searchHit, 0, len(queries)*maxResultsPerQuery)
	for i, query := range queries {
		i, query := i, query
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			emit(progress("search", step, "检索："+shorten(query, 72), i+1, len(queries)))
			found, err := e.search(ctx, query)
			if err != nil {
				emit(progress("warning", step, "该检索未返回结果："+shorten(query, 44), i+1, len(queries)))
				return
			}
			mu.Lock()
			hits = append(hits, found...)
			mu.Unlock()
			emit(ProgressEvent{Kind: "sources", Step: step, Message: fmt.Sprintf("本轮发现 %d 条候选资料。", len(found)), Current: i + 1, Total: len(queries), Discovered: len(found), At: time.Now().UTC()})
		}()
	}
	wg.Wait()
	return hits
}

func (e *ResearchEngine) search(ctx context.Context, query string) ([]searchHit, error) {
	endpoint := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DeepSearchResearch/0.1 (+local evidence research)")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("search returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	page := string(body)
	if containsAny(page, "anomaly-modal", "bots use DuckDuckGo", "g-recaptcha") {
		return nil, errors.New("search requires human verification")
	}
	return parseDuckDuckGo(page, query), nil
}

// searchPapers queries arXiv's public Atom endpoint. arXiv entries are labelled
// as preprints: they may be valuable primary technical material, but are not
// automatically peer-reviewed evidence.
func (e *ResearchEngine) searchPapers(ctx context.Context, query string) ([]searchHit, error) {
	params := url.Values{}
	if !strings.Contains(query, ":") {
		query = "all:" + query
	}
	params.Set("search_query", query)
	params.Set("start", "0")
	params.Set("max_results", fmt.Sprint(maxPapersPerQuery))
	params.Set("sortBy", "relevance")
	params.Set("sortOrder", "descending")
	endpoint := "https://export.arxiv.org/api/query?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DeepSearchResearch/0.2 (local evidence research)")
	req.Header.Set("Accept", "application/atom+xml,application/xml;q=0.9,text/xml;q=0.8")
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("arXiv returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	return parseArxiv(string(body), query), nil
}

func parseArxiv(feedXML, query string) []searchHit {
	var feed arxivFeed
	if err := xml.Unmarshal([]byte(feedXML), &feed); err != nil {
		return nil
	}
	hits := make([]searchHit, 0, min(len(feed.Entries), maxPapersPerQuery))
	for _, entry := range feed.Entries {
		absURL := strings.TrimSpace(entry.ID)
		if !isSafePublicURL(absURL) {
			continue
		}
		absURL = strings.Replace(absURL, "http://arxiv.org/", "https://arxiv.org/", 1)
		abstract := normalizePaperText(entry.Summary)
		title := normalizePaperText(entry.Title)
		if title == "" || len([]rune(abstract)) < 80 {
			continue
		}
		authorNames := make([]string, 0, len(entry.Authors))
		for _, author := range entry.Authors {
			if name := strings.TrimSpace(author.Name); name != "" {
				authorNames = append(authorNames, name)
			}
		}
		hits = append(hits, searchHit{
			Title:       title,
			URL:         absURL,
			Snippet:     shorten(abstract, 420),
			Query:       query,
			Kind:        "论文 / arXiv 预印本",
			Authors:     shortenAuthors(authorNames),
			PublishedAt: normalizePaperDate(entry.Published),
			PDFURL:      strings.Replace(absURL, "/abs/", "/pdf/", 1) + ".pdf",
			Abstract:    abstract,
		})
		if len(hits) == maxPapersPerQuery {
			break
		}
	}
	return hits
}

func normalizePaperText(value string) string {
	return strings.TrimSpace(spaceRE.ReplaceAllString(value, " "))
}

func normalizePaperDate(value string) string {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return shorten(strings.TrimSpace(value), 32)
	}
	return parsed.Format("2006-01-02")
}

func shortenAuthors(authors []string) string {
	if len(authors) == 0 {
		return ""
	}
	if len(authors) > 4 {
		return strings.Join(authors[:4], ", ") + fmt.Sprintf(" 等 %d 位作者", len(authors))
	}
	return strings.Join(authors, ", ")
}
func parseDuckDuckGo(page, query string) []searchHit {
	matches := anchorRE.FindAllStringSubmatchIndex(page, -1)
	hits := make([]searchHit, 0, maxResultsPerQuery)
	for _, match := range matches {
		attrs := page[match[2]:match[3]]
		if !strings.Contains(attrs, "result__a") {
			continue
		}
		href := hrefRE.FindStringSubmatch(attrs)
		if len(href) < 2 {
			continue
		}
		link := unwrapDuckDuckGo(stdhtml.UnescapeString(href[1]))
		if !isSafePublicURL(link) {
			continue
		}
		title := cleanText(page[match[4]:match[5]])
		if title == "" {
			continue
		}
		end := min(len(page), match[1]+4000)
		snippet := ""
		if part := snippetRE.FindStringSubmatch(page[match[1]:end]); len(part) > 1 {
			snippet = cleanText(part[1])
		}
		hits = append(hits, searchHit{Title: title, URL: link, Snippet: snippet, Query: query})
		if len(hits) == maxResultsPerQuery {
			break
		}
	}
	return hits
}

func unwrapDuckDuckGo(raw string) string {
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if strings.Contains(strings.ToLower(u.Host), "duckduckgo.com") {
		if target := u.Query().Get("uddg"); target != "" {
			if decoded, err := url.QueryUnescape(target); err == nil {
				return decoded
			}
			return target
		}
		return ""
	}
	return raw
}

func (e *ResearchEngine) readSources(ctx context.Context, hits []searchHit, emit func(ProgressEvent)) []Source {
	if len(hits) > maxPagesPerRun {
		hits = hits[:maxPagesPerRun]
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	sources := make([]Source, 0, len(hits))
	for i, hit := range hits {
		i, hit := i, hit
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			kind := hit.Kind
			if kind == "" {
				kind = sourceType(domainOf(hit.URL))
			}
			emit(progress("read", "read", "阅读 "+kind+"："+shorten(hit.Title, 52), i+1, len(hits)))
			source := e.readSource(ctx, hit)
			mu.Lock()
			sources = append(sources, source)
			mu.Unlock()
			if source.Status == "已阅读" {
				label := "已提取网页正文："
				if strings.Contains(source.Kind, "论文") {
					label = "已提取论文" + source.Extractor + "："
				}
				emit(ProgressEvent{Kind: "source-read", Step: "read", Message: label + shorten(source.Title, 48), Current: i + 1, Total: len(hits), ReadCount: 1, At: time.Now().UTC()})
			}
		}()
	}
	wg.Wait()
	return sources
}

func (e *ResearchEngine) readSource(ctx context.Context, hit searchHit) Source {
	domain := domainOf(hit.URL)
	kind := hit.Kind
	if kind == "" {
		kind = sourceType(domain)
	}
	source := Source{ID: shortID(hit.URL), Title: hit.Title, URL: hit.URL, Domain: domain, Kind: kind, Query: hit.Query, Authors: hit.Authors, PublishedAt: hit.PublishedAt, Trust: trustForDomain(domain), FetchedAt: time.Now().UTC(), Status: "未读取"}
	if !isSafePublicURL(hit.URL) {
		source.Error = "安全策略跳过了非公开地址。"
		return source
	}
	if hit.Abstract != "" {
		return e.readPaperSource(ctx, source, hit)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hit.URL, nil)
	if err != nil {
		source.Error = "无法创建读取请求。"
		return source
	}
	req.Header.Set("User-Agent", "DeepSearchResearch/0.1 (+local evidence research)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9,*/*;q=0.2")
	resp, err := e.client.Do(req)
	if err != nil {
		source.Error = "来源暂时不可访问。"
		return source
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		source.Error = fmt.Sprintf("来源返回 HTTP %d。", resp.StatusCode)
		return source
	}
	contentTypeHeader := resp.Header.Get("Content-Type")
	contentType := strings.ToLower(contentTypeHeader)
	if strings.Contains(contentType, "pdf") {
		content, err := readPDFBody(resp.Body)
		if err != nil || len([]rune(content)) < 180 {
			source.Error = "PDF 正文提取失败或为扫描版；未计为有效证据。"
			return source
		}
		source.Content, source.Status, source.Extractor = content, "已阅读", "公开 PDF 全文"
		source.Kind = "公开 PDF 文档"
		return source
	}
	if contentType != "" && !strings.Contains(contentType, "html") && !strings.Contains(contentType, "text/") {
		source.Error = "此版本只读取 HTML、纯文本或公开 PDF 来源。"
		return source
	}
	raw, err := decodePageText(resp.Body, contentTypeHeader)
	if err != nil {
		source.Error = "网页编码无法识别或正文读取失败。"
		return source
	}
	if strings.Count(raw, "�") >= 3 {
		source.Error = "网页文本编码异常，已跳过避免乱码证据。"
		return source
	}
	content, extractor := extractReadableText(raw)
	if len([]rune(content)) < 180 {
		source.Error = "未能从页面提取足够正文。"
		return source
	}
	if pageTitle := pageTitle(raw); len([]rune(pageTitle)) > 4 {
		source.Title = pageTitle
	}
	source.Content, source.PublishedAt, source.Status, source.Extractor = content, publishedAt(raw), "已阅读", extractor
	return source
}

func (e *ResearchEngine) readPaperSource(ctx context.Context, source Source, hit searchHit) Source {
	if hit.PDFURL != "" && isSafePublicURL(hit.PDFURL) {
		content, err := e.readPDFText(ctx, hit.PDFURL)
		if err == nil && len([]rune(content)) >= 180 {
			source.Content, source.Status, source.Extractor = content, "已阅读", "公开 PDF 全文"
			return source
		}
		if err != nil {
			source.Error = "公开 PDF 全文未能提取，已回退为论文摘要。"
		}
	}
	abstract := normalizePaperText(hit.Abstract)
	if len([]rune(abstract)) < 80 {
		source.Error = "论文摘要不足，且公开 PDF 全文不可读。"
		return source
	}
	source.Content = "论文摘要：" + abstract
	source.Status, source.Extractor = "已阅读", "公开论文摘要"
	return source
}

func (e *ResearchEngine) readPDFText(ctx context.Context, pdfURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pdfURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "DeepSearchResearch/0.2 (local evidence research)")
	req.Header.Set("Accept", "application/pdf,*/*;q=0.4")
	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("PDF returned HTTP %d", resp.StatusCode)
	}
	if contentType := strings.ToLower(resp.Header.Get("Content-Type")); contentType != "" && !strings.Contains(contentType, "pdf") {
		return "", fmt.Errorf("returned a non-PDF response")
	}
	return readPDFBody(resp.Body)
}

func readPDFBody(body io.Reader) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(body, maxPDFBytes+1))
	if err != nil {
		return "", err
	}
	if len(raw) > maxPDFBytes {
		return "", fmt.Errorf("PDF exceeds the %d MB reading limit", maxPDFBytes>>20)
	}
	file, err := os.CreateTemp("", "deepsearch-paper-*.pdf")
	if err != nil {
		return "", err
	}
	filename := file.Name()
	defer os.Remove(filename)
	if _, err := file.Write(raw); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return extractPDFFileText(filename)
}

func extractPDFFileText(filename string) (content string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("PDF text extraction failed")
		}
	}()
	file, reader, err := pdf.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	plain, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}
	decoded, err := io.ReadAll(io.LimitReader(plain, 2500<<10))
	if err != nil {
		return "", err
	}
	content = strings.TrimSpace(spaceRE.ReplaceAllString(string(decoded), " "))
	if strings.Count(content, "�") >= 3 {
		return "", fmt.Errorf("PDF text encoding is not reliable")
	}
	return shorten(content, 24000), nil
}
func decodePageText(body io.Reader, contentType string) (string, error) {
	reader, err := charset.NewReader(io.LimitReader(body, 1500<<10), contentType)
	if err != nil {
		return "", err
	}
	decoded, err := io.ReadAll(io.LimitReader(reader, 1500<<10))
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
func extractEvidence(question string, sources []Source) []Evidence {
	items := make([]Evidence, 0, len(sources))
	for _, source := range sources {
		if source.Status != "已阅读" || source.Content == "" {
			continue
		}
		quote, relevance, tags := bestPassage(question, source.Content)
		if quote == "" {
			continue
		}
		score := int(math.Round(float64(source.Trust)*0.55 + relevance*45))
		if containsNumber(quote) {
			score += 4
		}
		if source.PublishedAt != "" {
			score += 3
		}
		score = min(96, max(35, score))
		confidence := "低"
		if score >= 76 {
			confidence = "高"
		} else if score >= 59 {
			confidence = "中"
		}
		items = append(items, Evidence{ID: fmt.Sprintf("E%d", len(items)+1), SourceID: source.ID, SourceTitle: source.Title, URL: source.URL, Quote: quote, Score: score, Confidence: confidence, Tags: tags})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Score > items[j].Score })
	for i := range items {
		items[i].ID = fmt.Sprintf("E%d", i+1)
	}
	return items
}

func bestPassage(question, content string) (string, float64, []string) {
	terms := queryTerms(question)
	best, bestScore := "", -1.0
	for _, sentence := range splitSentences(content) {
		sentence = strings.TrimSpace(sentence)
		length := len([]rune(sentence))
		if length < 45 || length > 420 {
			continue
		}
		lower, matches := strings.ToLower(sentence), 0
		for _, term := range terms {
			if strings.Contains(lower, strings.ToLower(term)) {
				matches++
			}
		}
		score := 0.28 + math.Min(0.52, float64(matches)*0.17)
		if containsNumber(sentence) {
			score += 0.09
		}
		if containsAny(lower, "增长", "market", "预计", "报告", "adoption") {
			score += 0.06
		}
		if score > bestScore {
			best, bestScore = sentence, score
		}
	}
	if best == "" {
		return "", 0, nil
	}
	return shorten(best, 360), math.Min(1, bestScore), passageTags(best)
}

func (e *ResearchEngine) buildReport(ctx context.Context, question string, evidence []Evidence, stats Stats, conflicts []string, coverage ...ResearchCoverage) (Report, error) {
	provider, err := reportProviderFromEnv(os.Getenv)
	if err != nil {
		return Report{}, err
	}
	if len(evidence) == 0 {
		return fallbackReport(question, evidence, stats, conflicts), nil
	}
	input := evidencePrompt(question, evidence, stats)
	if len(coverage) > 0 {
		input += coveragePrompt(coverage[0])
	}
	instructions := reportSynthesisPrompt() + "\n事实须逐项标注支持它的 [E#]；分析推断和产品假设分别标注，不得引用未提供的 ID。对市场数字写明地区、时间和统计口径；定价注明币种、单位、套餐及核验日期，缺失则未核验。区分官方主张、客户行为证据与独立评测。资料只达到初步覆盖时，输出有边界的结论和最优先的补证动作，不宣称完整市场全景。"
	text, err := e.requestModelText(ctx, provider, instructions, input, 5000)
	if err != nil {
		return Report{}, err
	}
	return Report{Title: "证据约束研究报告", Analysis: text, Method: fmt.Sprintf("按市场、用户、竞品、定价、技术与风险缺口进行有界多轮补证；阅读 %d 个来源，提取 %d 条相关摘录。覆盖门槛不是事实认证，模型只能使用提供的证据。", stats.Read, len(evidence)), Limitations: commonLimitations(stats), Conflicts: conflictsOrFallback(conflicts), ModelUsed: provider.Label}, nil
}

func (e *ResearchEngine) requestModelText(ctx context.Context, provider reportProvider, instructions, input string, maxTokens int) (string, error) {
	body, err := json.Marshal(modelRequestPayload(provider, instructions, input, maxTokens))
	if err != nil {
		return "", err
	}
	response, err := postModelJSON(ctx, provider.Endpoint, provider.APIKey, body, "报告模型")
	if err != nil {
		return "", err
	}
	return checkedResponseOutputText(response.Body)
}

func modelRequestPayload(provider reportProvider, instructions, input string, maxTokens int) map[string]any {
	if provider.Protocol == "chat_completions" {
		return map[string]any{
			"model":       provider.Model,
			"messages":    []map[string]string{{"role": "system", "content": instructions}, {"role": "user", "content": input}},
			"temperature": 0.2,
			"max_tokens":  maxTokens,
		}
	}
	payload := map[string]any{"model": provider.Model, "instructions": instructions, "input": input, "max_output_tokens": maxTokens}
	if provider.SupportsStore {
		payload["store"] = false
	}
	return payload
}

func reportProviderFromEnv(getenv func(string) string) (reportProvider, error) {
	if provider, configured, err := reportProviderFromConfigFile("config.yaml"); err != nil || configured {
		return provider, err
	}
	if apiKey := strings.TrimSpace(getenv("DEEPSEEK_API_KEY")); apiKey != "" {
		return newReportProvider("deepseek", apiKey, getenv("DEEPSEEK_MODEL"))
	}
	if apiKey := strings.TrimSpace(getenv("OPENAI_API_KEY")); apiKey != "" {
		return newReportProvider("openai", apiKey, getenv("OPENAI_MODEL"))
	}
	return reportProvider{}, errNoReportModel
}

func reportProviderFromConfigFile(path string) (reportProvider, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return reportProvider{}, false, nil
	}
	if err != nil {
		return reportProvider{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	return reportProviderFromYAML(data)
}

func reportProviderFromYAML(data []byte) (reportProvider, bool, error) {
	values := reportModelYAMLValues(string(data))
	apiKey := values["api_key"]
	if apiKey == "" {
		return reportProvider{}, false, nil
	}
	provider, err := newReportProvider(values["provider"], apiKey, values["model"])
	if err != nil {
		return reportProvider{}, true, err
	}
	return provider, true, nil
}

func reportModelYAMLValues(data string) map[string]string {
	values := make(map[string]string)
	inReportModel := false
	for _, line := range strings.Split(data, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent == 0 {
			inReportModel = strings.HasPrefix(trimmed, "report_model:")
			continue
		}
		if !inReportModel {
			continue
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key != "provider" && key != "api_key" && key != "model" {
			continue
		}
		value = strings.TrimSpace(value)
		if comment := strings.Index(value, " #"); comment >= 0 {
			value = strings.TrimSpace(value[:comment])
		}
		values[key] = strings.Trim(value, "\"'")
	}
	return values
}

func newReportProvider(name, apiKey, model string) (reportProvider, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return reportProvider{}, errNoReportModel
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "deepseek":
		if model = strings.TrimSpace(model); model == "" {
			model = "deepseek-flash"
		}
		return reportProvider{APIKey: apiKey, Endpoint: "https://api.deepseek.com/responses", Model: model, Label: "DeepSeek / " + model}, nil
	case "bai", "b.ai":
		if model = strings.TrimSpace(model); model == "" {
			model = "gpt-5.2"
		}
		return reportProvider{APIKey: apiKey, Endpoint: "https://api.b.ai/v1/chat/completions", Model: model, Label: "B.AI / " + model, Protocol: "chat_completions"}, nil
	case "openai":
		if model = strings.TrimSpace(model); model == "" {
			model = "gpt-6-astra"
		}
		return reportProvider{APIKey: apiKey, Endpoint: "https://api.openai.com/v1/responses", Model: model, Label: "OpenAI / " + model, SupportsStore: true}, nil
	default:
		return reportProvider{}, fmt.Errorf("unsupported report model provider %q", name)
	}
}
func responseOutputText(body []byte) string {
	text, _ := checkedResponseOutputText(body)
	return text
}

func fallbackReport(question string, evidence []Evidence, stats Stats, conflicts []string) Report {
	if len(evidence) == 0 {
		return emptyReport(question, "已完成检索与阅读尝试，但没有提取到足以支撑判断的正文证据。")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## 执行摘要\n\n本次研究围绕“%s”完成两轮检索，获得 %d 条可追溯证据，其中高置信度 %d 条。以下是证据优先的研究底稿：它展示了应继续验证的信号，但不会把有限材料包装成确定的创业或投资结论。\n\n", question, len(evidence), stats.HighConfidence)
	b.WriteString("## 已验证的相关信号\n\n")
	for _, item := range evidence[:min(5, len(evidence))] {
		fmt.Fprintf(&b, "- **[%s] %s**（%s置信度，%d/100）：%s\n", item.ID, item.SourceTitle, item.Confidence, item.Score, item.Quote)
	}
	b.WriteString("\n## 机会与下一步\n\n现有材料可用来筛选需求、竞争或政策信号。形成具体产品路线、市场规模或创业判断前，应补齐原始数据、目标用户访谈和独立来源的交叉验证。\n\n## 风险与反证\n\n自动提取的段落不能证明因果关系，也不能替代原始来源的定义、样本和时间口径核验。\n")
	return Report{Title: "可审计研究底稿", Analysis: b.String(), Method: fmt.Sprintf("两轮检索与学术索引；发现 %d 个候选来源；阅读 %d 个来源；从正文、公开论文 PDF 或摘要抽取 %d 条相关段落。", stats.Discovered, stats.Read, len(evidence)), Limitations: commonLimitations(stats), Conflicts: conflictsOrFallback(conflicts), ModelUsed: "保守规则合成（模型合成未完成）"}
}

func emptyReport(question, detail string) Report {
	return Report{Title: "证据不足的研究记录", Analysis: "## 未形成结论\n\n“" + question + "”本次未获得可验证的网页正文。\n\n" + detail + "\n\n请稍后重试，或换成更具体的问题（对象、区域、时间范围与比较维度）。", Method: "系统没有使用无来源的文本补齐结论。", Limitations: []string{"检索或网页读取没有返回可用于引用的正文。"}, Conflicts: []string{"没有足够证据可供冲突检查。"}, ModelUsed: "未调用模型"}
}

func evidencePrompt(question string, evidence []Evidence, stats Stats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "研究问题：%s\n研究覆盖：%d 次搜索、%d 个已阅读网页、%d 条证据。\n仅可使用下列证据：\n", question, stats.Queries, stats.Read, len(evidence))
	for _, item := range evidence {
		fmt.Fprintf(&b, "\n[%s]\n标题：%s\n网址：%s\n来源发布日期：%s；抓取日期：%s（不是发布日期）\n启发式可信度：%s（%d/100，不是事实正确概率）\n正文摘录：%s\n", item.ID, item.SourceTitle, item.URL, item.PublishedAt, item.FetchedAt.Format(time.RFC3339), item.Confidence, item.Score, item.Quote)
	}
	return b.String()
}

func buildPlan() []PlanStep {
	return []PlanStep{
		{ID: "scope", Title: "问题定义", Description: "明确研究对象、决策目标与证据门槛。", Status: "active"},
		{ID: "discover", Title: "市场与产品", Description: "搜索市场、需求、产品和竞争信号。", Status: "pending"},
		{ID: "verify", Title: "交叉验证", Description: "补充官方、时间与反向证据。", Status: "pending"},
		{ID: "read", Title: "阅读与提取", Description: "从网页、公开论文 PDF 或摘要提取可回溯段落。", Status: "pending"},
		{ID: "synthesize", Title: "报告与证据链", Description: "连接结论、来源和不确定性。", Status: "pending"},
	}
}

func updatePlan(plan []PlanStep, state string) []PlanStep {
	for i := range plan {
		if state == "blocked" && i > 2 {
			plan[i].Status = "pending"
		} else {
			plan[i].Status = state
		}
	}
	return plan
}

func firstRoundQueries(question string) []string {
	topic := searchTopic(question)
	if isProductDiscoveryTask(question) {
		if hasHan(topic) {
			return []string{topic, topic + " 目标用户 使用场景 痛点", topic + " 竞品 功能 工作流", topic + " MVP 数据权限 原型"}
		}
		return []string{topic, topic + " target users jobs to be done pain points", topic + " competitors features workflows", topic + " MVP data permissions prototype"}
	}
	if hasHan(topic) {
		return []string{topic, topic + " 市场规模 增长 报告", topic + " 竞争格局 产品 采用"}
	}
	return []string{topic, topic + " market size growth report", topic + " competitors products adoption"}
}

func paperSearchQuery(question string) string {
	return searchTopic(question)
}

func secondRoundQueries(question string) []string {
	topic := searchTopic(question)
	if isProductDiscoveryTask(question) {
		if hasHan(topic) {
			return []string{topic + " 用户反馈 访谈 失败案例", topic + " 合规 风险 成本 集成", topic + " 竞品 功能 对比 定价 客户案例"}
		}
		return []string{topic + " user feedback interviews failure cases", topic + " compliance risks cost integrations", topic + " competitors feature comparison pricing customer cases"}
	}
	if hasHan(topic) {
		return []string{topic + " 政策 风险 监管 数据", topic + " 研究 报告 数据集"}
	}
	return []string{topic + " policy risks regulation data", topic + " research report dataset"}
}
func searchTopic(question string) string {
	return shorten(strings.Trim(strings.TrimSpace(question), "。！？?!"), 120)
}

func deduplicateHits(hits []searchHit) []searchHit {
	seen := make(map[string]bool, len(hits))
	unique := make([]searchHit, 0, len(hits))
	for _, hit := range hits {
		key := canonicalURL(hit.URL)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, hit)
	}
	return unique
}

func prioritizeHits(hits []searchHit) []searchHit {
	sort.SliceStable(hits, func(i, j int) bool {
		leftPaper := strings.Contains(hits[i].Kind, "论文")
		rightPaper := strings.Contains(hits[j].Kind, "论文")
		if leftPaper != rightPaper {
			return leftPaper
		}
		return trustForDomain(domainOf(hits[i].URL)) > trustForDomain(domainOf(hits[j].URL))
	})
	return hits
}
func extractReadableText(raw string) (string, string) {
	withoutNoise := noiseRE.ReplaceAllString(raw, " ")
	segment, extractor := withoutNoise, "page text"
	if match := articleRE.FindStringSubmatch(withoutNoise); len(match) > 1 {
		segment, extractor = match[1], "article"
	} else if match := mainRE.FindStringSubmatch(withoutNoise); len(match) > 1 {
		segment, extractor = match[1], "main"
	}
	var parts []string
	for _, paragraph := range paragraphRE.FindAllStringSubmatch(segment, -1) {
		if len(paragraph) > 1 {
			clean := cleanText(paragraph[1])
			if len([]rune(clean)) >= 30 {
				parts = append(parts, clean)
			}
		}
	}
	text := strings.Join(parts, "\n")
	if len([]rune(text)) < 180 {
		text = cleanText(segment)
	}
	return shorten(text, 24000), extractor
}

func pageTitle(raw string) string {
	match := titleRE.FindStringSubmatch(raw)
	if len(match) < 2 {
		return ""
	}
	return shorten(cleanText(match[1]), 160)
}

func publishedAt(raw string) string {
	for _, re := range []*regexp.Regexp{publishedRE, publishedRE2} {
		if match := re.FindStringSubmatch(raw); len(match) > 1 {
			return shorten(strings.TrimSpace(stdhtml.UnescapeString(match[1])), 40)
		}
	}
	return ""
}

func cleanText(input string) string {
	input = tagRE.ReplaceAllString(input, " ")
	input = stdhtml.UnescapeString(input)
	return strings.TrimSpace(spaceRE.ReplaceAllString(input, " "))
}
func splitSentences(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool { return r == '。' || r == '！' || r == '？' || r == '!' || r == '?' || r == '\n' })
}

func queryTerms(question string) []string {
	words := strings.FieldsFunc(strings.ToLower(question), func(r rune) bool { return !(unicode.IsLetter(r) || unicode.IsNumber(r)) })
	seen, terms := map[string]bool{}, make([]string, 0, 8)
	for _, word := range words {
		runes := []rune(word)
		if len(runes) < 3 {
			continue
		}
		if hasHan(word) && len(runes) > 7 {
			word = string(runes[:min(6, len(runes))])
		}
		if !seen[word] {
			seen[word] = true
			terms = append(terms, word)
		}
		if len(terms) == 8 {
			break
		}
	}
	return terms
}

func passageTags(text string) []string {
	lower := strings.ToLower(text)
	tags := make([]string, 0, 3)
	for term, label := range map[string]string{"市场": "市场", "增长": "增长", "market": "市场", "竞争": "竞争", "compet": "竞争", "监管": "监管", "policy": "政策", "风险": "风险", "fund": "资金"} {
		if strings.Contains(lower, term) {
			tags = append(tags, label)
		}
	}
	if len(tags) == 0 {
		tags = append(tags, "相关材料")
	}
	return uniqueStrings(tags)
}

func findConflicts(evidence []Evidence) []string {
	var conflicts []string
	for i := 0; i < len(evidence); i++ {
		for j := i + 1; j < len(evidence); j++ {
			a, b := strings.ToLower(evidence[i].Quote), strings.ToLower(evidence[j].Quote)
			opposing := (containsAny(a, "增长", "increase", "rising") && containsAny(b, "下降", "decrease", "decline")) || (containsAny(b, "增长", "increase", "rising") && containsAny(a, "下降", "decrease", "decline"))
			if opposing && sharedTag(evidence[i].Tags, evidence[j].Tags) {
				conflicts = append(conflicts, fmt.Sprintf("[%s] 与 [%s] 对同一主题出现方向相反的表述；需核对统计口径、区域和时间范围。", evidence[i].ID, evidence[j].ID))
			}
		}
	}
	return uniqueStrings(conflicts)
}

func sharedTag(a, b []string) bool {
	for _, left := range a {
		for _, right := range b {
			if left == right {
				return true
			}
		}
	}
	return false
}
func conflictsOrFallback(conflicts []string) []string {
	if len(conflicts) > 0 {
		return conflicts
	}
	return []string{"未发现词面上的方向性冲突；这不代表不同来源的定义、样本或时间范围完全一致。"}
}

func trustForDomain(domain string) int {
	domain = strings.ToLower(domain)
	switch {
	case domain == "arxiv.org":
		return 74
	case strings.HasSuffix(domain, ".gov"), strings.HasSuffix(domain, ".gov.cn"), strings.Contains(domain, ".gov."):
		return 91
	case strings.HasSuffix(domain, ".edu"), strings.HasSuffix(domain, ".edu.cn"), strings.Contains(domain, ".ac."):
		return 85
	case strings.HasSuffix(domain, ".org"), containsAny(domain, "who.int", "worldbank.org", "oecd.org"):
		return 80
	case containsAny(domain, "reuters.com", "apnews.com", "ft.com", "bloomberg.com", "nature.com", "science.org"):
		return 76
	case containsAny(domain, "mckinsey.com", "deloitte.com", "pwc.com", "gartner.com", "statista.com", "cbinsights.com"):
		return 67
	case strings.HasSuffix(domain, ".com"), strings.HasSuffix(domain, ".cn"):
		return 56
	default:
		return 52
	}
}

func sourceType(domain string) string {
	domain = strings.ToLower(domain)
	switch {
	case domain == "arxiv.org":
		return "论文 / arXiv 预印本"
	case strings.HasSuffix(domain, ".gov"), strings.HasSuffix(domain, ".gov.cn"), strings.Contains(domain, ".gov."):
		return "官方文档"
	case strings.HasSuffix(domain, ".edu"), strings.HasSuffix(domain, ".edu.cn"), strings.Contains(domain, ".ac.") || strings.HasSuffix(domain, ".org"):
		return "研究机构"
	case containsAny(domain, "reuters.com", "apnews.com", "ft.com", "bloomberg.com"):
		return "新闻媒体"
	case containsAny(domain, "github.com", "docs.", "openai.com", "anthropic.com", "google.com", "microsoft.com"):
		return "公司/技术资料"
	default:
		return "网页来源"
	}
}

func isSafePublicURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified()
	}
	return true
}

func canonicalURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	u.Fragment = ""
	query := u.Query()
	for _, name := range []string{"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content", "gclid", "fbclid"} {
		query.Del(name)
	}
	u.RawQuery = query.Encode()
	return strings.TrimRight(u.String(), "/")
}

func domainOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
}
func commonLimitations(stats Stats) []string {
	return []string{"可信度评分衡量来源类型、正文相关度、可定位日期等信号，不等于事实正确率。", fmt.Sprintf("本次阅读了 %d 个可访问来源；论文优先提取公开 PDF 全文，失败时只使用可核验摘要。付费墙、扫描版 PDF、动态页面和未收录资料仍可能缺失。", stats.Read), "arXiv 属于预印本平台，并不等同于同行评审；引用论文仍需回到原文核对方法、样本、版本与发布时间。"}
}
func progress(kind, step, message string, current, total int) ProgressEvent {
	return ProgressEvent{Kind: kind, Step: step, Message: message, Current: current, Total: total, At: time.Now().UTC()}
}
func containsNumber(value string) bool { return strings.IndexFunc(value, unicode.IsNumber) >= 0 }
func containsAny(value string, items ...string) bool {
	for _, item := range items {
		if strings.Contains(value, item) {
			return true
		}
	}
	return false
}
func uniqueStrings(items []string) []string {
	seen, output := map[string]bool{}, make([]string, 0, len(items))
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			output = append(output, item)
		}
	}
	return output
}
func hasHan(value string) bool {
	for _, r := range value {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}
func shorten(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "…"
}
func shortID(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:10]
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
