package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const imageJobDirectory = "data/image-jobs"

var (
	errImageJobInvalid = errors.New("图片任务请求无效")
	errImageJobBusy    = errors.New("同时生成的图片任务已达上限，请等待现有任务完成")
	errImageJobUnknown = errors.New("上次请求结果未知，供应商可能已生成或计费；确认该风险后才能重新提交")
)

type ImageGenerationJob struct {
	ID             string              `json:"id"`
	Status         string              `json:"status"`
	Images         []GeneratedPRDImage `json:"images"`
	Warning        string              `json:"warning,omitempty"`
	Provider       string              `json:"provider,omitempty"`
	TimeoutSeconds int                 `json:"timeoutSeconds"`
	RequestIDs     []string            `json:"requestIds,omitempty"`
}

type imageJobProvider func() (imageProvider, error)
type imageJobGenerator func(context.Context, imageProvider, DeliveryCopilot, GeneratedPRDImage, string) (GeneratedPRDImage, error)

type imageJobManager struct {
	mu        sync.Mutex
	dir       string
	jobs      map[string]ImageGenerationJob
	done      map[string]chan struct{}
	active    int
	maxActive int
	provider  imageJobProvider
	generate  imageJobGenerator
}

func newImageJobManager(dir string) (*imageJobManager, error) {
	return newImageJobManagerWithDependencies(dir, func() (imageProvider, error) { return imageProviderFromEnv(os.Getenv) }, generatePRDImage)
}

func newImageJobManagerWithDependencies(dir string, provider imageJobProvider, generate imageJobGenerator) (*imageJobManager, error) {
	if provider == nil || generate == nil {
		return nil, errors.New("图片任务缺少生成依赖")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0700); err != nil {
		return nil, err
	}
	m := &imageJobManager{dir: abs, jobs: make(map[string]ImageGenerationJob), done: make(map[string]chan struct{}), maxActive: 2, provider: provider, generate: generate}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if !validImageJobID(id) {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(abs, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf("无法恢复图片任务 %s：%w", id, readErr)
		}
		job := ImageGenerationJob{}
		if len(data) > 2<<20 || json.Unmarshal(data, &job) != nil || job.ID != id {
			// A damaged ledger is not permission to issue another paid request.
			job = ImageGenerationJob{ID: id, Images: queuedPRDImages(), TimeoutSeconds: int(defaultImageTimeout / time.Second)}
		}
		job.Status = "completed"
		if job.TimeoutSeconds <= 0 {
			job.TimeoutSeconds = int(defaultImageTimeout / time.Second)
		}
		known := make(map[string]GeneratedPRDImage)
		for _, img := range job.Images {
			if img.Kind == "prototype" || img.Kind == "architecture" {
				known[img.Kind] = img
			}
		}
		job.Images = queuedPRDImages()
		for i, base := range job.Images {
			img, ok := known[base.Kind]
			if ok {
				job.Images[i] = img
			}
			img = job.Images[i]
			if !ok || (img.Status != "ready" && img.Status != "failed" && img.Status != "unknown") || (img.Status == "ready" && img.URL == "") {
				img.Status = "unknown"
				img.Error = "服务曾中断，无法确认该图片是否已由供应商生成或计费；不会自动重发。"
			}
			job.Images[i] = img
		}
		job.Warning = imageJobWarning(job.Images)
		if err := m.persist(job); err != nil {
			return nil, err
		}
		m.jobs[id] = job
	}
	return m, nil
}

func validImageJobID(id string) bool {
	if len(id) != 36 || !strings.HasPrefix(id, "img-") {
		return false
	}
	_, err := hex.DecodeString(id[4:])
	return err == nil && id == strings.ToLower(id)
}

func imageJobKey(copilot DeliveryCopilot) string {
	// Scope to the PRD and content actually sent to the image model. Unrelated
	// timestamps and already-generated images cannot trigger new billing.
	data, _ := json.Marshal([]string{copilot.PRD.Info.PRDID, prdPrototypeImagePrompt(copilot), prdArchitectureImagePrompt(copilot)})
	sum := sha256.Sum256(data)
	return "img-" + hex.EncodeToString(sum[:16])
}

func queuedPRDImages() []GeneratedPRDImage {
	return []GeneratedPRDImage{
		{ID: "prototype", Kind: "prototype", Title: "AI 生成页面原型图", Status: "queued"},
		{ID: "architecture", Kind: "architecture", Title: "AI 生成系统架构图", Status: "queued"},
	}
}

func cloneImageJob(job ImageGenerationJob) ImageGenerationJob {
	job.Images = append([]GeneratedPRDImage(nil), job.Images...)
	job.RequestIDs = append([]string(nil), job.RequestIDs...)
	return job
}

