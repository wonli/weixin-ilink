package app

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/wonli/weixin-ilink/internal/ai"
	"github.com/wonli/weixin-ilink/internal/ilinkapi"
	"github.com/wonli/weixin-ilink/internal/session"
)

func (a *App) startPollLoop(accountID string) {
	a.mu.Lock()
	rt := a.runtimes[accountID]
	if rt == nil {
		a.mu.Unlock()
		return
	}
	if rt.cancel != nil {
		rt.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	rt.cancel = cancel
	a.mu.Unlock()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			a.mu.RLock()
			current := a.runtimes[accountID]
			if current == nil {
				a.mu.RUnlock()
				return
			}
			record := current.record
			state := current.state
			a.mu.RUnlock()

			if strings.TrimSpace(record.Token) == "" {
				time.Sleep(time.Second)
				continue
			}
			baseURL := record.BaseURL
			if baseURL == "" {
				baseURL = ilinkapi.DefaultBaseURL
			}
			resp, err := a.client.GetUpdates(ctx, baseURL, record.Token, record.GetUpdatesBuf)
			if err != nil {
				if isBenignCancel(err, ctx) {
					return
				}
				state.SetError(err.Error())
				a.updateAccountRequest(accountID, "getupdates", "error", err.Error(), 0)
				a.updateAccountStatus(accountID, "error", err.Error(), 0, false, "")
				time.Sleep(2 * time.Second)
				continue
			}
			a.updateAccountRequest(accountID, "getupdates", "ok", "", resp.ErrCode)
			a.markPollOK(accountID)
			if resp.ErrCode == ilinkapi.SessionExpiredCode || resp.Ret == ilinkapi.SessionExpiredCode {
				state.SetError("session expired, please login again")
				a.updateAccountRequest(accountID, "getupdates", "expired", "session expired, please login again", ilinkapi.SessionExpiredCode)
				state.SetSession(record.AccountID, baseURL, record.UserID, false)
				a.updateAccountStatus(accountID, "expired", "session expired, please login again", ilinkapi.SessionExpiredCode, true, "")
				return
			}
			if resp.ErrCode != 0 || resp.ErrMsg != "" || resp.Ret != 0 {
				a.updateAccountRequest(accountID, "getupdates", "api_error", firstNonEmpty(resp.ErrMsg, "api error"), resp.ErrCode)
				a.updateAccountStatus(accountID, "api_error", firstNonEmpty(resp.ErrMsg, "api error"), resp.ErrCode, false, "")
			}
			a.persistGetUpdatesBuf(accountID, &record, resp.GetUpdatesBuf)
			for _, msg := range resp.Msgs {
				if msg.MessageType != ilinkapi.MessageTypeUser {
					continue
				}
				if raw, err := json.Marshal(msg); err == nil {
					log.Printf("weixin inbound raw account=%s from=%s context_token=%s message=%s", accountID, msg.FromUserID, msg.ContextToken, string(raw))
				} else {
					log.Printf("weixin inbound raw account=%s from=%s context_token=%s marshal_error=%v", accountID, msg.FromUserID, msg.ContextToken, err)
				}
				go a.captureInboundMedia(accountID, msg)
				inbound := extractInboundMessage(msg)
				if !inbound.HasContent {
					continue
				}
				state.AddInbound(msg.FromUserID, msg.ContextToken, inbound.HistoryText, time.Now())
				a.markInbound(accountID)
				a.persistContacts(accountID)
				if strings.TrimSpace(msg.ContextToken) == "" {
					state.SetError("received message without context_token")
					continue
				}
				go a.handleInboundMessage(accountID, msg, inbound)
			}
		}
	}()
}

func (a *App) persistGetUpdatesBuf(accountID string, record *session.AccountRecord, getUpdatesBuf string) {
	if getUpdatesBuf == "" || getUpdatesBuf == record.GetUpdatesBuf {
		return
	}
	record.GetUpdatesBuf = getUpdatesBuf
	a.mu.Lock()
	if current := a.runtimes[accountID]; current != nil {
		current.record.GetUpdatesBuf = getUpdatesBuf
	}
	a.mu.Unlock()
	_ = a.store.Update(func(d *session.Data) {
		for i := range d.Accounts {
			if d.Accounts[i].AccountID == accountID {
				d.Accounts[i].GetUpdatesBuf = getUpdatesBuf
				return
			}
		}
	})
}

