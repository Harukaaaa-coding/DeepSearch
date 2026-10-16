package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

//go:embed web/index.html web/app.js
var webFiles embed.FS

func main() {
	engine := NewResearchEngine()
	knowledge, err := NewKnowledgeStore("data")
	if err != nil {
		log.Fatalf("无法初始化内部知识库：%v", err)
	}
	imageJobs, err := newImageJobManager(imageJobDirectory)
	if err != nil {
		log.Fatalf("无法初始化图片任务：%v", err)
	}
	mux := http.NewServeMux()
	registerResearchDiagnosticRoutes(mux, knowledge)
	registerResearchHistoryRoutes(mux, knowledge)

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/research", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input struct {
			Question string `json:"question"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		if err := decoder.Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请输入有效的研究问题。"})
			return
		}
		input.Question = strings.TrimSpace(input.Question)
		if len([]rune(input.Question)) < 8 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "研究问题至少需要 8 个字符，请说明对象、市场或决策目标。"})
			return
		}
		streamResearch(w, r, engine, knowledge, input.Question)
	})
	mux.HandleFunc("/api/knowledge/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, knowledge.Status())
	})
	mux.HandleFunc("/api/agent-memory/overview", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		overview, err := knowledge.AgentMemoryOverview(strings.TrimSpace(r.URL.Query().Get("question")))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, overview)
	})
	mux.HandleFunc("/api/knowledge/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxKnowledgeFileBytes+64<<10)
		if err := r.ParseMultipartForm(maxKnowledgeFileBytes); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请上传不超过 10 MB 的文件。"})
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请选择要导入的文件。"})
			return
		}
		defer file.Close()
		raw, err := io.ReadAll(io.LimitReader(file, maxKnowledgeFileBytes+1))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "读取上传文件失败。"})
			return
		}
		document, err := knowledge.Ingest(header.Filename, raw, KnowledgeMetadata{
			Title: r.FormValue("title"), DocumentType: r.FormValue("documentType"), Department: r.FormValue("department"),
			Version: r.FormValue("version"), Status: r.FormValue("status"), AccessLevel: r.FormValue("accessLevel"),
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"document": document, "status": knowledge.Status()})
	})
	mux.HandleFunc("/api/knowledge/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input struct {
			Query  string `json:"query"`
			Status string `json:"status"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		if err := decoder.Decode(&input); err != nil || strings.TrimSpace(input.Query) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请输入内部资料检索问题。"})
			return
		}
		status := strings.TrimSpace(input.Status)
		if status == "" {
			status = "active"
		}
		writeJSON(w, http.StatusOK, map[string]any{"hits": knowledge.Search(input.Query, status, 8), "status": knowledge.Status(), "retrieval": "keyword + metadata"})
	})
	mux.HandleFunc("/api/knowledge/seed", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		result, err := knowledge.SeedAIAgentDemo()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "导入模拟样本失败：" + err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("/api/knowledge/history", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input struct {
			Query string `json:"query"`
			Type  string `json:"type"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		if err := decoder.Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "历史资料检索请求无效。"})
			return
		}
		result, err := knowledge.SearchHistory(input.Query, input.Type, 30)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("/api/product-analysis", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		result, err := knowledge.BuildProductAnalysis()
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("/api/delivery-copilot", func(w http.ResponseWriter, r *http.Request) {
		var result DeliveryCopilot
		var err error
		switch r.Method {
		case http.MethodGet:
			result, err = knowledge.BuildDeliveryCopilot()
		case http.MethodPost:
			var research ResearchResult
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
			if decodeErr := decoder.Decode(&research); decodeErr != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "研究结果无效，无法生成专属 PRD。"})
				return
			}
			result, err = knowledge.BuildDeliveryCopilotForResearch(research)
			if err == nil {
				if saveErr := knowledge.SaveResearchDelivery(research.RunID, result); saveErr != nil {
					result.HistoryNotice = "PRD 已生成，但未自动归档。请点击保存 PRD 草案；不要直接关闭页面。"
					if researchDiagnosticID.MatchString(research.RunID) {
						log.Printf("research_delivery_archive_failed run_id=%s", research.RunID)
					}
				}
			}
		default:
			w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("/api/research/scenarios", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var result ResearchResult
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&result); err != nil || len([]rune(strings.TrimSpace(result.Question))) < 8 || len(result.Evidence) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "需要本轮研究问题和已读证据，才能重建业务场景。"})
			return
		}
		scenarios, err := engine.buildResearchScenarios(r.Context(), result)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "场景重建失败，原草案保持不变：" + shorten(err.Error(), 200)})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"scenarios": scenarios, "scenarioStatus": "已根据本轮证据重建业务场景假设；未修改 PRD 正文、未生成图片，仍需目标用户验证。"})
	})
	registerImageJobRoutes(mux, imageJobs)
	mux.HandleFunc("/api/generated-images/", serveGeneratedImage)
	mux.HandleFunc("/api/delivery-copilot/coedit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input PRDCoEditRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
		if err := decoder.Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "PRD 共创请求无效。"})
			return
		}
		response, err := engine.ProposePRDCoEdit(r.Context(), input)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, response)
	})
	mux.HandleFunc("/api/delivery-copilot/export", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if format := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("format"))); format != "" && format != "docx" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "当前只支持导出 docx 格式。"})
			return
		}
		var result DeliveryCopilot
		var err error
		if r.Method == http.MethodPost {
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
			if err := decoder.Decode(&result); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "PRD 草案数据无效，无法导出。"})
				return
			}
		} else {
			result, err = knowledge.BuildDeliveryCopilot()
		}
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		data, err := buildDeliveryPRDDOCX(result)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		w.Header().Set("Content-Disposition", `attachment; filename="deepsearch-enterprise-prd.docx"`)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/api/delivery-copilot/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		state, err := knowledge.GetDeliveryWorkflowState()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, state)
	})
	mux.HandleFunc("/api/delivery-copilot/persist", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var copilot DeliveryCopilot
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
		if err := decoder.Decode(&copilot); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "PRD 草案数据无效，无法保存。"})
			return
		}
		state, err := knowledge.PersistDeliveryCopilot(copilot)
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, state)
	})
	mux.HandleFunc("/api/delivery-copilot/submit-review", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		state, err := knowledge.SubmitDeliveryForReview()
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "state": state})
			return
		}
		writeJSON(w, http.StatusOK, state)
	})
	mux.HandleFunc("/api/delivery-copilot/approvals", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input struct {
			Role     string `json:"role"`
			Reviewer string `json:"reviewer"`
			Decision string `json:"decision"`
			Comment  string `json:"comment"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		if err := decoder.Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "审批请求无效。"})
			return
		}
		state, err := knowledge.DecideDeliveryApproval(input.Role, input.Reviewer, input.Decision, input.Comment)
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "state": state})
			return
		}
		writeJSON(w, http.StatusOK, state)
	})
	mux.HandleFunc("/api/export/docx", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var result ResearchResult
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
		if err := decoder.Decode(&result); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "报告数据无效，无法导出 Word 文档。"})
			return
		}
		data, err := buildReportDOCX(result)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		w.Header().Set("Content-Disposition", `attachment; filename="deepsearch-report.docx"`)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/app.js", func(w http.ResponseWriter, r *http.Request) {
		script, err := webFiles.ReadFile("web/app.js")
		if err != nil {
			http.Error(w, "无法加载界面脚本", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		_, _ = w.Write(script)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		page, err := webFiles.ReadFile("web/index.html")
		if err != nil {
			http.Error(w, "无法加载界面", http.StatusInternalServerError)
			return
		}
		page = []byte(strings.Replace(string(page), "<script>/* APP_SCRIPT */</script>", "<script src=\"/app.js\"></script>", 1))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})

	server := &http.Server{Addr: ":8080", Handler: securityHeaders(mux), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("DeepSearch 已启动：http://localhost%s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func streamResearch(w http.ResponseWriter, r *http.Request, engine *ResearchEngine, knowledge *KnowledgeStore, question string) {
	streamResearchWithRunner(w, r, engine.Run, knowledge, question)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; base-uri 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
