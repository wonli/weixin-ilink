package ai

import (
	"strings"
	"testing"
	"time"
)

func TestBuildSystemPromptIncludesDateAndExtraRules(t *testing.T) {
	now := time.Date(2026, 4, 2, 15, 4, 5, 0, time.FixedZone("CST", 8*3600))
	got := buildSystemPrompt(Config{
		SystemPrompt: "请控制在两段以内。",
	}, PromptContext{
		Now:                 now,
		AccountID:           "bot-123",
		BotUserID:           "wx-bot-owner",
		BaseURL:             "https://example.test",
		ContactUserID:       "user-456",
		InboundMessageKinds: []string{"文本", "图片"},
		ConversationHistory: []HistoryTurn{
			{Role: "user", Text: "历史上的今天", At: now.Add(-2 * time.Minute)},
		},
	})

	if !strings.Contains(got, "系统当前日期: 2026-04-02") {
		t.Fatalf("expected current date in prompt, got %q", got)
	}
	if !strings.Contains(got, "输出必须是纯文本") {
		t.Fatalf("expected plain text rule in prompt, got %q", got)
	}
	if !strings.Contains(got, "当前会话联系人 ID: user-456") {
		t.Fatalf("expected runtime context in prompt, got %q", got)
	}
	if !strings.Contains(got, "当前用户消息类型: 文本、图片") {
		t.Fatalf("expected inbound message kinds in prompt, got %q", got)
	}
	if !strings.Contains(got, "最近对话上下文:") {
		t.Fatalf("expected conversation history in prompt, got %q", got)
	}
	if !strings.Contains(got, "请控制在两段以内。") {
		t.Fatalf("expected extra config prompt in prompt, got %q", got)
	}
	if !strings.Contains(got, "多模态输入处理要求") {
		t.Fatalf("expected media guidance in prompt, got %q", got)
	}
}

func TestNormalizePlainTextRemovesMarkdownMarkers(t *testing.T) {
	input := "# 标题\n\n- **今天是** 2026-04-02\n\n```txt\n测试\n```"
	got := normalizePlainText(input)

	if strings.Contains(got, "#") || strings.Contains(got, "**") || strings.Contains(got, "```") {
		t.Fatalf("expected markdown markers removed, got %q", got)
	}
	if !strings.Contains(got, "今天是 2026-04-02") {
		t.Fatalf("expected content preserved, got %q", got)
	}
}
