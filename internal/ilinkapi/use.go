package ilinkapi

import "context"

type API interface {
	StartLogin(ctx context.Context) (*QRCodeResponse, error)
	PollLogin(ctx context.Context, qrcode string) (*QRStatusResponse, error)
	GetUpdates(ctx context.Context, token, buf string) (*GetUpdatesResponse, error)
	SendText(ctx context.Context, token, toUserID, contextToken, text string) error
	SendMessage(ctx context.Context, token string, req SendMessageRequest) error
	GetUploadURL(ctx context.Context, token string, req GetUploadURLRequest) (*GetUploadURLResponse, error)
	GetConfig(ctx context.Context, token, ilinkUserID, contextToken string) (*GetConfigResponse, error)
	SendTyping(ctx context.Context, token, ilinkUserID, typingTicket string, status int) error
	UploadImageBuffer(ctx context.Context, token, toUserID string, image []byte) (*UploadedMedia, error)
	UploadVideoBuffer(ctx context.Context, token, toUserID string, video []byte) (*UploadedMedia, error)
	UploadFileBuffer(ctx context.Context, token, toUserID string, file []byte) (*UploadedMedia, error)
	SendImage(ctx context.Context, token, toUserID, contextToken string, uploaded *UploadedMedia) error
	SendVideo(ctx context.Context, token, toUserID, contextToken string, uploaded *UploadedMedia) error
	SendFile(ctx context.Context, token, toUserID, contextToken, fileName string, uploaded *UploadedMedia) error
	DownloadImage(ctx context.Context, item *ImageItem) ([]byte, error)
	DownloadVoice(ctx context.Context, item *VoiceItem) ([]byte, error)
	DownloadFile(ctx context.Context, item *FileItem) ([]byte, error)
	DownloadVideo(ctx context.Context, item *VideoItem) ([]byte, error)
	DownloadRemoteMediaToTemp(ctx context.Context, rawURL string) (string, error)
	DownloadImageToTemp(ctx context.Context, item *ImageItem) (string, error)
	DownloadVoiceToTemp(ctx context.Context, item *VoiceItem) (string, string, error)
	DownloadFileToTemp(ctx context.Context, item *FileItem) (string, error)
	DownloadVideoToTemp(ctx context.Context, item *VideoItem) (string, error)
	TranscodeSilkToWAV(ctx context.Context, silk []byte) ([]byte, error)
	RawClient() *Client
}

type defaultAPI struct {
	client     *Client
	baseURL    string
	cdnBaseURL string
}

func Use() API {
	return &defaultAPI{
		client:     NewClient(),
		baseURL:    DefaultBaseURL,
		cdnBaseURL: DefaultCDNBaseURL,
	}
}

func (a *defaultAPI) StartLogin(ctx context.Context) (*QRCodeResponse, error) {
	return a.client.StartLogin(ctx, a.baseURL)
}

func (a *defaultAPI) PollLogin(ctx context.Context, qrcode string) (*QRStatusResponse, error) {
	return a.client.PollLogin(ctx, a.baseURL, qrcode)
}

func (a *defaultAPI) GetUpdates(ctx context.Context, token, buf string) (*GetUpdatesResponse, error) {
	return a.client.GetUpdates(ctx, a.baseURL, token, buf)
}

func (a *defaultAPI) SendText(ctx context.Context, token, toUserID, contextToken, text string) error {
	return a.client.SendText(ctx, a.baseURL, token, toUserID, contextToken, text)
}

func (a *defaultAPI) SendMessage(ctx context.Context, token string, req SendMessageRequest) error {
	return a.client.SendMessage(ctx, a.baseURL, token, req)
}

func (a *defaultAPI) GetUploadURL(ctx context.Context, token string, req GetUploadURLRequest) (*GetUploadURLResponse, error) {
	return a.client.GetUploadURL(ctx, a.baseURL, token, req)
}

func (a *defaultAPI) GetConfig(ctx context.Context, token, ilinkUserID, contextToken string) (*GetConfigResponse, error) {
	return a.client.GetConfig(ctx, a.baseURL, token, ilinkUserID, contextToken)
}

