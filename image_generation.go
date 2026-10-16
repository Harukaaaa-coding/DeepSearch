package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const generatedImageDirectory = "data/generated-images"

type imageProvider struct {
	APIKey   string
	Endpoint string
	Model    string
	Size     string
	Quality  string
	Label    string
	Timeout  time.Duration
}

type GeneratedPRDImage struct {
	ID            string    `json:"id"`
	Kind          string    `json:"kind"`
	Title         string    `json:"title"`
	Status        string    `json:"status"`
	URL           string    `json:"url,omitempty"`
	DownloadURL   string    `json:"downloadUrl,omitempty"`
	Prompt        string    `json:"prompt,omitempty"`
	RevisedPrompt string    `json:"revisedPrompt,omitempty"`
	Error         string    `json:"error,omitempty"`
	GeneratedAt   time.Time `json:"generatedAt,omitempty"`
	StartedAt     time.Time `json:"startedAt,omitempty"`
	AttemptID     string    `json:"attemptId,omitempty"`
}

type PRDImageGenerationResponse struct {
	Images   []GeneratedPRDImage `json:"images"`
	Warning  string              `json:"warning,omitempty"`
	Provider string              `json:"provider,omitempty"`
}

func imageProviderFromEnv(getenv func(string) string) (imageProvider, error) {
	if provider, configured, err := imageProviderFromConfigFile("config.yaml"); err != nil || configured {
		return provider, err
	}
	if apiKey := strings.TrimSpace(getenv("BAI_API_KEY")); apiKey != "" {
		provider, err := newImageProvider("bai", apiKey, getenv("BAI_IMAGE_MODEL"), getenv("BAI_IMAGE_SIZE"), getenv("BAI_IMAGE_QUALITY"))
		if err != nil {
			return imageProvider{}, err
		}
		provider.Timeout, err = imageTimeoutFromValue(getenv("BAI_IMAGE_TIMEOUT_SECONDS"))
		return provider, err
	}
	return imageProvider{}, errors.New("未配置图片模型：请在 config.yaml 的 image_model 中选择 bai / gpt-image-2")
}

