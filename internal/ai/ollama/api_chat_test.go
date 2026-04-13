package ollama

import "testing"

func TestAPIChatURLUsesConfiguredBaseURL(t *testing.T) {
	chat := &apiChat{baseURL: "http://192.168.1.10:11434/"}
	if got := chat.Url(); got != "http://192.168.1.10:11434/api/chat" {
		t.Fatalf("unexpected url: %s", got)
	}
}

func TestReceiveEventStreamsThinkingAndAggregatesContent(t *testing.T) {
	var got string
	chat := &apiChat{
		onDelta: func(delta string) {
			got += delta
		},
	}

	chat.ReceiveEvent(`{"message":{"content":"","thinking":"先想"}}`)
	chat.ReceiveEvent(`{"message":{"content":"{\"sql\":\"SELECT "}}`)
	chat.ReceiveEvent(`{"message":{"content":"1\"}"}}`)

	if got != "先想{\"sql\":\"SELECT 1\"}" {
		t.Fatalf("unexpected streamed output: %q", got)
	}
	if chat.contentBuilder.String() != "{\"sql\":\"SELECT 1\"}" {
		t.Fatalf("unexpected content builder: %q", chat.contentBuilder.String())
	}
}
