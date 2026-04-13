package ilinkapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var errRemoteMediaTooLarge = errors.New("remote media exceeds size limit")

func mediaProgramDir() (string, error) {
	exe, err := os.Executable()
	if err == nil && strings.TrimSpace(exe) != "" {
		dir := filepath.Join(filepath.Dir(exe), "weixin-media")
		if mkErr := os.MkdirAll(dir, 0o755); mkErr == nil {
			return dir, nil
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(wd, "weixin-media")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func sanitizeExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		return ".bin"
	}
	if !strings.HasPrefix(ext, ".") {
		return "." + ext
	}
	return ext
}

func extensionFromContentType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	case "audio/mpeg":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	case "audio/silk", "audio/x-silk":
		return ".silk"
	case "video/mp4":
		return ".mp4"
	case "application/pdf":
		return ".pdf"
	case "text/plain":
		return ".txt"
	default:
		return ""
	}
}

func extensionFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ".bin"
	}
	ext := path.Ext(parsed.Path)
	if ext == "" {
		return ".bin"
	}
	return sanitizeExt(ext)
}

func extensionFromBuffer(buf []byte) string {
	contentType := http.DetectContentType(buf)
	if ext := extensionFromContentType(contentType); ext != "" {
		return ext
	}
	return ".bin"
}

func saveBufferToProgramDir(prefix, ext string, data []byte) (string, error) {
	dir, err := mediaProgramDir()
	if err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s-%d%s", prefix, time.Now().UnixNano(), sanitizeExt(ext))
	fullPath := filepath.Join(dir, filename)
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", err
	}
	return fullPath, nil
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	return strings.ToLower(host)
}

func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	privateCIDRs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"100.64.0.0/10",
		"169.254.0.0/16",
		"fc00::/7",
		"fe80::/10",
	}
	for _, cidr := range privateCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func (c *Client) validateRemoteMediaURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("parse remote media url: %w", err)
	}
	if parsed.Scheme != "https" {
		return nil, fmt.Errorf("remote media url must use https")
	}
	host := normalizeHost(parsed.Host)
	if host == "" {
		return nil, fmt.Errorf("remote media url host is missing")
	}
	if len(c.allowHosts) > 0 {
		if _, ok := c.allowHosts[host]; !ok {
			return nil, fmt.Errorf("remote media host %q is not allowed", host)
		}
	}
	if ip := net.ParseIP(host); ip != nil && isPrivateIP(ip) && !c.allowPrivateHosts {
		return nil, fmt.Errorf("remote media host %q is private", host)
	}
	return parsed, nil
}

func (c *Client) downloadURLBytes(ctx context.Context, rawURL string) ([]byte, string, error) {
	if _, err := c.validateRemoteMediaURL(rawURL); err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.apiClient.HTTPClient().Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, "", fmt.Errorf("remote media download failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	limit := c.maxRemoteMediaBytes
	if limit <= 0 {
		limit = 20 << 20
	}
	reader := io.LimitReader(resp.Body, limit+1)
	buf, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}
	if int64(len(buf)) > limit {
		return nil, "", errRemoteMediaTooLarge
	}
	return buf, resp.Header.Get("Content-Type"), nil
}

func downloadRemoteMediaToTemp(ctx context.Context, client *Client, rawURL string) (string, error) {
	buf, contentType, err := client.downloadURLBytes(ctx, rawURL)
	if err != nil {
		return "", err
	}
	ext := extensionFromContentType(contentType)
	if ext == "" {
		ext = extensionFromURL(rawURL)
	}
	return saveBufferToProgramDir("weixin-remote", ext, buf)
}

func saveImageToTemp(item *ImageItem, buf []byte) (string, error) {
	ext := extensionFromBuffer(buf)
	if item != nil && strings.TrimSpace(item.URL) != "" {
		if urlExt := extensionFromURL(item.URL); urlExt != ".bin" {
			ext = urlExt
		}
	}
	return saveBufferToProgramDir("weixin-image", ext, buf)
}

