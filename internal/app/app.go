package app

import (
	"context"
	"strings"
	"sync"

	"github.com/wonli/weixin-ilink/internal/ai"
	"github.com/wonli/weixin-ilink/internal/chatstate"
	"github.com/wonli/weixin-ilink/internal/ilinkapi"
	"github.com/wonli/weixin-ilink/internal/session"
)

type accountRuntime struct {
	record session.AccountRecord
	state  *chatstate.State
	cancel context.CancelFunc
}

type App struct {
	client *ilinkapi.Client
	store  *session.Store
	aiCfg  ai.Config

	mu                  sync.RWMutex
	loginCancel         context.CancelFunc
	loginSeq            uint64
	loginState          chatstate.LoginState
	runtimes            map[string]*accountRuntime
	accountOrder        []string
	activeAccountID     string
	sharedQRCodeCancels map[string]context.CancelFunc
}

type accountSummary struct {
	AccountID            string `json:"account_id"`
	UserID               string `json:"user_id"`
	BaseURL              string `json:"base_url"`
	LoggedIn             bool   `json:"logged_in"`
	Status               string `json:"status"`
	LastError            string `json:"last_error"`
	LastErrorCode        int    `json:"last_error_code"`
	LastErrorAt          string `json:"last_error_at"`
	LastPollOKAt         string `json:"last_poll_ok_at"`
	LastInboundAt        string `json:"last_inbound_at"`
	LastSendOKAt         string `json:"last_send_ok_at"`
	LastSendError        string `json:"last_send_error"`
	LastSendErrorAt      string `json:"last_send_error_at"`
	SendDisabledReason   string `json:"send_disabled_reason"`
	LastRequestAction    string `json:"last_request_action"`
	LastRequestResult    string `json:"last_request_result"`
	LastRequestError     string `json:"last_request_error"`
	LastRequestErrorCode int    `json:"last_request_error_code"`
	LastRequestAt        string `json:"last_request_at"`
	StatusAt             string `json:"status_at"`
	ReplacedBy           string `json:"replaced_by"`
	CreatedAt            string `json:"created_at"`
	LastLoginAt          string `json:"last_login_confirmed_at"`
}

type sharedQRCodeSummary struct {
	QRCode          string `json:"qrcode"`
	SourceAccountID string `json:"source_account_id"`
	SharedToUserID  string `json:"shared_to_user_id"`
	CreatedAt       string `json:"created_at"`
	Status          string `json:"status"`
	ExpiredAt       string `json:"expired_at,omitempty"`
}

type uiState struct {
	Accounts           []accountSummary        `json:"accounts"`
	PendingQRCodes     []sharedQRCodeSummary   `json:"pending_qr_codes"`
	ExpiredQRCodes     []sharedQRCodeSummary   `json:"expired_qr_codes"`
	ActiveAccountID    string                  `json:"active_account_id"`
	LoggedIn           bool                    `json:"logged_in"`
	AccountID          string                  `json:"account_id"`
	BaseURL            string                  `json:"base_url"`
	UserID             string                  `json:"user_id"`
	Login              chatstate.LoginState    `json:"login"`
	Contacts           []chatstate.Contact     `json:"contacts"`
	SelectedUserID     string                  `json:"selected_user_id"`
	Messages           []chatstate.Message     `json:"messages"`
	DraftMessage       *chatstate.DraftMessage `json:"draft_message"`
	LastError          string                  `json:"last_error"`
	CanSend            bool                    `json:"can_send"`
	SendDisabledReason string                  `json:"send_disabled_reason"`
}

func New(client *ilinkapi.Client, store *session.Store, _ *chatstate.State, aiCfg ai.Config) *App {
	app := &App{
		client:              client,
		store:               store,
		aiCfg:               aiCfg,
		runtimes:            map[string]*accountRuntime{},
		sharedQRCodeCancels: map[string]context.CancelFunc{},
	}

	data := store.Get()
	app.activeAccountID = data.ActiveAccountID
	for _, record := range data.Accounts {
		rt := &accountRuntime{
			record: record,
			state:  chatstate.New(),
		}
		restoreContacts(rt.state, record)
		baseURL := record.BaseURL
		if baseURL == "" {
			baseURL = ilinkapi.DefaultBaseURL
		}
		rt.state.SetSession(record.AccountID, baseURL, record.UserID, strings.TrimSpace(record.Token) != "")
		app.runtimes[record.AccountID] = rt
		app.accountOrder = append(app.accountOrder, record.AccountID)
		if strings.TrimSpace(record.Token) != "" {
			app.startPollLoop(record.AccountID)
		}
	}
	if app.activeAccountID == "" {
		for id := range app.runtimes {
			app.activeAccountID = id
			break
		}
	}
	app.restorePendingQRCodes(data.PendingQRCodes)
	return app
}

func (a *App) Shutdown() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.loginCancel != nil {
		a.loginCancel()
	}
	for _, cancel := range a.sharedQRCodeCancels {
		cancel()
	}
	for _, rt := range a.runtimes {
		if rt.cancel != nil {
			rt.cancel()
		}
	}
}
