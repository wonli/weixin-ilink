package ilinkapi

import (
	"context"

	"github.com/wonli/apic/v2"
)

// getUpdatesAPI 用于长轮询拉取新消息和事件更新。
// 机器人主循环通常依赖它持续获取用户消息、状态变化和新的游标。
type getUpdatesAPI struct {
	*Client
	baseURL string
	body    getUpdatesBody
}

func (a *getUpdatesAPI) Url() string                 { return a.baseURL }
func (a *getUpdatesAPI) Path() string                { return "/ilink/bot/getupdates" }
func (a *getUpdatesAPI) HttpMethod() apic.HttpMethod { return apic.POST }
func (a *getUpdatesAPI) PostBody() any               { return a.body }

func newGetUpdatesAPI(client *Client, baseURL, buf string) *getUpdatesAPI {
	return &getUpdatesAPI{
		Client:  client,
		baseURL: stringsTrimRightSlash(baseURL),
		body: getUpdatesBody{
			BaseInfo: BaseInfo{ChannelVersion: ChannelVersion},
			GetUpdatesRequest: GetUpdatesRequest{
				GetUpdatesBuf: buf,
			},
		},
	}
}

func (a *getUpdatesAPI) GetUpdates(ctx context.Context, token, buf string) (*GetUpdatesResponse, error) {
	headers, err := buildHeaders(token)
	if err != nil {
		return nil, err
	}
	var out GetUpdatesResponse
	err = a.callJSON(ctx, &apic.ApiId{
		Name:   "getupdates",
		Client: a,
	}, &out, &apic.Options{
		Headers: headers,
		Timeout: DefaultPollTimeout,
	})
	if err != nil && containsContextDeadlineExceeded(err) {
		return &GetUpdatesResponse{Ret: 0, GetUpdatesBuf: buf}, nil
	}
	return &out, err
}
