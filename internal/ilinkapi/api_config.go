package ilinkapi

import (
	"context"
	"fmt"
	"time"

	"github.com/wonli/apic/v2"
)

// getConfigAPI 用于拉取当前会话的配置数据。
// 常见用途是获取输入状态、上下文配置等对话相关附加信息。
type getConfigAPI struct {
	*Client
	baseURL string
	body    getConfigBody
}

func (a *getConfigAPI) Url() string                 { return a.baseURL }
func (a *getConfigAPI) Path() string                { return "/ilink/bot/getconfig" }
func (a *getConfigAPI) HttpMethod() apic.HttpMethod { return apic.POST }
func (a *getConfigAPI) PostBody() any               { return a.body }

func newGetConfigAPI(client *Client, baseURL, ilinkUserID, contextToken string) *getConfigAPI {
	return &getConfigAPI{
		Client:  client,
		baseURL: stringsTrimRightSlash(baseURL),
		body: getConfigBody{
			BaseInfo: BaseInfo{ChannelVersion: ChannelVersion},
			GetConfigRequest: GetConfigRequest{
				ILinkUserID:  ilinkUserID,
				ContextToken: contextToken,
			},
		},
	}
}

func (a *getConfigAPI) GetConfig(ctx context.Context, token string) (*GetConfigResponse, error) {
	headers, err := buildHeaders(token)
	if err != nil {
		return nil, err
	}
	var out GetConfigResponse
	if err := a.callJSON(ctx, &apic.ApiId{
		Name:   "getconfig",
		Client: a,
	}, &out, &apic.Options{
		Headers: headers,
		Timeout: 10 * time.Second,
	}); err != nil {
		return nil, err
	}
	if out.Ret != 0 || out.ErrCode != 0 {
		return nil, fmt.Errorf("getconfig failed: ret=%d errcode=%d errmsg=%s", out.Ret, out.ErrCode, out.ErrMsg)
	}
	return &out, nil
}
