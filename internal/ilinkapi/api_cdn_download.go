package ilinkapi

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"

	"github.com/wonli/apic/v2"
)

var hex32Pattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

func buildCDNDownloadURL(cdnBaseURL, encryptedQueryParam string) string {
	return fmt.Sprintf(
		"%s/download?encrypted_query_param=%s",
		stringsTrimRightSlash(cdnBaseURL),
		url.QueryEscape(encryptedQueryParam),
	)
}

// cdnDownloadAPI 描述从 CDN 拉取原始媒体内容的请求。
// encryptedQueryParam 来自消息里的媒体字段，cdnBaseURL 则来自运行时配置或默认值。
type cdnDownloadAPI struct {
	*Client
	baseURL string
	path    string
	query   url.Values
}

func (a *cdnDownloadAPI) Url() string                 { return a.baseURL }
func (a *cdnDownloadAPI) Path() string                { return a.path }
func (a *cdnDownloadAPI) Query() url.Values           { return a.query }
func (a *cdnDownloadAPI) HttpMethod() apic.HttpMethod { return apic.GET }

// newCDNDownloadAPI 根据 CDN 基地址和媒体里的 encrypted_query_param 组装下载请求。
func newCDNDownloadAPI(client *Client, cdnBaseURL, encryptedQueryParam string) (*cdnDownloadAPI, error) {
	downloadURL := buildCDNDownloadURL(cdnBaseURL, encryptedQueryParam)
	parsed, err := url.Parse(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("parse download url: %w", err)
	}
	return &cdnDownloadAPI{
		Client:  client,
		baseURL: stringsTrimRightSlash(parsed.Scheme + "://" + parsed.Host),
		path:    parsed.EscapedPath(),
		query:   parsed.Query(),
	}, nil
}

func (a *cdnDownloadAPI) Download(ctx context.Context) ([]byte, error) {
	apiResp, err := a.callAPI(ctx, &apic.ApiId{
		Name:   "cdn_download",
		Client: a,
	}, nil)
	if err != nil {
		return nil, err
	}
	return apiResp.Data, nil
}

func downloadCDNBytes(ctx context.Context, client *Client, cdnBaseURL, encryptedQueryParam string) ([]byte, error) {
	api, err := newCDNDownloadAPI(client, cdnBaseURL, encryptedQueryParam)
	if err != nil {
		return nil, err
	}
	return api.Download(ctx)
}

func parseMediaAESKey(aesKeyBase64 string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(aesKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode aes_key: %w", err)
	}
	if len(decoded) == 16 {
		return decoded, nil
	}
	if len(decoded) == 32 && hex32Pattern.Match(decoded) {
		raw := make([]byte, hex.DecodedLen(len(decoded)))
		if _, err := hex.Decode(raw, decoded); err != nil {
			return nil, fmt.Errorf("decode hex aes_key: %w", err)
		}
		return raw, nil
	}
	return nil, fmt.Errorf("aes_key must decode to 16 raw bytes or 32-char hex string, got %d bytes", len(decoded))
}

func decodeImageAESKey(item *ImageItem) (string, error) {
	if item == nil {
		return "", fmt.Errorf("image item is nil")
	}
	if item.AESKeyHex != "" {
		raw, err := hex.DecodeString(item.AESKeyHex)
		if err != nil {
			return "", fmt.Errorf("decode image aeskey: %w", err)
		}
		return base64.StdEncoding.EncodeToString(raw), nil
	}
	if item.Media == nil || item.Media.AESKey == "" {
		return "", fmt.Errorf("image aes_key is missing")
	}
	return item.Media.AESKey, nil
}

// downloadEncryptedMedia 下载并解密 CDN 上的媒体内容。
func downloadEncryptedMedia(ctx context.Context, client *Client, cdnBaseURL, encryptedQueryParam, aesKeyBase64 string) ([]byte, error) {
	if stringsTrimRightSlash(cdnBaseURL) == "" {
		cdnBaseURL = DefaultCDNBaseURL
	}
	key, err := parseMediaAESKey(aesKeyBase64)
	if err != nil {
		return nil, err
	}
	encrypted, err := downloadCDNBytes(ctx, client, cdnBaseURL, encryptedQueryParam)
	if err != nil {
		return nil, err
	}
	return decryptAESECB(encrypted, key)
}

func downloadPlainMedia(ctx context.Context, client *Client, cdnBaseURL, encryptedQueryParam string) ([]byte, error) {
	if stringsTrimRightSlash(cdnBaseURL) == "" {
		cdnBaseURL = DefaultCDNBaseURL
	}
	return downloadCDNBytes(ctx, client, cdnBaseURL, encryptedQueryParam)
}
