package ilinkapi

import (
	"context"
)

// StartLogin 拉起网页登录二维码，用于新的微信会话登录流程。
func (c *Client) StartLogin(ctx context.Context, baseURL string) (*QRCodeResponse, error) {
	return newGetBotQRCodeAPI(c, baseURL).StartLogin(ctx)
}

// PollLogin 轮询二维码状态，直到确认登录、失效或超时。
func (c *Client) PollLogin(ctx context.Context, baseURL, qrcode string) (*QRStatusResponse, error) {
	return newGetQRCodeStatusAPI(c, baseURL, qrcode).PollLogin(ctx)
}
