package kimi

import (
	"fmt"
	"os"
	"strings"

	"github.com/wonli/apic/v2"
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
		key = strings.TrimSpace(os.Getenv("MOONSHOT_API_KEY"))
	}
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
