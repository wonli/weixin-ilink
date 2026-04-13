package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wonli/weixin-ilink/internal/chatstate"
	"github.com/wonli/weixin-ilink/internal/ilinkapi"
	"github.com/wonli/weixin-ilink/internal/session"
)

func (a *App) State() any {
	a.mu.RLock()
	defer a.mu.RUnlock()

	summaries := make([]accountSummary, 0, len(a.runtimes))
	pendingQRCodes := make([]sharedQRCodeSummary, 0)
	expiredQRCodes := make([]sharedQRCodeSummary, 0)
	var snapshot chatstate.Snapshot
	login := a.loginState
	storeData := a.store.Get()
	if rt := a.runtimes[a.activeAccountID]; rt != nil {
		snapshot = rt.state.Snapshot()
	}
	for _, id := range a.accountOrder {
		rt := a.runtimes[id]
		if rt == nil {
			continue
		}
		s := rt.state.Snapshot()
		summaries = append(summaries, accountSummary{
			AccountID:            rt.record.AccountID,
			UserID:               rt.record.UserID,
			BaseURL:              rt.record.BaseURL,
			LoggedIn:             s.LoggedIn,
			Status:               rt.record.Status,
			LastError:            rt.record.LastError,
			LastErrorCode:        rt.record.LastErrorCode,
			LastErrorAt:          rt.record.LastErrorAt,
			LastPollOKAt:         rt.record.LastPollOKAt,
			LastInboundAt:        rt.record.LastInboundAt,
			LastSendOKAt:         rt.record.LastSendOKAt,
			LastSendError:        rt.record.LastSendError,
			LastSendErrorAt:      rt.record.LastSendErrorAt,
			SendDisabledReason:   snapshot.SendDisabledReason,
			LastRequestAction:    rt.record.LastRequestAction,
			LastRequestResult:    rt.record.LastRequestResult,
			LastRequestError:     rt.record.LastRequestError,
			LastRequestErrorCode: rt.record.LastRequestErrorCode,
			LastRequestAt:        rt.record.LastRequestAt,
			StatusAt:             rt.record.StatusAt,
			ReplacedBy:           rt.record.ReplacedBy,
			CreatedAt:            rt.record.CreatedAt,
			LastLoginAt:          rt.record.LastLoginAt,
		})
	}
	for _, record := range storeData.PendingQRCodes {
		summary := sharedQRCodeSummary{
			QRCode:          record.QRCode,
			SourceAccountID: record.SourceAccountID,
			SharedToUserID:  record.SharedToUserID,
			CreatedAt:       record.CreatedAt,
			Status:          firstNonEmpty(strings.TrimSpace(record.Status), "pending"),
			ExpiredAt:       record.ExpiredAt,
		}
		if summary.Status == "expired" {
			expiredQRCodes = append(expiredQRCodes, summary)
			continue
		}
		pendingQRCodes = append(pendingQRCodes, summary)
	}

	return uiState{
		Accounts:           summaries,
		PendingQRCodes:     pendingQRCodes,
		ExpiredQRCodes:     expiredQRCodes,
		ActiveAccountID:    a.activeAccountID,
		LoggedIn:           snapshot.LoggedIn,
		AccountID:          snapshot.AccountID,
		BaseURL:            snapshot.BaseURL,
		UserID:             snapshot.UserID,
		Login:              login,
		Contacts:           snapshot.Contacts,
		SelectedUserID:     snapshot.SelectedUserID,
		Messages:           snapshot.Messages,
		DraftMessage:       snapshot.DraftMessage,
		LastError:          snapshot.LastError,
		CanSend:            snapshot.CanSend,
		SendDisabledReason: snapshot.SendDisabledReason,
	}
}

func (a *App) QRFrame(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "qr frame disabled", http.StatusGone)
}

