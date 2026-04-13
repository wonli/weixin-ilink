package ilinkapi

import (
	"context"
	"fmt"
	"time"

	"github.com/wonli/apic/v2"
)

// getUploadURLAPI 用于申请媒体上传地址。
// 图片、视频、文件在真正上传到 CDN 之前，都会先通过它换取上传参数或完整 URL。
type getUploadURLAPI struct {
	*Client
	baseURL string
	body    getUploadURLBody
}

func (a *getUploadURLAPI) Url() string                 { return a.baseURL }
func (a *getUploadURLAPI) Path() string                { return "/ilink/bot/getuploadurl" }
func (a *getUploadURLAPI) HttpMethod() apic.HttpMethod { return apic.POST }
func (a *getUploadURLAPI) PostBody() any               { return a.body }

func newGetUploadURLAPI(client *Client, baseURL string, req GetUploadURLRequest) *getUploadURLAPI {
	return &getUploadURLAPI{
		Client:  client,
		baseURL: stringsTrimRightSlash(baseURL),
		body: getUploadURLBody{
			BaseInfo:            BaseInfo{ChannelVersion: ChannelVersion},
			GetUploadURLRequest: req,
		},
	}
}

func (a *getUploadURLAPI) GetUploadURL(ctx context.Context, token string) (*GetUploadURLResponse, error) {
	headers, err := buildHeaders(token)
	if err != nil {
		return nil, err
	}
	var out GetUploadURLResponse
	if err = a.callJSON(ctx, &apic.ApiId{
		Name:   "getuploadurl",
		Client: a,
	}, &out, &apic.Options{
		Headers: headers,
		Timeout: 15 * time.Second,
	}); err != nil {
		return nil, err
	}
	if out.Ret != 0 || out.ErrCode != 0 {
		return nil, fmt.Errorf("getuploadurl failed: ret=%d errcode=%d errmsg=%s", out.Ret, out.ErrCode, out.ErrMsg)
	}
	return &out, nil
}