func validImageRequestID(id string) bool {
	if len(id) < 16 || len(id) > 80 {
		return false
	}
	for _, ch := range id {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}

func (m *imageJobManager) Start(copilot DeliveryCopilot, retryKinds []string, acknowledgeUnknown bool, requestIDs ...string) (ImageGenerationJob, error) {
	if strings.TrimSpace(copilot.PRD.Info.Product) == "" {
		return ImageGenerationJob{}, fmt.Errorf("%w：请先生成包含产品信息的 PRD", errImageJobInvalid)
	}
	requestID := ""
	if len(requestIDs) > 0 {
		requestID = requestIDs[0]
	}
	if requestID != "" && !validImageRequestID(requestID) {
		return ImageGenerationJob{}, fmt.Errorf("%w：请求标识无效", errImageJobInvalid)
	}
	retry := make(map[string]bool)
	for _, kind := range retryKinds {
		if kind != "prototype" && kind != "architecture" {
			return ImageGenerationJob{}, fmt.Errorf("%w：不支持的图片种类", errImageJobInvalid)
		}
		retry[kind] = true
	}
	id := imageJobKey(copilot)
	m.mu.Lock()
	defer m.mu.Unlock()
	if requestID != "" {
		for _, previous := range m.jobs {
			for _, stored := range previous.RequestIDs {
				if stored != requestID {
					continue
				}
				if previous.ID != id {
					return ImageGenerationJob{}, fmt.Errorf("%w：请求标识已经用于其他 PRD 内容", errImageJobInvalid)
				}
				return cloneImageJob(previous), nil
			}
		}
	}
	job, exists := m.jobs[id]
	if !exists {
		if imported, hasLegacy := legacyImageGenerationJob(copilot, id); hasLegacy {
			if err := m.persist(imported); err != nil {
				return ImageGenerationJob{}, fmt.Errorf("无法保存历史图片记录，未发送生成请求：%w", err)
			}
			m.jobs[id] = imported
			job, exists = imported, true
		}
	}
	if exists && len(job.RequestIDs) >= 32 && requestID != "" {
		return ImageGenerationJob{}, fmt.Errorf("%w：该任务请求标识已达安全上限，请先核对已有生成记录", errImageJobInvalid)
	}
	if exists && (job.Status == "running" || len(retry) == 0) {
		if requestID != "" {
			job = cloneImageJob(job)
			job.RequestIDs = append(job.RequestIDs, requestID)
			if err := m.persist(job); err != nil {
				return ImageGenerationJob{}, err
			}
			m.jobs[id] = job
		}
		return cloneImageJob(job), nil
	}
	if !exists && len(retry) > 0 {
		return ImageGenerationJob{}, fmt.Errorf("%w：没有可重试的历史图片任务", errImageJobInvalid)
	}
	if exists {
		for _, img := range job.Images {
			if !retry[img.Kind] {
				continue
			}
			if img.Status != "failed" && img.Status != "unknown" {
				return ImageGenerationJob{}, fmt.Errorf("%w：只允许重试失败或结果未知的图片", errImageJobInvalid)
			}
			if img.Status == "unknown" && !acknowledgeUnknown {
				return ImageGenerationJob{}, errImageJobUnknown
			}
		}
	}
	if m.active >= m.maxActive {
		return ImageGenerationJob{}, errImageJobBusy
	}
	provider, err := m.provider()
	if err != nil {
		return ImageGenerationJob{}, err
	}
	if provider.Timeout <= 0 {
		provider.Timeout = defaultImageTimeout
	}
	if !exists {
		job = ImageGenerationJob{ID: id, Images: queuedPRDImages()}
	} else {
		job = cloneImageJob(job)
	}
	if requestID != "" {
		job.RequestIDs = append(job.RequestIDs, requestID)
	}
	for i := range job.Images {
		if !exists || retry[job.Images[i].Kind] {
			kind, title := job.Images[i].Kind, job.Images[i].Title
			job.Images[i] = GeneratedPRDImage{ID: kind, Kind: kind, Title: title, Status: "queued"}
		}
	}
	job.Status = "running"
	job.Provider = provider.Label
	job.TimeoutSeconds = int(provider.Timeout / time.Second)
	job.Warning = imageJobWarning(job.Images)
	if err := m.persist(job); err != nil {
		return ImageGenerationJob{}, fmt.Errorf("无法保存图片任务，未发送生成请求：%w", err)
	}
	m.jobs[id] = job
	m.done[id] = make(chan struct{})
	m.active++
	// The job is intentionally independent of browser disconnects or polling requests.
	go m.run(id, provider, copilot)
	return cloneImageJob(job), nil
}

func (m *imageJobManager) Get(id string) (ImageGenerationJob, bool) {
	if !validImageJobID(id) {
		return ImageGenerationJob{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	return cloneImageJob(job), ok
}

// Migration never generates an image. Existing ready assets are retained, and
// old transport failures become unknown because the old client retried them.
func legacyImageGenerationJob(copilot DeliveryCopilot, id string) (ImageGenerationJob, bool) {
	known := make(map[string]GeneratedPRDImage)
	for _, img := range copilot.GeneratedImages {
		kind := img.Kind
		if kind == "" {
			kind = img.ID
		}
		if kind == "prototype" || kind == "architecture" {
			img.Kind = kind
			known[kind] = img
		}
	}
	if len(known) == 0 && strings.TrimSpace(copilot.ImageJobID) == "" {
		return ImageGenerationJob{}, false
	}
	job := ImageGenerationJob{ID: id, Status: "completed", Images: queuedPRDImages(), TimeoutSeconds: int(defaultImageTimeout / time.Second)}
	for i, base := range job.Images {
		img, ok := known[base.Kind]
		if !ok {
			img = base
			img.Status = "unknown"
			img.Error = "旧版本未留下完整生成记录，请先核对供应商结果；不会自动补发。"
		}
		img.ID, img.Kind, img.Title = base.ID, base.Kind, base.Title
		switch img.Status {
		case "ready":
			if !validLegacyImageURL(img.URL) {
				img.Status = "unknown"
				img.Error = "历史图片记录存在，但文件已缺失或地址无效；请先核对或找回，不会自动重画。"
			}
		case "failed":
			if !legacyImageExplicitRejection(img.Error) {
				img.Status = "unknown"
				img.Error = "旧版本请求结果未知，可能已经生成或计费；" + shorten(img.Error, 300)
			}
		case "unknown":
		default:
			img.Status = "unknown"
			img.Error = "旧版本任务未确认完成，无法确定供应商是否已生成或计费；不会自动重发。"
		}
		job.Images[i] = img
	}
	job.Warning = imageJobWarning(job.Images)
	return job, true
}

func legacyImageExplicitRejection(message string) bool {
	marker := strings.Index(strings.ToUpper(message), "HTTP ")
	if marker < 0 {
		return false
	}
	rest := strings.TrimSpace(message[marker+5:])
	if len(rest) < 3 {
		return false
	}
	code, err := strconv.Atoi(rest[:3])
	return err == nil && code >= 400 && code < 500 && code != http.StatusRequestTimeout
}

func validLegacyImageURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User != nil {
		return false
	}
	if parsed.Scheme == "https" || parsed.Scheme == "http" {
		return parsed.Host != ""
	}
	if parsed.Scheme != "" || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/api/generated-images/") {
		return false
	}
	name := strings.TrimPrefix(parsed.Path, "/api/generated-images/")
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, "/\\") || !strings.HasSuffix(strings.ToLower(name), ".png") {
		return false
	}
	info, err := os.Stat(filepath.Join(generatedImageDirectory, name))
	return err == nil && !info.IsDir()
}

