package deepseek

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/wonli/apic/v2"
)

// 轻量封装：使用 apic/v2 走 DeepSeek 的 HTTP + 流式接口。

type apiChat struct {
	msg *localMsg
	apic.Apic

	client  *Client
	apiKey  string
	builder strings.Builder    // 聚合完整内容
	onDelta func(delta string) // 每个增量内容回调（用于 UI 流式展示）
}

func (a *apiChat) Url() string {
	return "https://api.deepseek.com/chat/completions"
}

func (a *apiChat) PostBody() any {
	return a.msg
}

func (a *apiChat) HttpMethod() apic.HttpMethod {
	return http.MethodPost
}

func (a *apiChat) Headers() apic.Params {
	return a.client.getGlobalParams(a.apiKey)
}

func (a *apiChat) Debug() bool {
	return true
}

// ReceiveEvent 由 apic 在 Stream=true 时按行回调，参数是整行文本（已 TrimSpace）。
// DeepSeek 的流式响应是 SSE: 每行 "data: {...}" 或 "data: [DONE]".
func (a *apiChat) ReceiveEvent(line string) {
	if !strings.HasPrefix(line, "data:") {
		return
	}
	data := strings.TrimPrefix(line, "data:")
	if strings.HasPrefix(data, " ") {
		data = data[1:]
	}
	if data == "" || data == "[DONE]" {
		return
	}

	// 解析 DeepSeek 流式增量，兼容 content 和 reasoning_content
	type streamChunk struct {
		Choices []struct {
			Delta struct {
				Content          *string `json:"content"`
				ReasoningContent *string `json:"reasoning_content"`
			} `json:"delta"`
		} `json:"choices"`
	}

	var chunk streamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return
	}
	if len(chunk.Choices) == 0 {
		return
	}
	d := chunk.Choices[0].Delta

	if d.Content != nil && *d.Content != "" {
		a.builder.WriteString(*d.Content)
	}
	if a.onDelta != nil {
		if d.ReasoningContent != nil && *d.ReasoningContent != "" {
			a.onDelta(*d.ReasoningContent)
		} else if d.Content != nil && *d.Content != "" {
			a.onDelta(*d.Content)
		}
	}
}

// StartChat 由上层传入已经拼好的 prompt 文本
func (a *apiChat) StartChat(model, systemPrompt, userPrompt string, handler func(msg []byte) error) error {
	a.msg = &localMsg{
		Model: model,
		Messages: []localMsgItem{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Think:       false,
		Stream:      true,
		Temperature: 0.2,
	}
	a.builder.Reset()

	id := apic.ApiId{
		Name:   "chat",
		Client: a,
		Stream: true,
	}

	err := apic.Init().Call(&id, &apic.Options{
		Debug: true,
	})
	if err != nil {
		return err
	}

	// 流式场景下，apic 不会自动填充 Response.Data，这里用聚合后的 builder 作为完整结果。
	if id.Response != nil && id.Response.HttpStatus != 0 && id.Response.HttpStatus != http.StatusOK {
		return fmt.Errorf("返回状态错误:%d", id.Response.HttpStatus)
	}

	return handler([]byte(a.builder.String()))
}

type localMsg struct {
	Model       string         `json:"model"`
	Messages    []localMsgItem `json:"messages"`
	Think       bool           `json:"think"`
	Stream      bool           `json:"stream"`
	Temperature float64        `json:"temperature"`
}

type localMsgItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
