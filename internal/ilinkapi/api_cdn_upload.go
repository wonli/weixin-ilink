package ilinkapi

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/wonli/apic/v2"
)

// requestImageUploadURL 兼容图片上传接口的两种参数模式。
// 当首轮 no_need_thumb 请求没有返回上传地址时，再自动降级到携带缩略图参数的模式。
func requestImageUploadURL(ctx context.Context, client *Client, baseURL, token, toUserID, fileKey, aesKeyHex string, image []byte) (*GetUploadURLResponse, error) {
	rawMD5 := md5.Sum(image)
	filesize := aesEcbPaddedSize(len(image))

	req := GetUploadURLRequest{
		FileKey:     fileKey,
		MediaType:   UploadMediaTypeImage,
		ToUserID:    toUserID,
		RawSize:     len(image),
		RawFileMD5:  hex.EncodeToString(rawMD5[:]),
		FileSize:    filesize,
		NoNeedThumb: true,
		AESKey:      aesKeyHex,
	}
	resp, err := client.GetUploadURL(ctx, baseURL, token, req)
	if err != nil {
		return nil, err
	}
	if resp.UploadParam != "" || resp.UploadFullURL != "" {
		return resp, nil
	}

	req = GetUploadURLRequest{
		FileKey:         fileKey,
		MediaType:       UploadMediaTypeImage,
		ToUserID:        toUserID,
		RawSize:         len(image),
		RawFileMD5:      hex.EncodeToString(rawMD5[:]),
		FileSize:        filesize,
		ThumbRawSize:    len(image),
		ThumbRawFileMD5: hex.EncodeToString(rawMD5[:]),
		ThumbFileSize:   filesize,
		AESKey:          aesKeyHex,
	}
	return client.GetUploadURL(ctx, baseURL, token, req)
}