func (a *App) QRImage(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	login := a.loginState
	a.mu.RUnlock()
	qrContent := strings.TrimSpace(login.QRCodeURL)
	if qrContent == "" {
		http.Error(w, "qr not ready", http.StatusNotFound)
		return
	}
	png, err := buildQRCodePosterPNG(qrContent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

func (a *App) Favicon(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) StartLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.startLoginFlow()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) PollLogin(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	login := a.loginState
	a.mu.RUnlock()
	writeJSON(w, http.StatusOK, login)
}

func (a *App) SelectAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		AccountID string `json:"account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	a.mu.Lock()
	if _, ok := a.runtimes[req.AccountID]; !ok {
		a.mu.Unlock()
		http.Error(w, "account not found", http.StatusNotFound)
		return
	}
	a.activeAccountID = req.AccountID
	a.mu.Unlock()
	_ = a.store.Update(func(d *session.Data) {
		d.ActiveAccountID = req.AccountID
	})
	a.persistContacts(req.AccountID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type sendRequest struct {
	Text       string `json:"text"`
	UserID     string `json:"user_id"`
	SelectOnly bool   `json:"select_only"`
}

type updateRemarkRequest struct {
	UserID     string `json:"user_id"`
	RemarkName string `json:"remark_name"`
}

func (a *App) SendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	rt, accountID := a.activeRuntime()
	if rt == nil {
		http.Error(w, "no active account", http.StatusBadRequest)
		return
	}
	if req.SelectOnly {
		rt.state.SelectUser(req.UserID)
		a.persistContacts(accountID)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		http.Error(w, "text required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(rt.record.Token) == "" {
		rt.state.SetError("not logged in")
		http.Error(w, "not logged in", http.StatusBadRequest)
		return
	}
	contact, ok := rt.state.SelectedContact()
	if !ok {
		http.Error(w, "select a contact first", http.StatusBadRequest)
		return
	}
	if !contact.CanSend {
		http.Error(w, "contact missing context token", http.StatusBadRequest)
		return
	}
	token, err := rt.state.ContextToken(contact.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	baseURL := rt.record.BaseURL
	if baseURL == "" {
		baseURL = ilinkapi.DefaultBaseURL
	}
	if err := a.client.SendText(r.Context(), baseURL, rt.record.Token, contact.UserID, token, text); err != nil {
		rt.state.SetError(err.Error())
		a.updateAccountRequest(accountID, "sendmessage", "error", err.Error(), 0)
		a.markSendError(accountID, err.Error())
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	a.updateAccountRequest(accountID, "sendmessage", "ok", "", 0)
	rt.state.AddOutbound(contact.UserID, text, time.Now())
	a.markSendOK(accountID)
	a.persistContacts(accountID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) SendImageMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "image required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "read image failed", http.StatusBadRequest)
		return
	}
	if len(imageData) == 0 {
		http.Error(w, "image is empty", http.StatusBadRequest)
		return
	}
	contentType := http.DetectContentType(imageData)
	if !strings.HasPrefix(contentType, "image/") {
		http.Error(w, "only image files are supported", http.StatusBadRequest)
		return
	}

	rt, accountID := a.activeRuntime()
	if rt == nil {
		http.Error(w, "no active account", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(rt.record.Token) == "" {
		rt.state.SetError("not logged in")
		http.Error(w, "not logged in", http.StatusBadRequest)
		return
	}
	contact, ok := rt.state.SelectedContact()
	if !ok {
		http.Error(w, "select a contact first", http.StatusBadRequest)
		return
	}
	if !contact.CanSend {
		http.Error(w, "contact missing context token", http.StatusBadRequest)
		return
	}
	contextToken, err := rt.state.ContextToken(contact.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	baseURL := rt.record.BaseURL
	if baseURL == "" {
		baseURL = ilinkapi.DefaultBaseURL
	}

	uploaded, err := a.client.UploadImageBuffer(r.Context(), baseURL, rt.record.Token, contact.UserID, ilinkapi.DefaultCDNBaseURL, imageData)
	if err != nil {
		rt.state.SetError(err.Error())
		a.updateAccountRequest(accountID, "sendimage", "error", err.Error(), 0)
		a.markSendError(accountID, err.Error())
		http.Error(w, fmt.Sprintf("upload image failed: %v", err), http.StatusBadGateway)
		return
	}
	if err := a.client.SendImage(r.Context(), baseURL, rt.record.Token, contact.UserID, contextToken, uploaded); err != nil {
		rt.state.SetError(err.Error())
		a.updateAccountRequest(accountID, "sendimage", "error", err.Error(), 0)
		a.markSendError(accountID, err.Error())
		http.Error(w, fmt.Sprintf("send image failed: %v", err), http.StatusBadGateway)
		return
	}

	label := "[图片已发送]"
	if name := strings.TrimSpace(header.Filename); name != "" {
		label = fmt.Sprintf("[图片已发送] %s", name)
	}
	a.updateAccountRequest(accountID, "sendimage", "ok", "", 0)
	rt.state.AddOutbound(contact.UserID, label, time.Now())
	a.markSendOK(accountID)
	a.persistContacts(accountID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) UpdateContactRemark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req updateRemarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}
	remarkName := strings.TrimSpace(req.RemarkName)

	rt, accountID := a.activeRuntime()
	if rt == nil {
		http.Error(w, "no active account", http.StatusBadRequest)
		return
	}
	if ok := rt.state.SetContactRemark(userID, remarkName); !ok {
		http.Error(w, "contact not found", http.StatusNotFound)
		return
	}
	a.persistContacts(accountID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
