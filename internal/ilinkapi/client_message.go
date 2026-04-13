package ilinkapi

import (
	"context"
	"errors"
	"strings"
)

func containsContextDeadlineExceeded(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return strings.Contains(err.Error(), "context deadline exceeded")
}

// GetUpdates 长轮询消息更新。
// 当服务端仅因轮询超时返回 context deadline exceeded 时，按“无新消息”处理。
func (c *Client) GetUpdates(ctx context.Context, baseURL, token, buf string) (*GetUpdatesResponse, error) {
	return newGetUpdatesAPI(c, baseURL, buf).GetUpdates(ctx, token, buf)
}

// SendText 构造一个最常用的文本消息请求。
func (c *Client) SendText(ctx context.Context, baseURL, token, toUserID, contextToken, text string) error {
	return newTextSendMessageAPI(c, baseURL, toUserID, contextToken, text).SendMessage(ctx, token)
}

// SendMessage 发送一个完整的微信消息结构。
func (c *Client) SendMessage(ctx context.Context, baseURL, token string, req SendMessageRequest) error {
	return newSendMessageAPI(c, baseURL, req).SendMessage(ctx, token)
}

// GetUploadURL 获取媒体上传到 CDN 前所需的上传参数。
func (c *Client) GetUploadURL(ctx context.Context, baseURL, token string, req GetUploadURLRequest) (*GetUploadURLResponse, error) {
	return newGetUploadURLAPI(c, baseURL, req).GetUploadURL(ctx, token)
}

// GetConfig 获取当前会话上下文下的配置数据。
func (c *Client) GetConfig(ctx context.Context, baseURL, token, ilinkUserID, contextToken string) (*GetConfigResponse, error) {
	return newGetConfigAPI(c, baseURL, ilinkUserID, contextToken).GetConfig(ctx, token)
}

// SendTyping 向对端同步机器人“正在输入”状态。
func (c *Client) SendTyping(ctx context.Context, baseURL, token, ilinkUserID, typingTicket string, status int) error {
	return newSendTypingAPI(c, baseURL, ilinkUserID, typingTicket, status).SendTyping(ctx, token)
}

// SendImage 发送已经上传完成的图片媒体。
func (c *Client) SendImage(ctx context.Context, baseURL, token, toUserID, contextToken string, uploaded *UploadedMedia) error {
	api, err := newImageSendMessageAPI(c, baseURL, toUserID, contextToken, uploaded)
	if err != nil {
		return err
	}
	return api.SendMessage(ctx, token)
}

// SendVideo 发送已经上传完成的视频媒体。
func (c *Client) SendVideo(ctx context.Context, baseURL, token, toUserID, contextToken string, uploaded *UploadedMedia) error {
	api, err := newVideoSendMessageAPI(c, baseURL, toUserID, contextToken, uploaded)
	if err != nil {
		return err
	}
	return api.SendMessage(ctx, token)
}

// SendFile 发送已经上传完成的文件媒体。
func (c *Client) SendFile(ctx context.Context, baseURL, token, toUserID, contextToken, fileName string, uploaded *UploadedMedia) error {
	api, err := newFileSendMessageAPI(c, baseURL, toUserID, contextToken, fileName, uploaded)
	if err != nil {
		return err
	}
	return api.SendMessage(ctx, token)
}
