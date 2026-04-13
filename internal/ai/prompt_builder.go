package ai

import (
	"fmt"
	"strings"
	"time"
)

type PromptContext struct {
	Now                 time.Time
	AccountID           string
	BotUserID           string
	BaseURL             string
	ContactUserID       string
	InboundMessageKinds []string
	ConversationHistory []HistoryTurn
}

type HistoryTurn struct {
	Role string
	Text string
	At   time.Time
}

func buildSystemPrompt(cfg Config, ctx PromptContext) string {
	now := ctx.Now
	if now.IsZero() {
		now = time.Now()
	}
	parts := []string{
		strings.TrimSpace(chatSystemPrompt),
		"系统当前时间: " + now.In(time.Local).Format("2006-01-02 15:04:05 MST"),
		"系统当前日期: " + now.In(time.Local).Format("2006-01-02"),
	}

	runtimeLines := []string{
		"当前运行上下文:",
		"- 渠道: 微信聊天",
		"- 回复形式: 单条微信消息气泡",
	}
	if v := strings.TrimSpace(ctx.AccountID); v != "" {
		runtimeLines = append(runtimeLines, "- 当前 Bot 账号 ID: "+v)
	}
	if v := strings.TrimSpace(ctx.BotUserID); v != "" {
		runtimeLines = append(runtimeLines, "- 当前 Bot 关联用户 ID: "+v)
	}
	if v := strings.TrimSpace(ctx.BaseURL); v != "" {
		runtimeLines = append(runtimeLines, "- 当前服务地址: "+v)
	}
	if v := strings.TrimSpace(ctx.ContactUserID); v != "" {
		runtimeLines = append(runtimeLines, "- 当前会话联系人 ID: "+v)
	}
	if len(ctx.InboundMessageKinds) > 0 {
		runtimeLines = append(runtimeLines, "- 当前用户消息类型: "+strings.Join(ctx.InboundMessageKinds, "、"))
	}
	parts = append(parts, strings.Join(runtimeLines, "\n"))

	parts = append(parts, strings.TrimSpace(`多模态输入处理要求:
- 如果本轮消息包含语音转写，请优先根据转写文本理解用户意图。
- 如果本轮消息包含图片、视频或文件，但系统没有提供可解析内容，不要编造附件内容。
- 当回答必须依赖附件内容而当前上下文不足时，先明确说明你收到了对应类型的附件，再请用户补充描述或重新发送可识别文本。`))

	if history := formatConversationHistory(ctx.ConversationHistory); history != "" {
		parts = append(parts, history)
	}

	if extra := strings.TrimSpace(cfg.SystemPrompt); extra != "" {
		parts = append(parts, "附加产品要求:\n"+extra)
	}

	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func formatConversationHistory(turns []HistoryTurn) string {
	if len(turns) == 0 {
		return ""
	}

	lines := []string{
		"最近对话上下文:",
	}
	for i, turn := range turns {
		role := strings.TrimSpace(turn.Role)
		if role == "" {
			role = "unknown"
		}
		text := strings.TrimSpace(turn.Text)
		if text == "" {
			continue
		}
		text = strings.ReplaceAll(text, "\n", " ")
		if len([]rune(text)) > 160 {
			runes := []rune(text)
			text = string(runes[:160]) + "..."
		}
		at := ""
		if !turn.At.IsZero() {
			at = turn.At.In(time.Local).Format("2006-01-02 15:04")
		}
		if at != "" {
			lines = append(lines, fmt.Sprintf("%d. [%s @ %s] %s", i+1, role, at, text))
			continue
		}
		lines = append(lines, fmt.Sprintf("%d. [%s] %s", i+1, role, text))
	}
	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func normalizePlainText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	lines := strings.Split(s, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "# ")
		line = strings.TrimPrefix(line, "## ")
		line = strings.TrimPrefix(line, "### ")
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "> ")
		line = strings.TrimPrefix(line, "`")
		line = strings.TrimSuffix(line, "`")
		lines[i] = line
	}

	s = strings.Join(lines, "\n")
	replacer := strings.NewReplacer(
		"**", "",
		"__", "",
		"```", "",
		"`", "",
	)
	s = replacer.Replace(s)

	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(s)
}
