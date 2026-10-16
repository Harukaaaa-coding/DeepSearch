package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	errIncompleteModelOutput = errors.New("报告模型输出不完整，未作为成功结果")
	errInvalidModelOutput    = errors.New("报告模型没有返回有效的完整文本")
)

type modelTextPart struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Refusal string `json:"refusal"`
}

type modelTextChoice struct {
	Index        *int            `json:"index"`
	FinishReason json.RawMessage `json:"finish_reason"`
	Message      struct {
		Content      json.RawMessage   `json:"content"`
		Refusal      json.RawMessage   `json:"refusal"`
		ToolCalls    []json.RawMessage `json:"tool_calls"`
		FunctionCall json.RawMessage   `json:"function_call"`
	} `json:"message"`
}

// checkedResponseOutputText never returns partial text together with an error.
// Some compatible providers (and older saved fixtures) omit completion metadata;
// that legacy shape remains supported, without claiming its completion is proven.
// Explicit failure, refusal, pending or truncation metadata always takes priority.
func checkedResponseOutputText(body []byte) (string, error) {
	var response struct {
		Status            json.RawMessage `json:"status"`
		Error             json.RawMessage `json:"error"`
		IncompleteDetails json.RawMessage `json:"incomplete_details"`
		OutputText        string          `json:"output_text"`
		Refusal           json.RawMessage `json:"refusal"`
		Output            []struct {
			Type    string          `json:"type"`
			Status  json.RawMessage `json:"status"`
			Content []modelTextPart `json:"content"`
		} `json:"output"`
		Choices json.RawMessage `json:"choices"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		// Do not echo provider bodies, JSON contents or refusal text into diagnostics.
		return "", fmt.Errorf("%w：响应 JSON 无法解析", errInvalidModelOutput)
	}
	if nonNullModelValue(response.Error) {
		return "", fmt.Errorf("%w：服务端返回错误状态", errInvalidModelOutput)
	}
	if nonNullModelValue(response.IncompleteDetails) {
		var details struct {
			Reason string `json:"reason"`
		}
		_ = json.Unmarshal(response.IncompleteDetails, &details)
		if details.Reason == "max_output_tokens" || details.Reason == "length" {
			return "", fmt.Errorf("%w：输出达到长度上限", errIncompleteModelOutput)
		}
		if details.Reason == "content_filter" {
			return "", fmt.Errorf("%w：内容被过滤", errIncompleteModelOutput)
		}
		return "", errIncompleteModelOutput
	}
	if err := checkedModelCompletionStatus(response.Status); err != nil {
		return "", err
	}
	if nonEmptyModelRefusal(response.Refusal) {
		return "", fmt.Errorf("%w：模型拒绝生成请求内容", errInvalidModelOutput)
	}
	if len(response.Choices) > 0 {
		var choices []modelTextChoice
		if err := json.Unmarshal(response.Choices, &choices); err != nil || len(choices) == 0 {
			return "", fmt.Errorf("%w：没有可用的模型候选结果", errInvalidModelOutput)
		}
		// A completion is one alternative, not a concatenation of alternatives.
		// Prefer index zero, otherwise consistently use the first returned choice.
		selected := choices[0]
		for _, choice := range choices {
			if choice.Index != nil && *choice.Index == 0 {
				selected = choice
				break
			}
		}
		return checkedChoiceText(selected)
	}
	var parts []string
	for _, item := range response.Output {
		if err := checkedModelCompletionStatus(item.Status); err != nil {
			return "", err
		}
		switch item.Type {
		case "function_call", "custom_tool_call", "tool_call":
			return "", fmt.Errorf("%w：模型仍要求执行工具，不能作为最终报告", errIncompleteModelOutput)
		case "refusal":
			return "", fmt.Errorf("%w：模型拒绝生成请求内容", errInvalidModelOutput)
		}
		text, err := checkedModelTextParts(item.Content)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(text) != "" {
			parts = append(parts, text)
		}
	}
	// Validate typed output before accepting the convenience output_text field;
	// otherwise it could mask an incomplete message or a refusal.
	if strings.TrimSpace(response.OutputText) != "" {
		return response.OutputText, nil
	}
	return requireModelText(strings.Join(parts, "\n"))
}

func checkedChoiceText(choice modelTextChoice) (string, error) {
	if len(choice.FinishReason) > 0 {
		var reason string
		if json.Unmarshal(choice.FinishReason, &reason) != nil || strings.TrimSpace(reason) == "" {
			return "", fmt.Errorf("%w：模型未确认生成已结束", errIncompleteModelOutput)
		}
		switch reason {
		case "stop":
		case "length":
			return "", fmt.Errorf("%w：输出达到长度上限", errIncompleteModelOutput)
		case "content_filter":
			return "", fmt.Errorf("%w：内容被过滤", errIncompleteModelOutput)
		case "tool_calls", "function_call":
			return "", fmt.Errorf("%w：模型仍要求执行工具，不能作为最终报告", errIncompleteModelOutput)
		default:
			return "", fmt.Errorf("%w：模型未正常结束", errIncompleteModelOutput)
		}
	}
	if nonEmptyModelRefusal(choice.Message.Refusal) {
		return "", fmt.Errorf("%w：模型拒绝生成请求内容", errInvalidModelOutput)
	}
	if len(choice.Message.ToolCalls) > 0 || nonNullModelValue(choice.Message.FunctionCall) {
		return "", fmt.Errorf("%w：模型仍要求执行工具，不能作为最终报告", errIncompleteModelOutput)
	}
	var text string
	if json.Unmarshal(choice.Message.Content, &text) == nil {
		return requireModelText(text)
	}
	var parts []modelTextPart
	if json.Unmarshal(choice.Message.Content, &parts) != nil {
		return "", fmt.Errorf("%w：消息内容格式无效", errInvalidModelOutput)
	}
	text, err := checkedModelTextParts(parts)
	if err != nil {
		return "", err
	}
	return requireModelText(text)
}

func checkedModelCompletionStatus(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil // Legacy compatible response: completion metadata unavailable.
	}
	var status string
	if json.Unmarshal(raw, &status) != nil {
		return fmt.Errorf("%w：生成状态格式无效", errInvalidModelOutput)
	}
	switch status {
	case "completed":
		return nil
	case "failed":
		return fmt.Errorf("%w：模型生成失败", errInvalidModelOutput)
	case "cancelled", "canceled":
		return fmt.Errorf("%w：模型生成已取消", errIncompleteModelOutput)
	case "queued", "in_progress":
		return fmt.Errorf("%w：模型仍在等待或生成中", errIncompleteModelOutput)
	default:
		return errIncompleteModelOutput
	}
}

func checkedModelTextParts(parts []modelTextPart) (string, error) {
	var texts []string
	for _, part := range parts {
		if part.Type == "refusal" || strings.TrimSpace(part.Refusal) != "" {
			return "", fmt.Errorf("%w：模型拒绝生成请求内容", errInvalidModelOutput)
		}
		if (part.Type == "text" || part.Type == "output_text") && strings.TrimSpace(part.Text) != "" {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, "\n"), nil
}

func requireModelText(text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("%w：没有可解析文本", errInvalidModelOutput)
	}
	return text, nil
}

func nonNullModelValue(raw json.RawMessage) bool {
	value := strings.TrimSpace(string(raw))
	return value != "" && value != "null"
}

func nonEmptyModelRefusal(raw json.RawMessage) bool {
	if !nonNullModelValue(raw) {
		return false
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return strings.TrimSpace(value) != ""
	}
	return true
}
