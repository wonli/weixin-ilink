package chatstate

import (
	"errors"
	"sync"
	"time"
)

type MessageDirection string

const (
	Inbound  MessageDirection = "inbound"
	Outbound MessageDirection = "outbound"
)

type Message struct {
	Direction MessageDirection `json:"direction"`
	UserID    string           `json:"user_id"`
	Text      string           `json:"text"`
	At        time.Time        `json:"at"`
}

type DraftStatus string

const (
	DraftGenerating DraftStatus = "generating"
	DraftSending    DraftStatus = "sending"
	DraftFailed     DraftStatus = "failed"
)

type DraftMessage struct {
	Direction MessageDirection `json:"direction"`
	UserID    string           `json:"user_id"`
	Text      string           `json:"text"`
	Status    DraftStatus      `json:"status"`
	StartedAt time.Time        `json:"started_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type Contact struct {
	UserID        string    `json:"user_id"`
	RemarkName    string    `json:"remark_name,omitempty"`
	ContextToken  string    `json:"-"`
	LastMessageAt time.Time `json:"last_message_at"`
	CanSend       bool      `json:"can_send"`
}

type LoginState struct {
	InProgress bool   `json:"in_progress"`
	QRCode     string `json:"qrcode"`
	QRCodeURL  string `json:"qrcode_url"`
	Error      string `json:"error"`
}

type Snapshot struct {
	LoggedIn           bool          `json:"logged_in"`
	AccountID          string        `json:"account_id"`
	BaseURL            string        `json:"base_url"`
	UserID             string        `json:"user_id"`
	Login              LoginState    `json:"login"`
	Contacts           []Contact     `json:"contacts"`
	SelectedUserID     string        `json:"selected_user_id"`
	Messages           []Message     `json:"messages"`
	DraftMessage       *DraftMessage `json:"draft_message"`
	LastError          string        `json:"last_error"`
	CanSend            bool          `json:"can_send"`
	SendDisabledReason string        `json:"send_disabled_reason"`
}

type State struct {
	mu             sync.RWMutex
	login          LoginState
	loggedIn       bool
	accountID      string
	baseURL        string
	userID         string
	selectedUserID string
	lastError      string
	contacts       map[string]*Contact
	order          []string
	messages       map[string][]Message
	drafts         map[string]*DraftMessage
}

type PersistedContact struct {
	UserID        string
	RemarkName    string
	ContextToken  string
	LastMessageAt time.Time
}

func New() *State {
	return &State{
		contacts: map[string]*Contact{},
		messages: map[string][]Message{},
		drafts:   map[string]*DraftMessage{},
	}
}

func (s *State) SetSession(accountID, baseURL, userID string, loggedIn bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountID = accountID
	s.baseURL = baseURL
	s.userID = userID
	s.loggedIn = loggedIn
}

func (s *State) SetLogin(in LoginState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.login = in
}

func (s *State) SetError(err string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = err
}

func (s *State) AddInbound(userID, contextToken, text string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	contact, ok := s.contacts[userID]
	if !ok {
		contact = &Contact{UserID: userID}
		s.contacts[userID] = contact
		s.order = append(s.order, userID)
	}
	contact.ContextToken = contextToken
	contact.LastMessageAt = at
	contact.CanSend = contextToken != ""
	s.messages[userID] = append(s.messages[userID], Message{
		Direction: Inbound,
		UserID:    userID,
		Text:      text,
		At:        at,
	})
	if s.selectedUserID == "" {
		s.selectedUserID = userID
	}
}

func (s *State) AddOutbound(userID, text string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.drafts, userID)
	s.messages[userID] = append(s.messages[userID], Message{
		Direction: Outbound,
		UserID:    userID,
		Text:      text,
		At:        at,
	})
}

func (s *State) StartOutboundDraft(userID string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.drafts[userID] = &DraftMessage{
		Direction: Outbound,
		UserID:    userID,
		Status:    DraftGenerating,
		StartedAt: at,
		UpdatedAt: at,
	}
}

func (s *State) AppendOutboundDraft(userID, delta string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.drafts[userID]
	if !ok {
		draft = &DraftMessage{
			Direction: Outbound,
			UserID:    userID,
			Status:    DraftGenerating,
			StartedAt: at,
		}
		s.drafts[userID] = draft
	}
	draft.Text += delta
	draft.Status = DraftGenerating
	draft.UpdatedAt = at
}

func (s *State) MarkOutboundDraftSending(userID string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if draft, ok := s.drafts[userID]; ok {
		draft.Status = DraftSending
		draft.UpdatedAt = at
	}
}

func (s *State) FailOutboundDraft(userID, text string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.drafts[userID]
	if !ok {
		draft = &DraftMessage{
			Direction: Outbound,
			UserID:    userID,
			StartedAt: at,
		}
		s.drafts[userID] = draft
	}
	if text != "" {
		draft.Text = text
	}
	draft.Status = DraftFailed
	draft.UpdatedAt = at
}

func (s *State) ClearOutboundDraft(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.drafts, userID)
}

func (s *State) SelectUser(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.contacts[userID]; ok {
		s.selectedUserID = userID
	}
}

func (s *State) SelectedContact() (Contact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	contact, ok := s.contacts[s.selectedUserID]
	if !ok {
		return Contact{}, false
	}
	return *contact, true
}

func (s *State) Contact(userID string) (Contact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	contact, ok := s.contacts[userID]
	if !ok {
		return Contact{}, false
	}
	return *contact, true
}

func (s *State) ContextToken(userID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	contact, ok := s.contacts[userID]
	if !ok {
		return "", errors.New("contact not found")
	}
	if contact.ContextToken == "" {
		return "", errors.New("context token missing")
	}
	return contact.ContextToken, nil
}

func (s *State) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	contacts := make([]Contact, 0, len(s.order))
	for _, id := range s.order {
		if c, ok := s.contacts[id]; ok {
			contacts = append(contacts, Contact{
				UserID:        c.UserID,
				RemarkName:    c.RemarkName,
				LastMessageAt: c.LastMessageAt,
				CanSend:       c.CanSend,
			})
		}
	}
	msgs := append([]Message{}, s.messages[s.selectedUserID]...)
	var draft *DraftMessage
	if current, ok := s.drafts[s.selectedUserID]; ok {
		copyDraft := *current
		draft = &copyDraft
	}
	canSend := false
	sendDisabledReason := "请先选择一个已建立会话的联系人"
	if s.selectedUserID != "" {
		if c, ok := s.contacts[s.selectedUserID]; ok {
			if c.ContextToken != "" {
				canSend = true
				sendDisabledReason = ""
			} else {
				sendDisabledReason = "当前联系人缺少 context_token，不能主动发送"
			}
		} else {
			sendDisabledReason = "当前联系人不存在"
		}
	}
	if !s.loggedIn {
		canSend = false
		sendDisabledReason = "当前机器人未登录"
	}
	return Snapshot{
		LoggedIn:           s.loggedIn,
		AccountID:          s.accountID,
		BaseURL:            s.baseURL,
		UserID:             s.userID,
		Login:              s.login,
		Contacts:           contacts,
		SelectedUserID:     s.selectedUserID,
		Messages:           msgs,
		DraftMessage:       draft,
		LastError:          s.lastError,
		CanSend:            canSend,
		SendDisabledReason: sendDisabledReason,
	}
}

func (s *State) MessagesForUser(userID string, limit int) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs := append([]Message{}, s.messages[userID]...)
	if limit > 0 && len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	return msgs
}

func (s *State) ExportContacts() ([]PersistedContact, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PersistedContact, 0, len(s.order))
	for _, id := range s.order {
		if c, ok := s.contacts[id]; ok {
			out = append(out, PersistedContact{
				UserID:        c.UserID,
				RemarkName:    c.RemarkName,
				ContextToken:  c.ContextToken,
				LastMessageAt: c.LastMessageAt,
			})
		}
	}
	return out, s.selectedUserID
}

func (s *State) RestoreContacts(contacts []PersistedContact, selectedUserID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contacts = map[string]*Contact{}
	s.order = nil
	for _, item := range contacts {
		if item.UserID == "" {
			continue
		}
		s.contacts[item.UserID] = &Contact{
			UserID:        item.UserID,
			RemarkName:    item.RemarkName,
			ContextToken:  item.ContextToken,
			LastMessageAt: item.LastMessageAt,
			CanSend:       item.ContextToken != "",
		}
		s.order = append(s.order, item.UserID)
	}
	if _, ok := s.contacts[selectedUserID]; ok {
		s.selectedUserID = selectedUserID
	}
}

func (s *State) SetContactRemark(userID, remarkName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	contact, ok := s.contacts[userID]
	if !ok {
		return false
	}
	contact.RemarkName = remarkName
	return true
}