func (a *App) handleInboundMessage(accountID string, msg ilinkapi.WeixinMessage, inbound inboundMessage) {
	if strings.TrimSpace(inbound.PlainText) == "二维码" {
		if err := a.sendShareQRCode(accountID, msg.FromUserID, msg.ContextToken); err != nil {
			if rt := a.runtimeByID(accountID); rt != nil {
				rt.state.SetError(err.Error())
				rt.state.FailOutboundDraft(msg.FromUserID, "发送二维码失败: "+err.Error(), time.Now())
			}
			a.markSendError(accountID, err.Error())
			a.updateAccountRequest(accountID, "send_qrcode", "error", err.Error(), 0)
		}
		return
	}
	a.replyWithAI(accountID, msg.FromUserID, msg.ContextToken, msg, inbound)
}

func (a *App) replyWithAI(accountID, userID, contextToken string, msg ilinkapi.WeixinMessage, inbound inboundMessage) {
	rt := a.runtimeByID(accountID)
	if rt == nil {
		return
	}
	startedAt := time.Now()
	rt.state.StartOutboundDraft(userID, startedAt)

	history := rt.state.MessagesForUser(userID, 8)
	promptCtx := ai.PromptContext{
		Now:                 startedAt,
		AccountID:           rt.record.AccountID,
		BotUserID:           rt.record.UserID,
		BaseURL:             firstNonEmpty(rt.record.BaseURL, ilinkapi.DefaultBaseURL),
		ContactUserID:       userID,
		InboundMessageKinds: messageKinds(msg),
	}
	for _, msg := range history {
		role := "assistant"
		if msg.Direction == "inbound" {
			role = "user"
		}
		promptCtx.ConversationHistory = append(promptCtx.ConversationHistory, ai.HistoryTurn{
			Role: role,
			Text: msg.Text,
			At:   msg.At,
		})
	}

	reply, err := ai.ReplyStream(a.aiCfg, promptCtx, inbound.UserPrompt, func(delta string) {
		if delta == "" {
			return
		}
		if current := a.runtimeByID(accountID); current != nil {
			current.state.AppendOutboundDraft(userID, delta, time.Now())
		}
	})
	if err != nil {
		a.markSendError(accountID, err.Error())
		a.updateAccountRequest(accountID, "ai_reply", "error", err.Error(), 0)
		rt.state.SetError(err.Error())
		rt.state.FailOutboundDraft(userID, "生成失败: "+err.Error(), time.Now())
		return
	}
	if strings.TrimSpace(reply) == "" {
		rt.state.FailOutboundDraft(userID, "生成结果为空", time.Now())
		a.updateAccountRequest(accountID, "ai_reply", "empty", "ai reply is empty", 0)
		return
	}

	rt.state.MarkOutboundDraftSending(userID, time.Now())
	baseURL := rt.record.BaseURL
	if baseURL == "" {
		baseURL = ilinkapi.DefaultBaseURL
	}
	if strings.TrimSpace(rt.record.Token) == "" {
		err := "not logged in"
		rt.state.SetError(err)
		rt.state.FailOutboundDraft(userID, "发送失败: "+err, time.Now())
		a.markSendError(accountID, err)
		a.updateAccountRequest(accountID, "sendmessage", "error", err, 0)
		return
	}
	if err := a.client.SendText(context.Background(), baseURL, rt.record.Token, userID, contextToken, reply); err != nil {
		rt.state.SetError(err.Error())
		rt.state.FailOutboundDraft(userID, reply+"\n\n发送失败: "+err.Error(), time.Now())
		a.markSendError(accountID, err.Error())
		a.updateAccountRequest(accountID, "sendmessage", "error", err.Error(), 0)
		return
	}

	a.updateAccountRequest(accountID, "sendmessage", "ok", "", 0)
	rt.state.AddOutbound(userID, reply, time.Now())
	a.markSendOK(accountID)
	a.persistContacts(accountID)
}