func (a *defaultAPI) SendTyping(ctx context.Context, token, ilinkUserID, typingTicket string, status int) error {
	return a.client.SendTyping(ctx, a.baseURL, token, ilinkUserID, typingTicket, status)
}

func (a *defaultAPI) UploadImageBuffer(ctx context.Context, token, toUserID string, image []byte) (*UploadedMedia, error) {
	return a.client.UploadImageBuffer(ctx, a.baseURL, token, toUserID, a.cdnBaseURL, image)
}

func (a *defaultAPI) UploadVideoBuffer(ctx context.Context, token, toUserID string, video []byte) (*UploadedMedia, error) {
	return a.client.UploadVideoBuffer(ctx, a.baseURL, token, toUserID, a.cdnBaseURL, video)
}

func (a *defaultAPI) UploadFileBuffer(ctx context.Context, token, toUserID string, file []byte) (*UploadedMedia, error) {
	return a.client.UploadFileBuffer(ctx, a.baseURL, token, toUserID, a.cdnBaseURL, file)
}

func (a *defaultAPI) SendImage(ctx context.Context, token, toUserID, contextToken string, uploaded *UploadedMedia) error {
	return a.client.SendImage(ctx, a.baseURL, token, toUserID, contextToken, uploaded)
}

func (a *defaultAPI) SendVideo(ctx context.Context, token, toUserID, contextToken string, uploaded *UploadedMedia) error {
	return a.client.SendVideo(ctx, a.baseURL, token, toUserID, contextToken, uploaded)
}

func (a *defaultAPI) SendFile(ctx context.Context, token, toUserID, contextToken, fileName string, uploaded *UploadedMedia) error {
	return a.client.SendFile(ctx, a.baseURL, token, toUserID, contextToken, fileName, uploaded)
}

func (a *defaultAPI) DownloadImage(ctx context.Context, item *ImageItem) ([]byte, error) {
	return a.client.DownloadImage(ctx, a.cdnBaseURL, item)
}

func (a *defaultAPI) DownloadVoice(ctx context.Context, item *VoiceItem) ([]byte, error) {
	return a.client.DownloadVoice(ctx, a.cdnBaseURL, item)
}

func (a *defaultAPI) DownloadFile(ctx context.Context, item *FileItem) ([]byte, error) {
	return a.client.DownloadFile(ctx, a.cdnBaseURL, item)
}

func (a *defaultAPI) DownloadVideo(ctx context.Context, item *VideoItem) ([]byte, error) {
	return a.client.DownloadVideo(ctx, a.cdnBaseURL, item)
}

func (a *defaultAPI) DownloadRemoteMediaToTemp(ctx context.Context, rawURL string) (string, error) {
	return a.client.DownloadRemoteMediaToTemp(ctx, rawURL)
}

func (a *defaultAPI) DownloadImageToTemp(ctx context.Context, item *ImageItem) (string, error) {
	return a.client.DownloadImageToTemp(ctx, a.cdnBaseURL, item)
}

func (a *defaultAPI) DownloadVoiceToTemp(ctx context.Context, item *VoiceItem) (string, string, error) {
	return a.client.DownloadVoiceToTemp(ctx, a.cdnBaseURL, item)
}

func (a *defaultAPI) DownloadFileToTemp(ctx context.Context, item *FileItem) (string, error) {
	return a.client.DownloadFileToTemp(ctx, a.cdnBaseURL, item)
}

func (a *defaultAPI) DownloadVideoToTemp(ctx context.Context, item *VideoItem) (string, error) {
	return a.client.DownloadVideoToTemp(ctx, a.cdnBaseURL, item)
}

func (a *defaultAPI) TranscodeSilkToWAV(ctx context.Context, silk []byte) ([]byte, error) {
	return a.client.TranscodeSilkToWAV(ctx, silk)
}

func (a *defaultAPI) RawClient() *Client {
	return a.client
}