func randomHex(bytesLen int) (string, error) {
	raw := make([]byte, bytesLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func buildCDNUploadURL(cdnBaseURL, uploadParam, fileKey string) string {
	return fmt.Sprintf(
		"%s/upload?encrypted_query_param=%s&filekey=%s",
		cdnBaseURL,
		url.QueryEscape(uploadParam),
		url.QueryEscape(fileKey),
	)
}

func resolveUploadURL(cdnBaseURL, uploadParam, uploadFullURL, fileKey string) string {
	if uploadFullURL != "" {
		return uploadFullURL
	}
	if uploadParam == "" {
		return ""
	}
	return buildCDNUploadURL(cdnBaseURL, uploadParam, fileKey)
}

func pickUploadTarget(primaryParam, primaryFullURL, fallbackParam, fallbackFullURL string) (uploadParam, uploadFullURL string) {
	if primaryParam != "" || primaryFullURL != "" {
		return primaryParam, primaryFullURL
	}
	return fallbackParam, fallbackFullURL
}

// cdnUploadAPI 描述向 CDN 上传加密媒体的原始二进制请求。
// uploadParam / fileKey 来自 getuploadurl，ciphertext 则是业务媒体加密后的结果。
type cdnUploadAPI struct {
	*Client
	baseURL    string
	path       string
	query      url.Values
	ciphertext []byte
}

func (a *cdnUploadAPI) Url() string                 { return a.baseURL }
func (a *cdnUploadAPI) Path() string                { return a.path }
func (a *cdnUploadAPI) Query() url.Values           { return a.query }
func (a *cdnUploadAPI) HttpMethod() apic.HttpMethod { return apic.POST }
func (a *cdnUploadAPI) PostBody() any               { return a.ciphertext }
func (a *cdnUploadAPI) SetData(data *apic.SetDataRequest) error {
	data.Type = apic.DataTypeRaw
	data.Content = a.ciphertext
	data.ContentType = "application/octet-stream"
	return nil
}

// newCDNUploadAPI 根据 getuploadurl 返回的上传信息组装 CDN 上传请求。
// 如果服务端直接给出完整 URL，则优先使用；否则根据 uploadParam 和 fileKey 本地拼装。
func newCDNUploadAPI(client *Client, cdnBaseURL, uploadParam, uploadFullURL, fileKey string, ciphertext []byte) (*cdnUploadAPI, error) {
	uploadURL := resolveUploadURL(cdnBaseURL, uploadParam, uploadFullURL, fileKey)
	if uploadURL == "" {
		return nil, fmt.Errorf("empty upload url")
	}
	parsed, err := url.Parse(uploadURL)
	if err != nil {
		return nil, fmt.Errorf("parse upload url: %w", err)
	}
	return &cdnUploadAPI{
		Client:     client,
		baseURL:    stringsTrimRightSlash(parsed.Scheme + "://" + parsed.Host),
		path:       parsed.EscapedPath(),
		query:      parsed.Query(),
		ciphertext: ciphertext,
	}, nil
}

func (a *cdnUploadAPI) Upload(ctx context.Context) (string, error) {
	apiResp, err := a.callAPI(ctx, &apic.ApiId{
		Name:   "cdn_upload",
		Client: a,
	}, &apic.Options{
		Timeout: 20 * time.Second,
	})
	if err != nil {
		return "", err
	}
	downloadParam := apiResp.Header.Get("x-encrypted-param")
	if downloadParam == "" {
		return "", fmt.Errorf("cdn upload missing x-encrypted-param")
	}
	return downloadParam, nil
}

func uploadBufferToCDN(ctx context.Context, client *Client, cdnBaseURL, uploadParam, uploadFullURL, fileKey string, plaintext, aesKey []byte) (string, error) {
	ciphertext, err := encryptAESECB(plaintext, aesKey)
	if err != nil {
		return "", fmt.Errorf("encrypt media: %w", err)
	}
	api, err := newCDNUploadAPI(client, cdnBaseURL, uploadParam, uploadFullURL, fileKey, ciphertext)
	if err != nil {
		return "", err
	}
	return api.Upload(ctx)
}

// uploadMediaBuffer 完成“申请上传地址 -> CDN 加密上传 -> 组装媒体元信息”的全流程。
func uploadMediaBuffer(ctx context.Context, client *Client, baseURL, token, toUserID, cdnBaseURL string, mediaType int, media []byte) (*UploadedMedia, error) {
	if len(media) == 0 {
		return nil, fmt.Errorf("media buffer is empty")
	}
	if cdnBaseURL == "" {
		cdnBaseURL = DefaultCDNBaseURL
	}

	fileKey, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generate filekey: %w", err)
	}
	aesKeyHex, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generate aes key: %w", err)
	}
	aesKey, err := hex.DecodeString(aesKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode aes key: %w", err)
	}

	rawMD5 := md5.Sum(media)
	md5Hex := hex.EncodeToString(rawMD5[:])
	filesize := aesEcbPaddedSize(len(media))

	var uploadResp *GetUploadURLResponse
	if mediaType == UploadMediaTypeImage {
		uploadResp, err = requestImageUploadURL(ctx, client, baseURL, token, toUserID, fileKey, aesKeyHex, media)
	} else {
		uploadResp, err = client.GetUploadURL(ctx, baseURL, token, GetUploadURLRequest{
			FileKey:     fileKey,
			MediaType:   mediaType,
			ToUserID:    toUserID,
			RawSize:     len(media),
			RawFileMD5:  md5Hex,
			FileSize:    filesize,
			NoNeedThumb: true,
			AESKey:      aesKeyHex,
		})
	}
	if err != nil {
		return nil, err
	}

	uploadURL := resolveUploadURL(cdnBaseURL, uploadResp.UploadParam, uploadResp.UploadFullURL, fileKey)
	thumbUploadURL := resolveUploadURL(cdnBaseURL, uploadResp.ThumbUploadParam, uploadResp.ThumbUploadFullURL, fileKey)
	if uploadURL == "" {
		if thumbUploadURL == "" {
			return nil, fmt.Errorf(
				"getuploadurl returned no usable upload url (ret=%d errcode=%d errmsg=%q upload_param_empty=%t thumb_upload_param_empty=%t upload_full_url_empty=%t thumb_upload_full_url_empty=%t)",
				uploadResp.Ret,
				uploadResp.ErrCode,
				uploadResp.ErrMsg,
				uploadResp.UploadParam == "",
				uploadResp.ThumbUploadParam == "",
				uploadResp.UploadFullURL == "",
				uploadResp.ThumbUploadFullURL == "",
			)
		}
		uploadURL = thumbUploadURL
	}

	selectedUploadParam, selectedUploadFullURL := pickUploadTarget(
		uploadResp.UploadParam,
		uploadResp.UploadFullURL,
		uploadResp.ThumbUploadParam,
		uploadResp.ThumbUploadFullURL,
	)
	downloadParam, err := uploadBufferToCDN(ctx, client, cdnBaseURL, selectedUploadParam, selectedUploadFullURL, fileKey, media, aesKey)
	if err != nil {
		return nil, err
	}
	thumbDownloadParam := ""
	if thumbUploadURL != "" {
		thumbDownloadParam, err = uploadBufferToCDN(ctx, client, cdnBaseURL, uploadResp.ThumbUploadParam, uploadResp.ThumbUploadFullURL, fileKey, media, aesKey)
		if err != nil {
			return nil, fmt.Errorf("upload thumbnail to cdn: %w", err)
		}
	}

	return &UploadedMedia{
		FileKey:                     fileKey,
		DownloadEncryptedParam:      downloadParam,
		ThumbDownloadEncryptedParam: thumbDownloadParam,
		AESKeyHex:                   aesKeyHex,
		MD5Hex:                      md5Hex,
		FileSize:                    len(media),
		FileSizeCiphertextBytes:     filesize,
	}, nil
}
