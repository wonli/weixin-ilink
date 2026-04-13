package ilinkapi

import (
	"encoding/json"
	"testing"
)

func TestGetUploadURLBodyMarshalsBaseInfoAndPayload(t *testing.T) {
	body := getUploadURLBody{
		BaseInfo: BaseInfo{ChannelVersion: ChannelVersion},
		GetUploadURLRequest: GetUploadURLRequest{
			FileKey:     "fk",
			MediaType:   UploadMediaTypeImage,
			ToUserID:    "u1",
			RawSize:     12,
			RawFileMD5:  "md5",
			FileSize:    16,
			NoNeedThumb: true,
			AESKey:      "key",
		},
	}

	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf, &out); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if out["filekey"] != "fk" {
		t.Fatalf("expected filekey, got %#v", out["filekey"])
	}
	baseInfo, ok := out["base_info"].(map[string]any)
	if !ok {
		t.Fatalf("expected base_info object, got %#v", out["base_info"])
	}
	if baseInfo["channel_version"] != ChannelVersion {
		t.Fatalf("expected channel_version %q, got %#v", ChannelVersion, baseInfo["channel_version"])
	}
}

func TestGetUpdatesBodyMarshalsBaseInfoAndPayload(t *testing.T) {
	body := getUpdatesBody{
		BaseInfo: BaseInfo{ChannelVersion: ChannelVersion},
		GetUpdatesRequest: GetUpdatesRequest{
			GetUpdatesBuf: "cursor-1",
		},
	}

	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf, &out); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if out["get_updates_buf"] != "cursor-1" {
		t.Fatalf("expected get_updates_buf, got %#v", out["get_updates_buf"])
	}
}

func TestSendMessageBodyMarshalsBaseInfoAndMsg(t *testing.T) {
	body := sendMessageBody{
		BaseInfo: BaseInfo{ChannelVersion: ChannelVersion},
		Msg: WeixinMessage{
			ToUserID:     "user-1",
			ClientID:     "client-1",
			MessageType:  MessageTypeBot,
			MessageState: MessageStateFinish,
			ItemList: []MessageItem{{
				Type: MessageItemTypeText,
				TextItem: &TextItem{
					Text: "hello",
				},
			}},
		},
	}

	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf, &out); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	msg, ok := out["msg"].(map[string]any)
	if !ok {
		t.Fatalf("expected msg object, got %#v", out["msg"])
	}
	if msg["to_user_id"] != "user-1" {
		t.Fatalf("expected to_user_id, got %#v", msg["to_user_id"])
	}
}

func TestSendTypingBodyMarshalsBaseInfoAndPayload(t *testing.T) {
	body := sendTypingBody{
		BaseInfo: BaseInfo{ChannelVersion: ChannelVersion},
		SendTypingRequest: SendTypingRequest{
			ILinkUserID:  "user-1",
			TypingTicket: "ticket-1",
			Status:       TypingStatusTyping,
		},
	}

	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf, &out); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if out["typing_ticket"] != "ticket-1" {
		t.Fatalf("expected typing_ticket, got %#v", out["typing_ticket"])
	}
}
