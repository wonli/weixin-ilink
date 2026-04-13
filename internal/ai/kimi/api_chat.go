package kimi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/wonli/apic/v2"
)

// 轻量封装：只负责把已经构造好的 system/user 文本走 apic 发给 Kimi

type apiChat struct {
	msg *localMsg
	apic.Apic

	client         *Client
	apiKey         string
	contentBuilder strings.Builder
	onDelta        func(delta string)
}

func (a *apiChat) Url() string {
	return "https://api.moonshot.cn/v1/chat/completions"
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
	delta := chunk.Choices[0].Delta
	if delta.Content != nil {
		a.contentBuilder.WriteString(*delta.Content)
	}
	if a.onDelta != nil {
		if delta.ReasoningContent != nil {
			a.onDelta(*delta.ReasoningContent)
		} else if delta.Content != nil {
			a.onDelta(*delta.Content)
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
		Think:       true,
		Stream:      true,
		Temperature: 1,
	}
	a.contentBuilder.Reset()

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

	if id.Response != nil && id.Response.HttpStatus != 0 && id.Response.HttpStatus != http.StatusOK {
		return fmt.Errorf("返回状态错误:%d", id.Response.HttpStatus)
	}

	return handler([]byte(a.contentBuilder.String()))
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
