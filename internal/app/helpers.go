package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/wonli/weixin-ilink/internal/chatstate"
	"github.com/wonli/weixin-ilink/internal/ilinkapi"
	"github.com/wonli/weixin-ilink/internal/session"
)

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func restoreContacts(state *chatstate.State, record session.AccountRecord) {
	contacts := make([]chatstate.PersistedContact, 0, len(record.Contacts))
	for _, item := range record.Contacts {
		var parsed time.Time
		if item.LastMessageAt != "" {
			if t, err := time.Parse(time.RFC3339Nano, item.LastMessageAt); err == nil {
				parsed = t
			}
		}
		contacts = append(contacts, chatstate.PersistedContact{
			UserID:        item.UserID,
			RemarkName:    item.RemarkName,
			ContextToken:  item.ContextToken,
			LastMessageAt: parsed,
		})
	}
	state.RestoreContacts(contacts, record.SelectedUserID)
}

func isBenignCancel(err error, ctx context.Context) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context canceled")
}

type inboundMessage struct {
	PlainText   string
	HistoryText string
	UserPrompt  string
	HasContent  bool
}

func extractInboundMessage(msg ilinkapi.WeixinMessage) inboundMessage {
	var textParts []string
	var historyParts []string
	var promptParts []string
	var imageCount int
	var videoCount int
	var voiceCount int
	var fileNames []string

	for _, item := range msg.ItemList {
		switch item.Type {
		case ilinkapi.MessageItemTypeText:
			if item.TextItem == nil {
				continue
			}
			text := strings.TrimSpace(item.TextItem.Text)
			if text == "" {
				continue
			}
			textParts = append(textParts, text)
			historyParts = append(historyParts, text)
			promptParts = append(promptParts, "用户发送的文字内容: "+text)
		case ilinkapi.MessageItemTypeVoice:
			voiceCount++
			transcript := ""
			if item.VoiceItem != nil {
				transcript = strings.TrimSpace(item.VoiceItem.Text)
			}
			if transcript != "" {
				historyParts = append(historyParts, "[语音转写] "+transcript)
				promptParts = append(promptParts, "用户发送了一条语音，接口返回的转写内容: "+transcript)
			} else {
				historyParts = append(historyParts, "[语音]")
				promptParts = append(promptParts, "用户发送了一条语音消息，但当前接口没有返回可用转写。")
			}
		case ilinkapi.MessageItemTypeImage:
			imageCount++
			historyParts = append(historyParts, "[图片]")
		case ilinkapi.MessageItemTypeVideo:
			videoCount++
			historyParts = append(historyParts, "[视频]")
		case ilinkapi.MessageItemTypeFile:
			name := "未命名文件"
			if item.FileItem != nil && strings.TrimSpace(item.FileItem.FileName) != "" {
				name = strings.TrimSpace(item.FileItem.FileName)
			}
			fileNames = append(fileNames, name)
			historyParts = append(historyParts, "[文件] "+name)
		}
	}

	if imageCount > 0 {
		promptParts = append(promptParts, fmt.Sprintf("用户本轮还发送了 %d 张图片。当前系统只知道收到了图片，暂时没有做图片视觉识别，不要臆测图片细节；如果回答依赖图片内容，请先请用户补充描述。", imageCount))
	}
	if videoCount > 0 {
		promptParts = append(promptParts, fmt.Sprintf("用户本轮还发送了 %d 个视频。当前系统还不能解析视频内容，不要假设视频细节。", videoCount))
	}
	if len(fileNames) > 0 {
		promptParts = append(promptParts, "用户本轮还发送了文件: "+strings.Join(fileNames, "、"))
	}
	if voiceCount > 0 && len(promptParts) == 0 {
		promptParts = append(promptParts, "用户发送了语音消息。")
	}

	plainText := strings.TrimSpace(strings.Join(textParts, "\n"))
	historyText := strings.TrimSpace(strings.Join(historyParts, "\n"))
	userPrompt := strings.TrimSpace(strings.Join(promptParts, "\n"))
	if historyText == "" {
		historyText = "[无法提取文本内容的消息]"
	}
	if userPrompt == "" {
		userPrompt = "用户发送了一条当前系统无法提取具体文本内容的微信消息，请先说明你收到的是非文本消息，并请对方补充文字。"
	}

	return inboundMessage{
		PlainText:   plainText,
		HistoryText: historyText,
		UserPrompt:  userPrompt,
		HasContent:  historyText != "",
	}
}

func (a *App) captureInboundMedia(accountID string, msg ilinkapi.WeixinMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	for idx, item := range msg.ItemList {
		switch item.Type {
		case ilinkapi.MessageItemTypeImage:
			if item.ImageItem == nil {
				continue
			}
			path, err := a.client.DownloadImageToTemp(ctx, ilinkapi.DefaultCDNBaseURL, item.ImageItem)
			if err != nil {
				log.Printf("weixin inbound media account=%s from=%s type=image index=%d save_error=%v", accountID, msg.FromUserID, idx, err)
				continue
			}
			log.Printf("weixin inbound media account=%s from=%s type=image index=%d saved_path=%s", accountID, msg.FromUserID, idx, path)
		case ilinkapi.MessageItemTypeVoice:
			if item.VoiceItem == nil {
				continue
			}
			path, mediaType, err := a.client.DownloadVoiceToTemp(ctx, ilinkapi.DefaultCDNBaseURL, item.VoiceItem)
			if err != nil {
				log.Printf("weixin inbound media account=%s from=%s type=voice index=%d save_error=%v", accountID, msg.FromUserID, idx, err)
				continue
			}
			log.Printf("weixin inbound media account=%s from=%s type=voice index=%d media_type=%s saved_path=%s", accountID, msg.FromUserID, idx, mediaType, path)
		case ilinkapi.MessageItemTypeFile:
			if item.FileItem == nil {
				continue
			}
			path, err := a.client.DownloadFileToTemp(ctx, ilinkapi.DefaultCDNBaseURL, item.FileItem)
			if err != nil {
				log.Printf("weixin inbound media account=%s from=%s type=file index=%d file_name=%s save_error=%v", accountID, msg.FromUserID, idx, item.FileItem.FileName, err)
				continue
			}
			log.Printf("weixin inbound media account=%s from=%s type=file index=%d file_name=%s saved_path=%s", accountID, msg.FromUserID, idx, item.FileItem.FileName, path)
		case ilinkapi.MessageItemTypeVideo:
			if item.VideoItem == nil {
				continue
			}
			path, err := a.client.DownloadVideoToTemp(ctx, ilinkapi.DefaultCDNBaseURL, item.VideoItem)
			if err != nil {
				log.Printf("weixin inbound media account=%s from=%s type=video index=%d save_error=%v", accountID, msg.FromUserID, idx, err)
				continue
			}
			log.Printf("weixin inbound media account=%s from=%s type=video index=%d saved_path=%s", accountID, msg.FromUserID, idx, path)
		}
	}
}
