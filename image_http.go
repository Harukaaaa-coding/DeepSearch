package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultImageTimeout = 600 * time.Second
const maxImageResponseBytes = 48 << 20 // A 30 MiB PNG is about 40 MiB as base64.

type imageRequestError struct {
	status  string
	message string
	cause   error
}

func (e *imageRequestError) Error() string { return e.message }
func (e *imageRequestError) Unwrap() error { return e.cause }

func imageGenerationStatus(err error) string {
	var requestErr *imageRequestError
	if errors.As(err, &requestErr) {
		return requestErr.status
	}
	// A malformed response or local save failure can happen after the provider
	// completed a billable generation. They are not proof of non-execution.
	return "unknown"
}

func unknownImageResult(message string, cause error) error {
	return &imageRequestError{status: "unknown", message: message + "；生成结果未知，未自动重试。供应商可能已处理或计费，请先核对供应商记录再决定是否重试。", cause: cause}
}

func imageTimeoutFromValue(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultImageTimeout, nil
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 30 || seconds > 1800 {
		return 0, errors.New("image_model.timeout_seconds 必须为 30–1800 之间的整数秒")
	}
	return time.Duration(seconds) * time.Second, nil
}

// Image creation is a side-effecting request. A dropped response does not prove
// the job was not accepted: unlike text synthesis this never resubmits a POST.
func postImageJSON(ctx context.Context, provider imageProvider, payload []byte) (modelHTTPResponse, error) {
	timeout := provider.Timeout
	if timeout <= 0 {
		timeout = defaultImageTimeout
	}
	// A dedicated non-reused connection also prevents net/http's implicit
	// replay of an idempotency-labelled POST on a broken pooled connection.
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, DisableKeepAlives: true, TLSHandshakeTimeout: 30 * time.Second}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return modelHTTPResponse{}, &imageRequestError{status: "failed", message: "图片请求地址无效，未发送请求", cause: err}
	}
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Close = true
	req.Header.Set("Idempotency-Key", modelRequestID(provider.Endpoint, payload))
	resp, err := client.Do(req)
	if err != nil {
		message := "图片请求连接中断"
		if errors.Is(err, context.DeadlineExceeded) {
			message = fmt.Sprintf("图片请求等待超过 %.0f 秒", timeout.Seconds())
		} else if errors.Is(err, context.Canceled) {
			message = "图片请求等待被中断"
		}
		return modelHTTPResponse{}, unknownImageResult(message, err)
	}
	defer resp.Body.Close()
	// Explicit rejection is a failed request, even if its error body is lost.
	if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusRequestTimeout {
		return modelHTTPResponse{}, &imageRequestError{status: "failed", message: fmt.Sprintf("图片服务返回 HTTP %d，未自动重试；请检查授权、额度或请求参数。", resp.StatusCode)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return modelHTTPResponse{}, unknownImageResult(fmt.Sprintf("图片服务返回 HTTP %d", resp.StatusCode), nil)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxImageResponseBytes+1))
	if err != nil {
		return modelHTTPResponse{}, unknownImageResult("图片响应接收中断，尚未拿到完整图片", err)
	}
	if len(body) > maxImageResponseBytes {
		return modelHTTPResponse{}, unknownImageResult("图片响应超过 48 MiB 安全上限，未保存截断内容", nil)
	}
	return modelHTTPResponse{StatusCode: resp.StatusCode, Body: body, Header: resp.Header.Clone()}, nil
}
