package ilinkapi

import (
	"context"
	"fmt"
	"time"

	"github.com/wonli/apic/v2"
)

// sendMessageAPI 用于向微信会话发送消息。
// 文本、图片、视频、文件等最终都会组装成消息体后走这个接口发出。
type sendMessageAPI struct {
	*Client
	baseURL string
	body    sendMessageBody
}

func (a *sendMessageAPI) Url() string                 { return a.baseURL }
func (a *sendMessageAPI) Path() string                { return "/ilink/bot/sendmessage" }
func (a *sendMessageAPI) HttpMethod() apic.HttpMethod { return apic.POST }
func (a *sendMessageAPI) PostBody() any               { return a.body }

func newSendMessageAPI(client *Client, baseURL string, req SendMessageRequest) *sendMessageAPI {
	return &sendMessageAPI{
		Client:  client,
		baseURL: stringsTrimRightSlash(baseURL),
		body: sendMessageBody{
			BaseInfo: BaseInfo{ChannelVersion: ChannelVersion},
			Msg:      req.Msg,
		},
	}
}

func newClientMessageID() string {
	return fmt.Sprintf("go-%d", time.Now().UnixNano())
}

func newTextSendMessageAPI(client *Client, baseURL, toUserID, contextToken, text string) *sendMessageAPI {
	return newSendMessageAPI(client, baseURL, SendMessageRequest{
		Msg: WeixinMessage{
			FromUserID:   "",
			ToUserID:     toUserID,
			ClientID:     newClientMessageID(),
			MessageType:  MessageTypeBot,
			MessageState: MessageStateFinish,
			ContextToken: contextToken,
			ItemList: []MessageItem{{
				Type: MessageItemTypeText,
				TextItem: &TextItem{
					Text: text,
				},
			}},
		},
	})
}

func newImageSendMessageAPI(client *Client, baseURL, toUserID, contextToken string, uploaded *UploadedMedia) (*sendMessageAPI, error) {
	if uploaded == nil {
		return nil, fmt.Errorf("uploaded media is nil")
	}
	thumbParam := uploaded.ThumbDownloadEncryptedParam
	if thumbParam == "" {
		thumbParam = uploaded.DownloadEncryptedParam
	}
	aesKey := encodeUploadedAESKey(uploaded)
	return newSendMessageAPI(client, baseURL, SendMessageRequest{
		Msg: WeixinMessage{
			FromUserID:   "",
			ToUserID:     toUserID,
			ClientID:     newClientMessageID(),
			MessageType:  MessageTypeBot,
			MessageState: MessageStateFinish,
			ContextToken: contextToken,
			ItemList: []MessageItem{{
				Type: MessageItemTypeImage,
				ImageItem: &ImageItem{
					Media: &CDNMedia{
						EncryptQueryParam: uploaded.DownloadEncryptedParam,
						AESKey:            aesKey,
						EncryptType:       1,
					},
					ThumbMedia: &CDNMedia{
						EncryptQueryParam: thumbParam,
						AESKey:            aesKey,
						EncryptType:       1,
					},
					MidSize: int64(uploaded.FileSizeCiphertextBytes),
				},
			}},
		},
	}), nil
}

func newVideoSendMessageAPI(client *Client, baseURL, toUserID, contextToken string, uploaded *UploadedMedia) (*sendMessageAPI, error) {
	if uploaded == nil {
		return nil, fmt.Errorf("uploaded media is nil")
	}
	return newSendMessageAPI(client, baseURL, SendMessageRequest{
		Msg: WeixinMessage{
			FromUserID:   "",
			ToUserID:     toUserID,
			ClientID:     newClientMessageID(),
			MessageType:  MessageTypeBot,
			MessageState: MessageStateFinish,
			ContextToken: contextToken,
			ItemList: []MessageItem{{
				Type: MessageItemTypeVideo,
				VideoItem: &VideoItem{
					Media: &CDNMedia{
						EncryptQueryParam: uploaded.DownloadEncryptedParam,
						AESKey:            encodeUploadedAESKey(uploaded),
						EncryptType:       1,
					},
					VideoSize: int64(uploaded.FileSizeCiphertextBytes),
					VideoMD5:  uploaded.MD5Hex,
				},
			}},
		},
	}), nil
}

func newFileSendMessageAPI(client *Client, baseURL, toUserID, contextToken, fileName string, uploaded *UploadedMedia) (*sendMessageAPI, error) {
	if uploaded == nil {
		return nil, fmt.Errorf("uploaded media is nil")
	}
	return newSendMessageAPI(client, baseURL, SendMessageRequest{
		Msg: WeixinMessage{
			FromUserID:   "",
			ToUserID:     toUserID,
			ClientID:     newClientMessageID(),
			MessageType:  MessageTypeBot,
			MessageState: MessageStateFinish,
			ContextToken: contextToken,
			ItemList: []MessageItem{{
				Type: MessageItemTypeFile,
				FileItem: &FileItem{
					Media: &CDNMedia{
						EncryptQueryParam: uploaded.DownloadEncryptedParam,
						AESKey:            encodeUploadedAESKey(uploaded),
						EncryptType:       1,
					},
					FileName: fileName,
					MD5:      uploaded.MD5Hex,
					Len:      fmt.Sprintf("%d", uploaded.FileSize),
				},
			}},
		},
	}), nil
}

func (a *sendMessageAPI) SendMessage(ctx context.Context, token string) error {
	headers, err := buildHeaders(token)
	if err != nil {
		return err
	}
	var out SendMessageResponse
	if err := a.callJSON(ctx, &apic.ApiId{
		Name:   "sendmessage",
		Client: a,
	}, &out, &apic.Options{
		Headers: headers,
		Timeout: 15 * time.Second,
	}); err != nil {
		return err
	}
	if out.Ret != 0 || out.ErrCode != 0 {
		return fmt.Errorf("sendmessage failed: ret=%d errcode=%d errmsg=%s", out.Ret, out.ErrCode, out.ErrMsg)
	}
	return nil
}
