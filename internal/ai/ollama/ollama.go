package ollama

type rpcHandler func(msg []byte) error

type Api interface {
	StartChat(baseURL, model, systemPrompt, userPrompt string, handler rpcHandler, onDelta func(string)) error
}

type Client struct {
}

func NewClient() Api {
	return &Client{}
}

func (c *Client) StartChat(baseURL, model, systemPrompt, userPrompt string, handler rpcHandler, onDelta func(string)) error {
	return (&apiChat{baseURL: baseURL, onDelta: onDelta}).StartChat(model, systemPrompt, userPrompt, handler)
}