func imageProviderFromConfigFile(path string) (imageProvider, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return imageProvider{}, false, nil
	}
	if err != nil {
		return imageProvider{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	values := imageModelYAMLValues(string(data))
	if len(values) == 0 {
		return imageProvider{}, false, nil
	}
	report, reportConfigured, reportErr := reportProviderFromYAML(data)
	if reportErr != nil {
		return imageProvider{}, true, reportErr
	}
	providerName := strings.TrimSpace(values["provider"])
	apiKey := strings.TrimSpace(values["api_key"])
	if apiKey == "" && reportConfigured {
		apiKey = report.APIKey
		if providerName == "" && report.Protocol == "chat_completions" {
			providerName = "bai"
		}
	}
	provider, err := newImageProvider(providerName, apiKey, values["model"], values["size"], values["quality"])
	if err != nil {
		return imageProvider{}, true, err
	}
	provider.Timeout, err = imageTimeoutFromValue(values["timeout_seconds"])
	if err != nil {
		return imageProvider{}, true, err
	}
	return provider, true, nil
}

func imageModelYAMLValues(data string) map[string]string {
	values := make(map[string]string)
	inImageModel := false
	for _, line := range strings.Split(data, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent == 0 {
			inImageModel = strings.HasPrefix(trimmed, "image_model:")
			continue
		}
		if !inImageModel {
			continue
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key != "provider" && key != "api_key" && key != "model" && key != "size" && key != "quality" && key != "timeout_seconds" {
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

func newImageProvider(name, apiKey, model, size, quality string) (imageProvider, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name != "bai" && name != "b.ai" {
		return imageProvider{}, fmt.Errorf("暂只支持 B.AI 图片模型，当前 provider=%q", name)
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return imageProvider{}, errors.New("图片模型缺少 API Key；可复用 report_model.api_key 或在 image_model.api_key 中单独配置")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = "gpt-image-2"
	}
	size = strings.TrimSpace(size)
	if size == "" {
		size = "2528x1696"
	}
	quality = strings.TrimSpace(quality)
	if quality == "" {
		quality = "high"
	}
	return imageProvider{APIKey: apiKey, Endpoint: "https://api.b.ai/v1/images/generations", Model: model, Size: size, Quality: quality, Label: "B.AI / " + model, Timeout: defaultImageTimeout}, nil
}

func (e *ResearchEngine) GeneratePRDImages(ctx context.Context, copilot DeliveryCopilot) (PRDImageGenerationResponse, error) {
	if strings.TrimSpace(copilot.PRD.Info.Product) == "" {
		return PRDImageGenerationResponse{}, errors.New("请先生成 PRD，再生成视觉方案")
	}
	provider, err := imageProviderFromEnv(os.Getenv)
	if err != nil {
		return PRDImageGenerationResponse{}, err
	}
	specs := []struct {
		id, kind, title, prompt string
	}{
		{id: "prototype", kind: "prototype", title: "AI 生成页面原型图", prompt: prdPrototypeImagePrompt(copilot)},
		{id: "architecture", kind: "architecture", title: "AI 生成系统架构图", prompt: prdArchitectureImagePrompt(copilot)},
	}
	response := PRDImageGenerationResponse{Images: make([]GeneratedPRDImage, 0, len(specs)), Provider: provider.Label}
	warnings := make([]string, 0, len(specs))
	for _, spec := range specs {
		image := GeneratedPRDImage{ID: spec.id, Kind: spec.kind, Title: spec.title, Status: "generating", Prompt: shorten(spec.prompt, 400)}
		result, generateErr := generatePRDImage(ctx, provider, copilot, image, spec.prompt)
		if generateErr != nil {
			image.Status = imageGenerationStatus(generateErr)
			image.Error = shorten(generateErr.Error(), 360)
			warnings = append(warnings, spec.title+"未生成："+image.Error)
			response.Images = append(response.Images, image)
			continue
		}
		response.Images = append(response.Images, result)
	}
	response.Warning = strings.Join(warnings, "；")
	return response, nil
}

func generatePRDImage(ctx context.Context, provider imageProvider, copilot DeliveryCopilot, image GeneratedPRDImage, prompt string) (GeneratedPRDImage, error) {
	payload, err := json.Marshal(map[string]any{
		"background":    "opaque",
		"model":         provider.Model,
		"moderation":    "auto",
		"n":             1,
		"output_format": "png",
		"prompt":        prompt,
		"quality":       provider.Quality,
		"size":          provider.Size,
	})
	if err != nil {
		return image, err
	}
	response, err := postImageJSON(ctx, provider, payload)
	if err != nil {
		return image, err
	}
	var body struct {
		Data []struct {
			B64JSON       string `json:"b64_json"`
			URL           string `json:"url"`
			RevisedPrompt string `json:"revised_prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body, &body); err != nil || len(body.Data) == 0 {
		return image, unknownImageResult("图片服务未返回完整、可识别的图片数据", err)
	}
	item := body.Data[0]
	image.RevisedPrompt = shorten(strings.TrimSpace(item.RevisedPrompt), 800)
	image.GeneratedAt = time.Now().UTC()
	if encoded := compactBase64(item.B64JSON); encoded != "" {
		attemptID := image.AttemptID
		if attemptID == "" {
			attemptID = shortID(time.Now().UTC().String())
		}
		fileName := fmt.Sprintf("%s-%s-%s.png", safeImageName(copilot.PRD.Info.PRDID), safeImageName(image.ID), safeImageName(attemptID))
		if err := saveGeneratedPNG(fileName, encoded); err != nil {
			return image, unknownImageResult("图片响应已收到，但本地校验或保存失败", err)
		}
		image.Status = "ready"
		image.URL = "/api/generated-images/" + fileName
		image.DownloadURL = image.URL + "?download=1"
		return image, nil
	}
	if remoteURL := strings.TrimSpace(item.URL); remoteURL != "" {
		parsed, err := url.Parse(remoteURL)
		if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return image, unknownImageResult("图片模型返回了不安全的图片地址", nil)
		}
		image.Status = "ready"
		image.URL = remoteURL
		image.DownloadURL = remoteURL
		return image, nil
	}
	return image, unknownImageResult("图片模型响应缺少 b64_json 或 url", nil)
}

func prdPrototypeImagePrompt(copilot DeliveryCopilot) string {
	prd := copilot.PRD
	return shorten(fmt.Sprintf(`Create one polished desktop product-wireframe concept board for this PRD. Show three connected wide web screens in a 3:2 landscape image: (1) workbench / task list, (2) evidence and source review, (3) PRD review with editable requirements. Use an understated Chinese enterprise software style, clean grid, cards, source-confidence badges, review states, neutral ivory and sage palette. The image must communicate layout hierarchy and interaction affordances; do not invent market numbers, brand logos, or unreadable dense paragraphs. This is a product prototype concept, not marketing artwork.
Product: %s
Feature: %s
User problem: %s
Users: %s
Screens: %s
Top requirements: %s`, prd.Info.Product, prd.Info.Feature, prd.Background.UserProblem, prdImageUsers(prd.Users), prdImageScreens(prd.Prototype.Screens), prdImageRequirements(prd.Requirements)), 5200)
}

func prdArchitectureImagePrompt(copilot DeliveryCopilot) string {
	prd := copilot.PRD
	return shorten(fmt.Sprintf(`Create one readable technical system architecture diagram for this PRD in a 3:2 landscape image. Use five clearly separated layers with directional arrows: users and web UI; research planner and PRD co-edit service; web search, page/PDF reader and model services; evidence store, local knowledge base and generated-image storage; governance for permissions, audit, monitoring and rollback. Use concise Chinese labels, strong visual grouping, clear data-flow arrows, restrained blue/green technical-diagram style. Do not use company logos or invent third-party integrations. It should be suitable for a product and engineering review.
Product: %s
Feature: %s
Architecture: %s
Data entities: %s
APIs: %s
Constraints: %s`, prd.Info.Product, prd.Info.Feature, prdImageArchitecture(prd.Architecture), shortStringList(copilot.Solution.DataEntities, 5), shortStringList(copilot.Solution.APIs, 5), shortStringList(copilot.Solution.TechnicalConstraints, 4)), 5200)
}

func prdImageUsers(items []PRDUserScenario) string {
	values := make([]string, 0, min(len(items), 3))
	for _, item := range items {
		values = append(values, item.Role+"："+item.Scenario)
		if len(values) == 3 {
			break
		}
	}
	return shortStringList(values, 3)
}

func prdImageScreens(items []PrototypeScreen) string {
	values := make([]string, 0, min(len(items), 4))
	for _, item := range items {
		values = append(values, item.Name+"："+item.Purpose)
		if len(values) == 4 {
			break
		}
	}
	return shortStringList(values, 4)
}

func prdImageRequirements(items []ExecutionRequirement) string {
	values := make([]string, 0, min(len(items), 4))
	for _, item := range items {
		values = append(values, item.Title)
		if len(values) == 4 {
			break
		}
	}
	return shortStringList(values, 4)
}

func prdImageArchitecture(items []ArchitectureModule) string {
	values := make([]string, 0, min(len(items), 5))
	for _, item := range items {
		values = append(values, item.Name+"："+shortStringList(item.Children, 3))
		if len(values) == 5 {
			break
		}
	}
	return shortStringList(values, 5)
}

func shortStringList(items []string, limit int) string {
	values := make([]string, 0, min(len(items), limit))
	for _, item := range items {
		item = shorten(strings.TrimSpace(item), 180)
		if item != "" {
			values = append(values, item)
		}
		if len(values) == limit {
			break
		}
	}
	return strings.Join(values, "；")
}

func compactBase64(value string) string {
	return strings.Map(func(character rune) rune {
		if character == ' ' || character == '\n' || character == '\r' || character == '\t' {
			return -1
		}
		return character
	}, strings.TrimSpace(value))
}

func safeImageName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Map(func(character rune) rune {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '-' || character == '_' {
			return character
		}
		return '-'
	}, value)
	value = strings.Trim(value, "-")
	if value == "" {
		value = shortID(time.Now().UTC().String())
	}
	return shorten(value, 72)
}

func saveGeneratedPNG(fileName, encoded string) error {
	if filepath.Base(fileName) != fileName || !strings.HasSuffix(strings.ToLower(fileName), ".png") {
		return errors.New("不安全的图片文件名")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("图片模型返回的 PNG 编码无效：%w", err)
	}
	if len(data) < 8 || len(data) > 30<<20 {
		return errors.New("图片模型返回的文件大小异常")
	}
	if string(data[:8]) != "\x89PNG\r\n\x1a\n" {
		return errors.New("图片模型返回的文件不是 PNG")
	}
	if err := os.MkdirAll(generatedImageDirectory, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(generatedImageDirectory, fileName), data, 0o600)
}

func serveGeneratedImage(w http.ResponseWriter, r *http.Request) {
	fileName := strings.TrimPrefix(r.URL.Path, "/api/generated-images/")
	if fileName == "" || filepath.Base(fileName) != fileName || !strings.HasSuffix(strings.ToLower(fileName), ".png") {
		http.NotFound(w, r)
		return
	}
	filePath := filepath.Join(generatedImageDirectory, fileName)
	if _, err := os.Stat(filePath); err != nil {
		http.NotFound(w, r)
		return
	}
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeFile(w, r, filePath)
}