func saveFileToTemp(item *FileItem, buf []byte) (string, error) {
	ext := ".bin"
	if item != nil && strings.TrimSpace(item.FileName) != "" {
		ext = sanitizeExt(filepath.Ext(item.FileName))
	}
	if ext == ".bin" {
		ext = extensionFromBuffer(buf)
	}
	return saveBufferToProgramDir("weixin-file", ext, buf)
}

func saveVideoToTemp(buf []byte) (string, error) {
	ext := extensionFromBuffer(buf)
	if ext == ".bin" {
		ext = ".mp4"
	}
	return saveBufferToProgramDir("weixin-video", ext, buf)
}

// downloadVoiceToTemp 会优先把 Silk 转成 WAV，失败时再回退保存原始 Silk 文件。
func downloadVoiceToTemp(ctx context.Context, voice []byte) (string, string, error) {
	log.Printf("weixin voice download complete bytes=%d attempting_transcode=true", len(voice))
	wav, err := transcodeSilkToWAV(ctx, voice)
	if err == nil && len(wav) > 0 {
		path, saveErr := saveBufferToProgramDir("weixin-voice", ".wav", wav)
		if saveErr != nil {
			log.Printf("weixin voice save failed format=wav err=%v", saveErr)
		} else {
			log.Printf("weixin voice save success format=wav path=%s bytes=%d", path, len(wav))
		}
		return path, "audio/wav", saveErr
	}
	log.Printf("weixin voice fallback format=silk reason=%v", err)
	path, saveErr := saveBufferToProgramDir("weixin-voice", ".silk", voice)
	if saveErr != nil {
		log.Printf("weixin voice save failed format=silk err=%v", saveErr)
	} else {
		log.Printf("weixin voice save success format=silk path=%s bytes=%d", path, len(voice))
	}
	return path, "audio/silk", saveErr
}

// transcodeSilkToWAV 使用本地 ffmpeg 把微信语音转成更通用的 WAV。
func transcodeSilkToWAV(ctx context.Context, silk []byte) ([]byte, error) {
	if len(silk) == 0 {
		return nil, fmt.Errorf("silk buffer is empty")
	}
	log.Printf("weixin voice transcode start input_bytes=%d", len(silk))
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		log.Printf("weixin voice transcode skip reason=ffmpeg_not_found err=%v", err)
		return nil, fmt.Errorf("ffmpeg not found: %w", err)
	}
	dir, err := mediaProgramDir()
	if err != nil {
		log.Printf("weixin voice transcode failed stage=prepare_dir err=%v", err)
		return nil, err
	}
	inputPath := filepath.Join(dir, fmt.Sprintf("weixin-voice-src-%d.silk", unixNano()))
	outputPath := filepath.Join(dir, fmt.Sprintf("weixin-voice-out-%d.wav", unixNano()))
	if err := os.WriteFile(inputPath, silk, 0o644); err != nil {
		log.Printf("weixin voice transcode failed stage=write_input path=%s err=%v", inputPath, err)
		return nil, err
	}
	defer os.Remove(inputPath)
	defer os.Remove(outputPath)

	cmd := exec.CommandContext(
		ctx,
		ffmpegPath,
		"-y",
		"-i", inputPath,
		"-ar", "24000",
		"-ac", "1",
		outputPath,
	)
	var stderr bytes.Buffer
	cmd.Stdout = ioDiscard{}
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("weixin voice transcode failed stage=ffmpeg input=%s output=%s err=%v stderr=%s", inputPath, outputPath, err, stderr.String())
		return nil, fmt.Errorf("ffmpeg transcode failed: %w: %s", err, stderr.String())
	}
	out, err := os.ReadFile(outputPath)
	if err != nil {
		log.Printf("weixin voice transcode failed stage=read_output path=%s err=%v", outputPath, err)
		return nil, err
	}
	log.Printf("weixin voice transcode success input_bytes=%d output_bytes=%d ffmpeg=%s", len(silk), len(out), ffmpegPath)
	return out, nil
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
