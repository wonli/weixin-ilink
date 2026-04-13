package ilinkapi

import (
	"context"
	"fmt"
)

func (c *Client) UploadImageBuffer(ctx context.Context, baseURL, token, toUserID, cdnBaseURL string, image []byte) (*UploadedMedia, error) {
	return uploadMediaBuffer(ctx, c, baseURL, token, toUserID, cdnBaseURL, UploadMediaTypeImage, image)
}

func (c *Client) UploadVideoBuffer(ctx context.Context, baseURL, token, toUserID, cdnBaseURL string, video []byte) (*UploadedMedia, error) {
	return uploadMediaBuffer(ctx, c, baseURL, token, toUserID, cdnBaseURL, UploadMediaTypeVideo, video)
}

func (c *Client) UploadFileBuffer(ctx context.Context, baseURL, token, toUserID, cdnBaseURL string, file []byte) (*UploadedMedia, error) {
	return uploadMediaBuffer(ctx, c, baseURL, token, toUserID, cdnBaseURL, UploadMediaTypeFile, file)
}

func (c *Client) DownloadImage(ctx context.Context, cdnBaseURL string, item *ImageItem) ([]byte, error) {
	if item == nil || item.Media == nil || item.Media.EncryptQueryParam == "" {
		return nil, fmt.Errorf("image media is missing")
	}
	if item.Media.AESKey == "" && item.AESKeyHex == "" {
		return downloadPlainMedia(ctx, c, cdnBaseURL, item.Media.EncryptQueryParam)
	}
	aesKeyBase64, err := decodeImageAESKey(item)
	if err != nil {
		return nil, err
	}
	return downloadEncryptedMedia(ctx, c, cdnBaseURL, item.Media.EncryptQueryParam, aesKeyBase64)
}

func (c *Client) DownloadVoice(ctx context.Context, cdnBaseURL string, item *VoiceItem) ([]byte, error) {
	if item == nil || item.Media == nil || item.Media.EncryptQueryParam == "" || item.Media.AESKey == "" {
		return nil, fmt.Errorf("voice media is missing")
	}
	return downloadEncryptedMedia(ctx, c, cdnBaseURL, item.Media.EncryptQueryParam, item.Media.AESKey)
}

func (c *Client) DownloadFile(ctx context.Context, cdnBaseURL string, item *FileItem) ([]byte, error) {
	if item == nil || item.Media == nil || item.Media.EncryptQueryParam == "" || item.Media.AESKey == "" {
		return nil, fmt.Errorf("file media is missing")
	}
	return downloadEncryptedMedia(ctx, c, cdnBaseURL, item.Media.EncryptQueryParam, item.Media.AESKey)
}

func (c *Client) DownloadVideo(ctx context.Context, cdnBaseURL string, item *VideoItem) ([]byte, error) {
	if item == nil || item.Media == nil || item.Media.EncryptQueryParam == "" || item.Media.AESKey == "" {
		return nil, fmt.Errorf("video media is missing")
	}
	return downloadEncryptedMedia(ctx, c, cdnBaseURL, item.Media.EncryptQueryParam, item.Media.AESKey)
}

// DownloadRemoteMediaToTemp 直接把外部 URL 的媒体落盘到程序目录下。
func (c *Client) DownloadRemoteMediaToTemp(ctx context.Context, rawURL string) (string, error) {
	return downloadRemoteMediaToTemp(ctx, c, rawURL)
}

func (c *Client) DownloadImageToTemp(ctx context.Context, cdnBaseURL string, item *ImageItem) (string, error) {
	buf, err := c.DownloadImage(ctx, cdnBaseURL, item)
	if err != nil {
		return "", err
	}
	return saveImageToTemp(item, buf)
}

func (c *Client) DownloadFileToTemp(ctx context.Context, cdnBaseURL string, item *FileItem) (string, error) {
	buf, err := c.DownloadFile(ctx, cdnBaseURL, item)
	if err != nil {
		return "", err
	}
	return saveFileToTemp(item, buf)
}

func (c *Client) DownloadVideoToTemp(ctx context.Context, cdnBaseURL string, item *VideoItem) (string, error) {
	buf, err := c.DownloadVideo(ctx, cdnBaseURL, item)
	if err != nil {
		return "", err
	}
	return saveVideoToTemp(buf)
}

// DownloadVoiceToTemp 会优先把 Silk 转成 WAV，失败时再回退保存原始 Silk 文件。
func (c *Client) DownloadVoiceToTemp(ctx context.Context, cdnBaseURL string, item *VoiceItem) (string, string, error) {
	buf, err := c.DownloadVoice(ctx, cdnBaseURL, item)
	if err != nil {
		return "", "", err
	}
	return downloadVoiceToTemp(ctx, buf)
}

// TranscodeSilkToWAV 使用本地 ffmpeg 把微信语音转成更通用的 WAV。
func (c *Client) TranscodeSilkToWAV(ctx context.Context, silk []byte) ([]byte, error) {
	return transcodeSilkToWAV(ctx, silk)
}