func (m *imageJobManager) Lookup(requestID string) (ImageGenerationJob, bool) {
	if !validImageRequestID(requestID) {
		return ImageGenerationJob{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, job := range m.jobs {
		for _, stored := range job.RequestIDs {
			if stored == requestID {
				return cloneImageJob(job), true
			}
		}
	}
	return ImageGenerationJob{}, false
}

func (m *imageJobManager) Wait(ctx context.Context, id string) (ImageGenerationJob, error) {
	m.mu.Lock()
	job, exists := m.jobs[id]
	done := m.done[id]
	m.mu.Unlock()
	if !exists {
		return ImageGenerationJob{}, errors.New("图片任务不存在")
	}
	if job.Status == "completed" {
		return cloneImageJob(job), nil
	}
	select {
	case <-ctx.Done():
		return ImageGenerationJob{}, ctx.Err()
	case <-done:
		job, _ := m.Get(id)
		return job, nil
	}
}

func (m *imageJobManager) run(id string, provider imageProvider, copilot DeliveryCopilot) {
	defer func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		job := m.jobs[id]
		job.Status = "completed"
		job.Warning = imageJobWarning(job.Images)
		if err := m.persist(job); err != nil {
			job.Warning += "；任务状态未能保存到磁盘，重启后请先确认供应商结果。"
		}
		m.jobs[id] = job
		m.active--
		close(m.done[id])
	}()
	for index, kind := range []string{"prototype", "architecture"} {
		prompt := prdPrototypeImagePrompt(copilot)
		if kind == "architecture" {
			prompt = prdArchitectureImagePrompt(copilot)
		}
		m.mu.Lock()
		job := m.jobs[id]
		img := job.Images[index]
		if img.Status != "queued" {
			m.mu.Unlock()
			continue
		}
		img.Status = "generating"
		img.Prompt = shorten(prompt, 400)
		img.StartedAt = time.Now().UTC()
		img.AttemptID = fmt.Sprintf("%s-%d", id, img.StartedAt.UnixNano())
		job.Images[index] = img
		if err := m.persist(job); err != nil {
			img.Status = "failed"
			img.Error = "无法保存生成状态，未发送图片请求。"
			job.Images[index] = img
			m.jobs[id] = job
			m.mu.Unlock()
			continue
		}
		m.jobs[id] = job
		m.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), provider.Timeout)
		generated, err := m.generate(ctx, provider, copilot, img, prompt)
		cancel()
		if err != nil {
			img.Status = imageGenerationStatus(err)
			message := err.Error()
			if provider.APIKey != "" {
				message = strings.ReplaceAll(message, provider.APIKey, "[redacted]")
			}
			img.Error = shorten(message, 500)
		} else if generated.Status != "ready" || strings.TrimSpace(generated.URL) == "" {
			img.Status = "unknown"
			img.Error = "供应商请求已结束，但没有可展示的最终图片；结果与计费状态需人工确认，不会自动重发。"
		} else {
			img = generated
		}
		m.mu.Lock()
		job = m.jobs[id]
		job.Images[index] = img
		job.Warning = imageJobWarning(job.Images)
		if err := m.persist(job); err != nil {
			job.Warning += "；本张结果未能保存到磁盘，请下载已成功图片。"
		}
		m.jobs[id] = job
		m.mu.Unlock()
	}
}

