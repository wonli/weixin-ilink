package ilinkapi

import (
	"context"
	"net/url"

	"github.com/wonli/apic/v2"
)

// getQRCodeStatusAPI 用于轮询登录二维码状态。
// 它负责判断二维码是否已扫码、已确认、已失效，或者仍需继续等待。
type getQRCodeStatusAPI struct {
	*Client
	baseURL string
	qrcode  string
}

func (a *getQRCodeStatusAPI) Url() string                 { return a.baseURL }
func (a *getQRCodeStatusAPI) Path() string                { return "/ilink/bot/get_qrcode_status" }
func (a *getQRCodeStatusAPI) HttpMethod() apic.HttpMethod { return apic.GET }
func (a *getQRCodeStatusAPI) Query() url.Values {
	return url.Values{"qrcode": []string{a.qrcode}}
}

func newGetQRCodeStatusAPI(client *Client, baseURL, qrcode string) *getQRCodeStatusAPI {
	return &getQRCodeStatusAPI{
		Client:  client,
		baseURL: stringsTrimRightSlash(baseURL),
		qrcode:  qrcode,
	}
}

func (a *getQRCodeStatusAPI) PollLogin(ctx context.Context) (*QRStatusResponse, error) {
	headers, err := buildHeaders("")
	if err != nil {
		return nil, err
	}
	headers["iLink-App-ClientVersion"] = "1"
	var out QRStatusResponse
	err = a.callJSON(ctx, &apic.ApiId{
		Name:   "get_qrcode_status",
		Client: a,
	}, &out, &apic.Options{
		Headers: headers,
		Timeout: DefaultLoginTimeout,
	})
	return &out, err
}
