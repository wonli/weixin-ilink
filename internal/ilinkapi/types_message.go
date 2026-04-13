package ilinkapi

type GetUpdatesResponse struct {
	Ret                int             `json:"ret"`
	ErrCode            int             `json:"errcode"`
	ErrMsg             string          `json:"errmsg"`
	Msgs               []WeixinMessage `json:"msgs"`
	GetUpdatesBuf      string          `json:"get_updates_buf"`
	LongPollingTimeout int             `json:"longpolling_timeout_ms"`
}

type SendMessageRequest struct {
	Msg WeixinMessage `json:"msg"`
}

type SendMessageResponse struct {
	Ret     int    `json:"ret"`
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

type WeixinMessage struct {
	Seq          int64         `json:"seq"`
	MessageID    int64         `json:"message_id"`
	FromUserID   string        `json:"from_user_id"`
	ToUserID     string        `json:"to_user_id"`
	ClientID     string        `json:"client_id"`
	SessionID    string        `json:"session_id"`
	GroupID      string        `json:"group_id"`
	MessageType  int           `json:"message_type"`
	MessageState int           `json:"message_state"`
	ContextToken string        `json:"context_token"`
	CreateTimeMS int64         `json:"create_time_ms"`
	UpdateTimeMS int64         `json:"update_time_ms"`
	DeleteTimeMS int64         `json:"delete_time_ms"`
	ItemList     []MessageItem `json:"item_list"`
}

type MessageItem struct {
	Type         int         `json:"type"`
	CreateTimeMS int64       `json:"create_time_ms,omitempty"`
	UpdateTimeMS int64       `json:"update_time_ms,omitempty"`
	IsCompleted  bool        `json:"is_completed,omitempty"`
	MsgID        string      `json:"msg_id,omitempty"`
	RefMsg       *RefMessage `json:"ref_msg,omitempty"`
	TextItem     *TextItem   `json:"text_item,omitempty"`
	ImageItem    *ImageItem  `json:"image_item,omitempty"`
	VoiceItem    *VoiceItem  `json:"voice_item,omitempty"`
	FileItem     *FileItem   `json:"file_item,omitempty"`
	VideoItem    *VideoItem  `json:"video_item,omitempty"`
}

type TextItem struct {
	Text string `json:"text"`
}

type RefMessage struct {
	MessageItem *MessageItem `json:"message_item,omitempty"`
	Title       string       `json:"title,omitempty"`
}