func imageJobWarning(images []GeneratedPRDImage) string {
	var warnings []string
	for _, img := range images {
		if img.Status == "failed" || img.Status == "unknown" {
			warnings = append(warnings, img.Title+"："+img.Error)
		}
	}
	return strings.Join(warnings, "；")
}

func (m *imageJobManager) persist(job ImageGenerationJob) error {
	if !validImageJobID(job.ID) {
		return errors.New("图片任务标识无效")
	}
	file, err := os.CreateTemp(m.dir, ".image-job-*.tmp")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if err := json.NewEncoder(file).Encode(job); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	// Go's Windows rename uses replace-existing semantics; never remove the old
	// ledger first, so a failed replacement cannot permit an accidental new request.
	return os.Rename(temp, filepath.Join(m.dir, job.ID+".json"))
}

func writeImageJobError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	if errors.Is(err, errImageJobInvalid) {
		status = http.StatusBadRequest
	}
	if errors.Is(err, errImageJobUnknown) {
		status = http.StatusConflict
	}
	if errors.Is(err, errImageJobBusy) {
		status = http.StatusTooManyRequests
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func decodeImageJobRequest(w http.ResponseWriter, r *http.Request, value any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("%w：JSON 无效或超过 4 MB", errImageJobInvalid)
	}
	if decoder.Decode(new(any)) != io.EOF {
		return fmt.Errorf("%w：仅允许一个 JSON 对象", errImageJobInvalid)
	}
	return nil
}

func registerImageJobRoutes(mux *http.ServeMux, manager *imageJobManager) {
	mux.HandleFunc("/api/delivery-copilot/image-jobs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var input struct {
			Copilot            DeliveryCopilot `json:"copilot"`
			RetryKinds         []string        `json:"retryKinds"`
			AcknowledgeUnknown bool            `json:"acknowledgeUnknown"`
			RequestID          string          `json:"requestId"`
		}
		if err := decodeImageJobRequest(w, r, &input); err != nil {
			writeImageJobError(w, err)
			return
		}
		job, err := manager.Start(input.Copilot, input.RetryKinds, input.AcknowledgeUnknown, input.RequestID)
		if err != nil {
			writeImageJobError(w, err)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, job)
	})
	mux.HandleFunc("/api/delivery-copilot/image-jobs/lookup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		job, ok := manager.Lookup(r.URL.Query().Get("requestId"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "未找到已持久化的图片请求；未重新提交生成"})
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, job)
	})
	mux.HandleFunc("/api/delivery-copilot/image-jobs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/delivery-copilot/image-jobs/")
		job, ok := manager.Get(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "图片任务不存在"})
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, job)
	})
	// Keep the old response shape for older clients, but share the same durable
	// deduplication ledger. Disconnecting only stops waiting, not the generation.
	mux.HandleFunc("/api/delivery-copilot/images", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var copilot DeliveryCopilot
		if err := decodeImageJobRequest(w, r, &copilot); err != nil {
			writeImageJobError(w, err)
			return
		}
		job, err := manager.Start(copilot, nil, false)
		if err != nil {
			writeImageJobError(w, err)
			return
		}
		job, err = manager.Wait(r.Context(), job.ID)
		if err != nil {
			return
		}
		writeJSON(w, http.StatusOK, PRDImageGenerationResponse{Images: job.Images, Warning: job.Warning, Provider: job.Provider})
	})
}
