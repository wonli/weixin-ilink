package app

import (
	"context"
	"time"

	"github.com/wonli/weixin-ilink/internal/chatstate"
	"github.com/wonli/weixin-ilink/internal/ilinkapi"
	"github.com/wonli/weixin-ilink/internal/session"
)

func (a *App) startLoginFlow() {
	a.mu.Lock()
	if a.loginCancel != nil {
		a.loginCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.loginSeq++
	loginSeq := a.loginSeq
	a.loginCancel = cancel
	a.loginState = chatstate.LoginState{InProgress: true}
	a.mu.Unlock()

	go func() {
		defer func() {
			a.mu.Lock()
			if a.loginSeq == loginSeq {
				a.loginCancel = nil
			}
			a.mu.Unlock()
		}()

		qrResp, err := a.client.StartLogin(ctx, ilinkapi.DefaultBaseURL)
		if err != nil {
			a.updateLoginRequest("get_bot_qrcode", "error", err.Error(), false)
			a.mu.Lock()
			a.loginState = chatstate.LoginState{Error: err.Error()}
			a.mu.Unlock()
			return
		}
		a.updateLoginRequest("get_bot_qrcode", "ok", "", true)

		a.mu.Lock()
		a.loginState = chatstate.LoginState{
			InProgress: true,
			QRCode:     qrResp.QRCode,
			QRCodeURL:  qrResp.QRCodeImgContent,
		}
		a.mu.Unlock()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.mu.RLock()
				login := a.loginState
				a.mu.RUnlock()
				if login.QRCode == "" {
					continue
				}
				resp, err := a.client.PollLogin(ctx, ilinkapi.DefaultBaseURL, login.QRCode)
				if err != nil {
					a.updateLoginRequest("get_qrcode_status", "error", err.Error(), false)
					a.mu.Lock()
					a.loginState.Error = err.Error()
					a.mu.Unlock()
					continue
				}
				switch resp.Status {
				case "wait", "scaned":
					a.updateLoginRequest("get_qrcode_status", resp.Status, "", false)
				case "expired":
					a.updateLoginRequest("get_qrcode_status", "expired", "", false)
					qrResp, err := a.client.StartLogin(ctx, ilinkapi.DefaultBaseURL)
					if err != nil {
						a.updateLoginRequest("get_bot_qrcode", "error", err.Error(), false)
						a.mu.Lock()
						a.loginState = chatstate.LoginState{Error: err.Error()}
						a.mu.Unlock()
						return
					}
					a.updateLoginRequest("get_bot_qrcode", "ok", "", true)
					a.mu.Lock()
					a.loginState = chatstate.LoginState{
						InProgress: true,
						QRCode:     qrResp.QRCode,
						QRCodeURL:  qrResp.QRCodeImgContent,
					}
					a.mu.Unlock()
				case "confirmed":
					a.updateLoginRequest("get_qrcode_status", "confirmed", "", true)
					a.upsertAccount(resp)
					a.mu.Lock()
					a.loginState = chatstate.LoginState{}
					a.mu.Unlock()
					return
				}
			}
		}
	}()
}

func (a *App) upsertAccount(resp *ilinkapi.QRStatusResponse) {
	baseURL := resp.BaseURL
	if baseURL == "" {
		baseURL = ilinkapi.DefaultBaseURL
	}
	now := time.Now().Format(time.RFC3339Nano)

	a.mu.Lock()
	rt := a.runtimes[resp.ILinkBotID]
	if rt == nil {
		rt = &accountRuntime{
			record: session.AccountRecord{AccountID: resp.ILinkBotID},
			state:  chatstate.New(),
		}
		a.runtimes[resp.ILinkBotID] = rt
		a.accountOrder = append(a.accountOrder, resp.ILinkBotID)
	}
	rt.record.AccountID = resp.ILinkBotID
	rt.record.Token = resp.BotToken
	rt.record.BaseURL = baseURL
	rt.record.UserID = resp.ILinkUserID
	if rt.record.CreatedAt == "" {
		rt.record.CreatedAt = now
	}
	rt.record.LastLoginAt = now
	rt.record.Status = "active"
	rt.record.LastError = ""
	rt.record.LastErrorCode = 0
	rt.record.LastErrorAt = ""
	rt.record.StatusAt = now
	rt.record.ReplacedBy = ""
	rt.state.SetSession(resp.ILinkBotID, baseURL, resp.ILinkUserID, true)

	for id, other := range a.runtimes {
		if id == resp.ILinkBotID {
			continue
		}
		if other.record.UserID == resp.ILinkUserID {
			if other.cancel != nil {
				other.cancel()
				other.cancel = nil
			}
			other.record.Status = "replaced"
			other.record.LastError = "replaced by newer bot for same wechat user"
			other.record.StatusAt = now
			other.record.LastErrorAt = now
			other.record.ReplacedBy = resp.ILinkBotID
			other.record.Token = ""
			other.state.SetSession(other.record.AccountID, firstNonEmpty(other.record.BaseURL, ilinkapi.DefaultBaseURL), other.record.UserID, false)
		}
	}
	a.activeAccountID = resp.ILinkBotID
	a.mu.Unlock()

	_ = a.store.Update(func(d *session.Data) {
		d.ActiveAccountID = resp.ILinkBotID
		for i := range d.Accounts {
			if d.Accounts[i].AccountID == resp.ILinkBotID {
				continue
			}
			if d.Accounts[i].UserID == resp.ILinkUserID {
				d.Accounts[i].Status = "replaced"
				d.Accounts[i].LastError = "replaced by newer bot for same wechat user"
				d.Accounts[i].LastErrorCode = 0
				d.Accounts[i].LastErrorAt = now
				d.Accounts[i].StatusAt = now
				d.Accounts[i].ReplacedBy = resp.ILinkBotID
				d.Accounts[i].Token = ""
			}
		}
		found := false
		for i := range d.Accounts {
			if d.Accounts[i].AccountID == resp.ILinkBotID {
				d.Accounts[i].Token = resp.BotToken
				d.Accounts[i].BaseURL = baseURL
				d.Accounts[i].UserID = resp.ILinkUserID
				if d.Accounts[i].CreatedAt == "" {
					d.Accounts[i].CreatedAt = now
				}
				d.Accounts[i].LastLoginAt = now
				d.Accounts[i].Status = "active"
				d.Accounts[i].LastError = ""
				d.Accounts[i].LastErrorCode = 0
				d.Accounts[i].LastErrorAt = ""
				d.Accounts[i].StatusAt = now
				d.Accounts[i].ReplacedBy = ""
				found = true
				break
			}
		}
		if !found {
			d.Accounts = append(d.Accounts, session.AccountRecord{
				AccountID:   resp.ILinkBotID,
				Token:       resp.BotToken,
				BaseURL:     baseURL,
				UserID:      resp.ILinkUserID,
				CreatedAt:   now,
				LastLoginAt: now,
				Status:      "active",
				StatusAt:    now,
			})
		}
	})

	a.startPollLoop(resp.ILinkBotID)
}
