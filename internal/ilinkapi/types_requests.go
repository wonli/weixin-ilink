package ilinkapi

type GetUploadURLRequest struct {
	FileKey         string `json:"filekey"`
	MediaType       int    `json:"media_type"`
	ToUserID        string `json:"to_user_id"`
	RawSize         int    `json:"rawsize"`
	RawFileMD5      string `json:"rawfilemd5"`
	FileSize        int    `json:"filesize"`
	NoNeedThumb     bool   `json:"no_need_thumb,omitempty"`
	ThumbRawSize    int    `json:"thumb_rawsize,omitempty"`
	ThumbRawFileMD5 string `json:"thumb_rawfilemd5,omitempty"`
	ThumbFileSize   int    `json:"thumb_filesize,omitempty"`
	AESKey          string `json:"aeskey"`
}

type getUploadURLBody struct {
	BaseInfo BaseInfo `json:"base_info"`
	GetUploadURLRequest
}

type GetUpdatesRequest struct {
	GetUpdatesBuf string `json:"get_updates_buf"`
}

type getUpdatesBody struct {
	BaseInfo BaseInfo `json:"base_info"`
	GetUpdatesRequest
}

type GetConfigRequest struct {
	ILinkUserID  string `json:"ilink_user_id"`
	ContextToken string `json:"context_token"`
}

type getConfigBody struct {
	BaseInfo BaseInfo `json:"base_info"`
	GetConfigRequest
}

type SendTypingRequest struct {
	ILinkUserID  string `json:"ilink_user_id"`
	TypingTicket string `json:"typing_ticket"`
	Status       int    `json:"status"`
}

type sendTypingBody struct {
	BaseInfo BaseInfo `json:"base_info"`
	SendTypingRequest
}

type sendMessageBody struct {
	BaseInfo BaseInfo      `json:"base_info"`
	Msg      WeixinMessage `json:"msg"`
}
