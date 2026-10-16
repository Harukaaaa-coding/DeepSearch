package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type modelHTTPResponse struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

// postModelJSON retries only transient connection failures and 5xx gateway errors.
// The same idempotency key is sent on every attempt so compatible providers can avoid
// executing a request twice if the connection is cut after it reaches their edge.
func postModelJSON(ctx context.Context, endpoint, apiKey string, payload []byte, requestName string) (modelHTTPResponse, error) {
	const maxAttempts = 3
	client := &http.Client{Timeout: 120 * time.Second}
	idempotencyKey := modelRequestID(endpoint, payload)
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			return modelHTTPResponse{}, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxAttempts && retryableModelTransportError(err) && waitModelRetry(ctx, attempt) == nil {
				continue
			}
			return modelHTTPResponse{}, modelConnectionError(requestName, attempt, err)
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt < maxAttempts && retryableModelTransportError(readErr) && waitModelRetry(ctx, attempt) == nil {
				continue
			}
			return modelHTTPResponse{}, modelConnectionError(requestName, attempt, readErr)
		}
		result := modelHTTPResponse{StatusCode: resp.StatusCode, Body: body, Header: resp.Header.Clone()}
		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			return result, nil
		}
		lastErr = fmt.Errorf("%s 返回 HTTP %d：%s", requestName, resp.StatusCode, shorten(string(body), 420))
		if attempt < maxAttempts && retryableModelStatus(resp.StatusCode) && waitModelRetry(ctx, attempt) == nil {
			continue
		}
		return modelHTTPResponse{}, lastErr
	}
	return modelHTTPResponse{}, lastErr
}

func modelRequestID(endpoint string, payload []byte) string {
	seed := []byte(fmt.Sprintf("%s:%d:", endpoint, time.Now().UTC().UnixNano()))
	sum := sha256.Sum256(append(seed, payload...))
	return "deepsearch-" + hex.EncodeToString(sum[:16])
}

func retryableModelStatus(status int) bool {
	return status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

func retryableModelTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var networkError net.Error
	if errors.As(err, &networkError) && (networkError.Timeout() || networkError.Temporary()) {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "connection reset") || strings.Contains(text, "broken pipe") || strings.Contains(text, "server closed idle connection") || strings.Contains(text, "unexpected eof")
}

func waitModelRetry(ctx context.Context, attempt int) error {
	delay := time.Duration(attempt) * 700 * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func modelConnectionError(requestName string, attempts int, err error) error {
	if errors.Is(err, io.EOF) || strings.Contains(strings.ToLower(err.Error()), "eof") {
		return fmt.Errorf("%s连接在响应前被提前关闭（EOF），共尝试 %d 次（重试 %d 次）；请稍后重试，并检查 B.AI 服务状态、API 额度或本机代理", requestName, attempts, max(0, attempts-1))
	}
	return fmt.Errorf("%s网络请求失败，共尝试 %d 次（重试 %d 次）：%w", requestName, attempts, max(0, attempts-1), err)
}
