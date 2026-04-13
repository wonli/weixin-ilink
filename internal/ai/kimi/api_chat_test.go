package kimi

import "testing"

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
}

func TestReceiveEventAggregatesOnlyContent(t *testing.T) {
	chat := &apiChat{}

	chat.ReceiveEvent(`data: {"choices":[{"delta":{"reasoning_content":"先分析 "}}]}`)
	chat.ReceiveEvent(`data: {"choices":[{"delta":{"content":"{\"sql\":\"SELECT "}}]}`)
	chat.ReceiveEvent(`data: {"choices":[{"delta":{"content":"1\"}"}}]}`)

	if chat.contentBuilder.String() != "{\"sql\":\"SELECT 1\"}" {
		t.Fatalf("expected content builder to preserve output, got %q", chat.contentBuilder.String())
	}
}

func TestReceiveEventStreamsContentWhenReasoningMissing(t *testing.T) {
	var got string
	chat := &apiChat{
		onDelta: func(delta string) {
			got += delta
		},
	}

	chat.ReceiveEvent(`data: {"choices":[{"delta":{"content":"{\"sql\":\"SELECT "}}]}`)
	chat.ReceiveEvent(`data: {"choices":[{"delta":{"content":"1\"}"}}]}`)

	if got != "{\"sql\":\"SELECT 1\"}" {
		t.Fatalf("expected streamed content fallback, got %q", got)
	}
}
