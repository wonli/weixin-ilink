package app

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	"github.com/wonli/weixin-ilink/internal/assets"
	"github.com/wonli/weixin-ilink/internal/ilinkapi"
	"github.com/wonli/weixin-ilink/internal/session"
)

const sharedQRCodeWatchTimeout = 8 * time.Minute

func messageKinds(msg ilinkapi.WeixinMessage) []string {
	kinds := make([]string, 0, len(msg.ItemList))
	seen := map[string]bool{}
	add := func(kind string) {
		if !seen[kind] {
			seen[kind] = true
			kinds = append(kinds, kind)
		}
	}
	for _, item := range msg.ItemList {
		switch item.Type {
		case ilinkapi.MessageItemTypeText:
			add("文本")
		case ilinkapi.MessageItemTypeImage:
			add("图片")
		case ilinkapi.MessageItemTypeVoice:
			add("语音")
		case ilinkapi.MessageItemTypeFile:
			add("文件")
		case ilinkapi.MessageItemTypeVideo:
			add("视频")
		}
	}
	if len(kinds) == 0 {
		kinds = append(kinds, "未知")
	}
	return kinds
}

func buildQRCodePosterPNG(qrContent string) ([]byte, error) {
	const (
		qrSize = 640
		qrTopY = 450
	)

	qrPNG, err := qrcode.Encode(strings.TrimSpace(qrContent), qrcode.Medium, qrSize)
	if err != nil {
		return nil, fmt.Errorf("generate qrcode png: %w", err)
	}
	qrImg, err := png.Decode(bytes.NewReader(qrPNG))
	if err != nil {
		return nil, fmt.Errorf("decode qrcode png: %w", err)
	}
	bgImg, err := png.Decode(bytes.NewReader(assets.ShareBGPNG))
	if err != nil {
		return nil, fmt.Errorf("decode embedded share background: %w", err)
	}

	canvas := image.NewRGBA(bgImg.Bounds())
	draw.Draw(canvas, canvas.Bounds(), bgImg, bgImg.Bounds().Min, draw.Src)

	qrRect := image.Rect((canvas.Bounds().Dx()-qrSize)/2, qrTopY, (canvas.Bounds().Dx()-qrSize)/2+qrSize, qrTopY+qrSize)
	draw.Draw(canvas, qrRect, qrImg, image.Point{}, draw.Over)

	var out bytes.Buffer
	if err := png.Encode(&out, canvas); err != nil {
		return nil, fmt.Errorf("encode qrcode card: %w", err)
	}
	return out.Bytes(), nil
}

func (a *App) sendShareQRCode(accountID, userID, contextToken string) error {
	rt := a.runtimeByID(accountID)
	if rt == nil {
		return fmt.Errorf("runtime not found")
	}
	rt.state.StartOutboundDraft(userID, time.Now())
	defer a.persistContacts(accountID)

	baseURL := rt.record.BaseURL
	if baseURL == "" {
		baseURL = ilinkapi.DefaultBaseURL
	}
	if strings.TrimSpace(rt.record.Token) == "" {
		return fmt.Errorf("not logged in")
	}

	qrResp, err := a.client.StartLogin(context.Background(), ilinkapi.DefaultBaseURL)
	if err != nil {
		return fmt.Errorf("get official bot qrcode: %w", err)
	}
	png, err := buildQRCodePosterPNG(qrResp.QRCodeImgContent)
	if err != nil {
		return fmt.Errorf("generate qrcode card png: %w", err)
	}
	rt.state.AppendOutboundDraft(userID, "[二维码图片生成中]", time.Now())
	rt.state.MarkOutboundDraftSending(userID, time.Now())

	uploaded, err := a.client.UploadImageBuffer(context.Background(), baseURL, rt.record.Token, userID, ilinkapi.DefaultCDNBaseURL, png)
	if err != nil {
		return fmt.Errorf("upload qrcode image: %w", err)
	}
	if err := a.client.SendImage(context.Background(), baseURL, rt.record.Token, userID, contextToken, uploaded); err != nil {
		return fmt.Errorf("send qrcode image: %w", err)
	}

	record := session.PendingQRCodeRecord{
		QRCode:          qrResp.QRCode,
		SourceAccountID: accountID,
		SharedToUserID:  userID,
		CreatedAt:       time.Now().Format(time.RFC3339Nano),
		Status:          "pending",
	}
	a.persistPendingQRCode(record)
	a.watchSharedQRCodeLogin(record)
	a.updateAccountRequest(accountID, "send_qrcode", "ok", "", 0)
	a.markSendOK(accountID)
	rt.state.AddOutbound(userID, "[二维码图片已发送]", time.Now())
	return nil
}

