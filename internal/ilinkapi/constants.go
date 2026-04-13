package ilinkapi

import "time"

const (
	DefaultBaseURL      = "https://ilinkai.weixin.qq.com"
	DefaultCDNBaseURL   = "https://novac2c.cdn.weixin.qq.com/c2c"
	DefaultBotType      = "3"
	ChannelVersion      = "1.0.0-go"
	DefaultPollTimeout  = 38 * time.Second
	DefaultLoginTimeout = 35 * time.Second
	SessionExpiredCode  = -14
)

const (
	MessageTypeUser = 1
	MessageTypeBot  = 2

	MessageStateNew        = 0
	MessageStateGenerating = 1
	MessageStateFinish     = 2

	MessageItemTypeText  = 1
	MessageItemTypeImage = 2
	MessageItemTypeVoice = 3
	MessageItemTypeFile  = 4
	MessageItemTypeVideo = 5

	UploadMediaTypeImage = 1
	UploadMediaTypeVideo = 2
	UploadMediaTypeFile  = 3
	UploadMediaTypeVoice = 4

	TypingStatusTyping = 1
	TypingStatusCancel = 2
)
