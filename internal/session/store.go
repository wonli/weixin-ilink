package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type ContactRecord struct {
	UserID        string `json:"user_id"`
	RemarkName    string `json:"remark_name,omitempty"`
	ContextToken  string `json:"context_token"`
	LastMessageAt string `json:"last_message_at"`
}

type AccountRecord struct {
	AccountID            string          `json:"account_id"`
	Token                string          `json:"token"`
	BaseURL              string          `json:"base_url"`
	UserID               string          `json:"user_id"`
	CreatedAt            string          `json:"created_at"`
	LastLoginAt          string          `json:"last_login_confirmed_at"`
	GetUpdatesBuf        string          `json:"get_updates_buf"`
	SelectedUserID       string          `json:"selected_user_id"`
	Contacts             []ContactRecord `json:"contacts"`
	Status               string          `json:"status"`
	LastError            string          `json:"last_error"`
	LastErrorCode        int             `json:"last_error_code"`
	LastErrorAt          string          `json:"last_error_at"`
	LastPollOKAt         string          `json:"last_poll_ok_at"`
	LastInboundAt        string          `json:"last_inbound_at"`
	LastSendOKAt         string          `json:"last_send_ok_at"`
	LastSendError        string          `json:"last_send_error"`
	LastSendErrorAt      string          `json:"last_send_error_at"`
	SendDisabledReason   string          `json:"send_disabled_reason"`
	LastRequestAction    string          `json:"last_request_action"`
	LastRequestResult    string          `json:"last_request_result"`
	LastRequestError     string          `json:"last_request_error"`
	LastRequestErrorCode int             `json:"last_request_error_code"`
	LastRequestAt        string          `json:"last_request_at"`
	StatusAt             string          `json:"status_at"`
	ReplacedBy           string          `json:"replaced_by"`
}

type PendingQRCodeRecord struct {
	QRCode          string `json:"qrcode"`
	SourceAccountID string `json:"source_account_id"`
	SharedToUserID  string `json:"shared_to_user_id"`
	CreatedAt       string `json:"created_at"`
	Status          string `json:"status,omitempty"`
	ExpiredAt       string `json:"expired_at,omitempty"`
}

type Data struct {
	ActiveAccountID  string                `json:"active_account_id"`
	LastLoginAction  string                `json:"last_login_action"`
	LastLoginResult  string                `json:"last_login_result"`
	LastLoginError   string                `json:"last_login_error"`
	LastLoginErrorAt string                `json:"last_login_error_at"`
	LastLoginOKAt    string                `json:"last_login_ok_at"`
	Accounts         []AccountRecord       `json:"accounts"`
	PendingQRCodes   []PendingQRCodeRecord `json:"pending_qr_codes,omitempty"`
}

type Store struct {
	path string
	mu   sync.RWMutex
	data Data
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return s, nil
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var data Data
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	s.data = data
	return nil
}

func (s *Store) Save() error {
	s.mu.RLock()
	data := s.data
	s.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

func (s *Store) Get() Data {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

func (s *Store) Update(fn func(*Data)) error {
	s.mu.Lock()
	fn(&s.data)
	s.mu.Unlock()
	return s.Save()
}
