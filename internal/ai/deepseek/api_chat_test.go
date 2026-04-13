package deepseek

import "testing"

func TestReceiveEventPreservesContentWhitespace(t *testing.T) {
	var got string
	chat := &apiChat{
		onDelta: func(delta string) {
			got += delta
		},
	}

	chat.ReceiveEvent(`data: {"choices":[{"delta":{"content":"SELECT "}}]}`)
	chat.ReceiveEvent(`data: {"choices":[{"delta":{"content":"* FROM orders"}}]}`)

	if got != "SELECT * FROM orders" {
		t.Fatalf("expected streamed content to preserve spaces, got %q", got)
	}
	if chat.builder.String() != "SELECT * FROM orders" {
		t.Fatalf("expected builder to preserve spaces, got %q", chat.builder.String())
	}
}

func TestReceiveEventPreservesReasoningWhitespace(t *testing.T) {
	var got string
	chat := &apiChat{
		onDelta: func(delta string) {
			got += delta
		},
	}

	chat.ReceiveEvent(`data: {"choices":[{"delta":{"reasoning_content":"先分析 "}}]}`)
	chat.ReceiveEvent(`data: {"choices":[{"delta":{"reasoning_content":"再输出"}}]}`)

	if got != "先分析 再输出" {
		t.Fatalf("expected reasoning content to preserve spaces, got %q", got)
	}
	if chat.builder.String() != "" {
		t.Fatalf("expected builder to ignore reasoning content, got %q", chat.builder.String())
	}
}

func TestReceiveEventAggregatesOnlyContent(t *testing.T) {
	chat := &apiChat{}

	chat.ReceiveEvent(`data: {"choices":[{"delta":{"reasoning_content":"先分析 "}}]}`)
	chat.ReceiveEvent(`data: {"choices":[{"delta":{"content":"{\"sql\":\"SELECT "}}]}`)
	chat.ReceiveEvent(`data: {"choices":[{"delta":{"content":"1\"}"}}]}`)

	if chat.builder.String() != "{\"sql\":\"SELECT 1\"}" {
		t.Fatalf("expected builder to preserve output, got %q", chat.builder.String())
	}
}
