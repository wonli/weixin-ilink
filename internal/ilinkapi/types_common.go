package ilinkapi

type BaseInfo struct {
	ChannelVersion string `json:"channel_version"`
}

type QRCodeResponse struct {
	QRCode           string `json:"qrcode"`
	QRCodeImgContent string `json:"qrcode_img_content"`
}

type QRStatusResponse struct {
	Status      string `json:"status"`
	BotToken    string `json:"bot_token"`
	ILinkBotID  string `json:"ilink_bot_id"`
	BaseURL     string `json:"baseurl"`
	ILinkUserID string `json:"ilink_user_id"`
}

type GetUploadURLResponse struct {
	Ret                int    `json:"ret"`
	ErrCode            int    `json:"errcode"`
	ErrMsg             string `json:"errmsg"`
	UploadParam        string `json:"upload_param"`
	ThumbUploadParam   string `json:"thumb_upload_param"`
	UploadFullURL      string `json:"upload_full_url"`
	ThumbUploadFullURL string `json:"thumb_upload_full_url"`
}

type GetConfigResponse struct {
	Ret          int    `json:"ret"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
	TypingTicket string `json:"typing_ticket"`
}
