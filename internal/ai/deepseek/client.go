package deepseek

import (
	"fmt"
	"github.com/wonli/apic/v2"
	"os"
	"strings"
)

type rpcHandler func(msg []byte) error

type Api interface {
	StartChat(model, apiKey, systemPrompt, userPrompt string, handler rpcHandler, onDelta func(string)) error

	getGlobalParams(apiKey string) apic.Params
}

type Client struct {
}

func NewClient() Api {
	return &Client{}
}

func (c *Client) StartChat(model, apiKey, systemPrompt, userPrompt string, handler rpcHandler, onDelta func(string)) error {
	return (&apiChat{
		client:  c,
		apiKey:  apiKey,
		onDelta: onDelta,
	}).StartChat(model, systemPrompt, userPrompt, handler)
}

func (c *Client) getGlobalParams(apiKey string) apic.Params {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		// 兼容老环境：没有传入就退回环境变量，但不再 panic
		key = strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	}

	// 没有 key 就只返回 Content-Type，交给服务端报 401，而不是直接崩溃
	if key == "" {
		return apic.Params{
			"Content-Type": "application/json",
		}
	}
	return apic.Params{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", key),
	}
}
