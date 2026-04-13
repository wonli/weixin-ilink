package ollama

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/wonli/apic/v2"
)

type apiChat struct {
	msg *localMsg
	apic.Apic
	baseURL        string
	contentBuilder strings.Builder
	onDelta        func(string)
}

func (a *apiChat) Url() string {
	return strings.TrimRight(strings.TrimSpace(a.baseURL), "/") + "/api/chat"
}

func (a *apiChat) PostBody() any {
	return a.msg
}

func (a *apiChat) HttpMethod() apic.HttpMethod {
	return http.MethodPost
}

func (a *apiChat) Debug() bool {
	return true
}

func (a *apiChat) ReceiveEvent(line string) {
	if line == "" {
		return
	}

	type responseChunk struct {
		Message struct {
			Content  string `json:"content"`
			Thinking string `json:"thinking"`
		} `json:"message"`
		Error string `json:"error"`
		Done  bool   `json:"done"`
	}

	var out responseChunk
	if err := json.Unmarshal([]byte(line), &out); err != nil {
		return
	}
	if strings.TrimSpace(out.Error) != "" {
		return
	}
	if out.Message.Content != "" {
		a.contentBuilder.WriteString(out.Message.Content)
	}
	if a.onDelta != nil {
		if out.Message.Thinking != "" {
			a.onDelta(out.Message.Thinking)
		} else if out.Message.Content != "" {
			a.onDelta(out.Message.Content)
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
		Think:  true,
		Stream: true,
	}
	a.contentBuilder.Reset()

	id := apic.ApiId{
		Name:   "chat",
		Client: a,
		Stream: true,
	}

	err := apic.Init().Call(&id, &apic.Options{
		Debug:   true,
		Timeout: 10 * time.Minute,
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
	Model    string         `json:"model"`
	Messages []localMsgItem `json:"messages"`
	Think    bool           `json:"think"`
	Stream   bool           `json:"stream"`
}

type localMsgItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