func (a *App) restorePendingQRCodes(records []session.PendingQRCodeRecord) {
	for _, record := range records {
		if strings.TrimSpace(record.Status) == "expired" {
			continue
		}
		if remainingSharedQRCodeWatch(record.CreatedAt) <= 0 {
			a.markPendingQRCodeExpired(strings.TrimSpace(record.QRCode), time.Now().Format(time.RFC3339Nano))
			continue
		}
		a.watchSharedQRCodeLogin(record)
	}
}

func remainingSharedQRCodeWatch(createdAt string) time.Duration {
	createdAt = strings.TrimSpace(createdAt)
	if createdAt == "" {
		return sharedQRCodeWatchTimeout
	}
	startedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return sharedQRCodeWatchTimeout
	}
	remaining := sharedQRCodeWatchTimeout - time.Since(startedAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (a *App) watchSharedQRCodeLogin(record session.PendingQRCodeRecord) {
	record.QRCode = strings.TrimSpace(record.QRCode)
	if record.QRCode == "" {
		return
	}

	a.mu.Lock()
	if _, exists := a.sharedQRCodeCancels[record.QRCode]; exists {
		a.mu.Unlock()
		return
	}
	timeout := remainingSharedQRCodeWatch(record.CreatedAt)
	if timeout <= 0 {
		a.mu.Unlock()
		a.markPendingQRCodeExpired(record.QRCode, time.Now().Format(time.RFC3339Nano))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	a.sharedQRCodeCancels[record.QRCode] = cancel
	a.mu.Unlock()

	go func() {
		defer func() {
			cancel()
			a.mu.Lock()
			delete(a.sharedQRCodeCancels, record.QRCode)
			a.mu.Unlock()
		}()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			resp, err := a.client.PollLogin(ctx, ilinkapi.DefaultBaseURL, record.QRCode)
			if err != nil {
				if isBenignCancel(err, ctx) {
					return
				}
				log.Printf("shared qrcode poll failed: %v", err)
				continue
			}

			switch resp.Status {
			case "wait", "scaned":
				continue
			case "expired":
				log.Printf("shared qrcode expired without confirmation")
				a.markPendingQRCodeExpired(record.QRCode, time.Now().Format(time.RFC3339Nano))
				return
			case "confirmed":
				if strings.TrimSpace(resp.ILinkBotID) == "" {
					log.Printf("shared qrcode confirmed but ilink_bot_id missing")
					a.markPendingQRCodeExpired(record.QRCode, time.Now().Format(time.RFC3339Nano))
					return
				}
				a.upsertAccount(resp)
				log.Printf("shared qrcode confirmed, added bot account %s", resp.ILinkBotID)
				a.removePendingQRCode(record.QRCode)
				return
			default:
				log.Printf("shared qrcode returned unexpected status %q", resp.Status)
			}
		}
	}()
}

func (a *App) persistPendingQRCode(record session.PendingQRCodeRecord) {
	record.QRCode = strings.TrimSpace(record.QRCode)
	if record.QRCode == "" {
		return
	}
	if strings.TrimSpace(record.Status) == "" {
		record.Status = "pending"
	}
	_ = a.store.Update(func(d *session.Data) {
		for i := range d.PendingQRCodes {
			if strings.TrimSpace(d.PendingQRCodes[i].QRCode) == record.QRCode {
				d.PendingQRCodes[i] = record
				return
			}
		}
		d.PendingQRCodes = append(d.PendingQRCodes, record)
	})
}

func (a *App) markPendingQRCodeExpired(qrcode, expiredAt string) {
	qrcode = strings.TrimSpace(qrcode)
	if qrcode == "" {
		return
	}
	_ = a.store.Update(func(d *session.Data) {
		for i := range d.PendingQRCodes {
			if strings.TrimSpace(d.PendingQRCodes[i].QRCode) != qrcode {
				continue
			}
			d.PendingQRCodes[i].Status = "expired"
			d.PendingQRCodes[i].ExpiredAt = expiredAt
			return
		}
	})
}

func (a *App) removePendingQRCode(qrcode string) {
	qrcode = strings.TrimSpace(qrcode)
	if qrcode == "" {
		return
	}
	_ = a.store.Update(func(d *session.Data) {
		filtered := d.PendingQRCodes[:0]
		for _, record := range d.PendingQRCodes {
			if strings.TrimSpace(record.QRCode) == qrcode {
				continue
			}
			filtered = append(filtered, record)
		}
		d.PendingQRCodes = filtered
	})
}
