package ilinkapi

import (
	"context"
	"net/url"
	"time"

	"github.com/wonli/apic/v2"
)

// getBotQRCodeAPI 用于申请登录二维码。
// 调用后通常会拿到一个二维码，再配合轮询接口等待扫码登录结果。
type getBotQRCodeAPI struct {
	*Client
	baseURL string
}

func (a *getBotQRCodeAPI) Url() string                 { return a.baseURL }
func (a *getBotQRCodeAPI) Path() string                { return "/ilink/bot/get_bot_qrcode" }
func (a *getBotQRCodeAPI) HttpMethod() apic.HttpMethod { return apic.GET }
func (a *getBotQRCodeAPI) Query() url.Values {
	return url.Values{"bot_type": []string{DefaultBotType}}
}

func newGetBotQRCodeAPI(client *Client, baseURL string) *getBotQRCodeAPI {
	return &getBotQRCodeAPI{
		Client:  client,
		baseURL: stringsTrimRightSlash(baseURL),
	}
}

func (a *getBotQRCodeAPI) StartLogin(ctx context.Context) (*QRCodeResponse, error) {
	headers, err := buildHeaders("")
	if err != nil {
		return nil, err
	}
	var out QRCodeResponse
	err = a.callJSON(ctx, &apic.ApiId{
		Name:   "get_bot_qrcode",
		Client: a,
	}, &out, &apic.Options{
		Headers: headers,
		Timeout: 15 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
