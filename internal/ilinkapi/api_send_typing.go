package ilinkapi

import (
	"context"
	"fmt"
	"time"

	"github.com/wonli/apic/v2"
)

// sendTypingAPI 用于同步“正在输入”状态。
// 一般在机器人开始生成回复时调用，让对端看到输入中的提示。
type sendTypingAPI struct {
	*Client
	baseURL string
	body    sendTypingBody
}

func (a *sendTypingAPI) Url() string                 { return a.baseURL }
func (a *sendTypingAPI) Path() string                { return "/ilink/bot/sendtyping" }
func (a *sendTypingAPI) HttpMethod() apic.HttpMethod { return apic.POST }
func (a *sendTypingAPI) PostBody() any               { return a.body }

func newSendTypingAPI(client *Client, baseURL, ilinkUserID, typingTicket string, status int) *sendTypingAPI {
	return &sendTypingAPI{
		Client:  client,
		baseURL: stringsTrimRightSlash(baseURL),
		body: sendTypingBody{
			BaseInfo: BaseInfo{ChannelVersion: ChannelVersion},
			SendTypingRequest: SendTypingRequest{
				ILinkUserID:  ilinkUserID,
				TypingTicket: typingTicket,
				Status:       status,
			},
		},
	}
}

func (a *sendTypingAPI) SendTyping(ctx context.Context, token string) error {
	headers, err := buildHeaders(token)
	if err != nil {
		return err
	}
	var out SendMessageResponse
	if err := a.callJSON(ctx, &apic.ApiId{
		Name:   "sendtyping",
		Client: a,
	}, &out, &apic.Options{
		Headers: headers,
		Timeout: 10 * time.Second,
	}); err != nil {
		return err
	}
	if out.Ret != 0 || out.ErrCode != 0 {
		return fmt.Errorf("sendtyping failed: ret=%d errcode=%d errmsg=%s", out.Ret, out.ErrCode, out.ErrMsg)
	}
	return nil
}
