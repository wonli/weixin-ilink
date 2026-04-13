package app

import (
	"time"

	"github.com/wonli/weixin-ilink/internal/session"
)

func (a *App) activeRuntime() (*accountRuntime, string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.runtimes[a.activeAccountID], a.activeAccountID
}

func (a *App) runtimeByID(accountID string) *accountRuntime {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.runtimes[accountID]
}

func (a *App) persistContacts(accountID string) {
	a.mu.RLock()
	rt := a.runtimes[accountID]
	activeID := a.activeAccountID
	a.mu.RUnlock()
	if rt == nil {
		return
	}
	contacts, selectedUserID := rt.state.ExportContacts()
	snapshot := rt.state.Snapshot()
	_ = a.store.Update(func(d *session.Data) {
		d.ActiveAccountID = activeID
		for i := range d.Accounts {
			if d.Accounts[i].AccountID != accountID {
				continue
			}
			d.Accounts[i].SelectedUserID = selectedUserID
			d.Accounts[i].SendDisabledReason = snapshot.SendDisabledReason
			d.Accounts[i].Contacts = d.Accounts[i].Contacts[:0]
			for _, contact := range contacts {
				record := session.ContactRecord{
					UserID:       contact.UserID,
					RemarkName:   contact.RemarkName,
					ContextToken: contact.ContextToken,
				}
				if !contact.LastMessageAt.IsZero() {
					record.LastMessageAt = contact.LastMessageAt.Format(time.RFC3339Nano)
				}
				d.Accounts[i].Contacts = append(d.Accounts[i].Contacts, record)
			}
			return
		}
	})
}

func (a *App) updateAccountStatus(accountID, status, lastError string, lastErrorCode int, clearToken bool, replacedBy string) {
	now := time.Now().Format(time.RFC3339Nano)
	a.mu.Lock()
	if rt := a.runtimes[accountID]; rt != nil {
		rt.record.Status = status
		rt.record.LastError = lastError
		rt.record.LastErrorCode = lastErrorCode
		rt.record.LastErrorAt = now
		rt.record.StatusAt = now
		if replacedBy != "" {
			rt.record.ReplacedBy = replacedBy
		}
		if clearToken {
			rt.record.Token = ""
		}
	}
	a.mu.Unlock()

	_ = a.store.Update(func(d *session.Data) {
		for i := range d.Accounts {
			if d.Accounts[i].AccountID != accountID {
				continue
			}
			d.Accounts[i].Status = status
			d.Accounts[i].LastError = lastError
			d.Accounts[i].LastErrorCode = lastErrorCode
			d.Accounts[i].LastErrorAt = now
			d.Accounts[i].StatusAt = now
			if replacedBy != "" {
				d.Accounts[i].ReplacedBy = replacedBy
			}
			if clearToken {
				d.Accounts[i].Token = ""
			}
			return
		}
	})
}

func (a *App) markPollOK(accountID string) {
	now := time.Now().Format(time.RFC3339Nano)
	a.mu.Lock()
	if rt := a.runtimes[accountID]; rt != nil {
		rt.record.LastPollOKAt = now
	}
	a.mu.Unlock()
	_ = a.store.Update(func(d *session.Data) {
		for i := range d.Accounts {
			if d.Accounts[i].AccountID == accountID {
				d.Accounts[i].LastPollOKAt = now
				return
			}
		}
	})
}

func (a *App) markInbound(accountID string) {
	now := time.Now().Format(time.RFC3339Nano)
	a.mu.Lock()
	if rt := a.runtimes[accountID]; rt != nil {
		rt.record.LastInboundAt = now
	}
	a.mu.Unlock()
	_ = a.store.Update(func(d *session.Data) {
		for i := range d.Accounts {
			if d.Accounts[i].AccountID == accountID {
				d.Accounts[i].LastInboundAt = now
				return
			}
		}
	})
}

func (a *App) markSendOK(accountID string) {
	now := time.Now().Format(time.RFC3339Nano)
	a.mu.Lock()
	if rt := a.runtimes[accountID]; rt != nil {
		rt.record.LastSendOKAt = now
		rt.record.LastSendError = ""
		rt.record.LastSendErrorAt = ""
	}
	a.mu.Unlock()
	_ = a.store.Update(func(d *session.Data) {
		for i := range d.Accounts {
			if d.Accounts[i].AccountID == accountID {
				d.Accounts[i].LastSendOKAt = now
				d.Accounts[i].LastSendError = ""
				d.Accounts[i].LastSendErrorAt = ""
				return
			}
		}
	})
}

func (a *App) markSendError(accountID, message string) {
	now := time.Now().Format(time.RFC3339Nano)
	a.mu.Lock()
	if rt := a.runtimes[accountID]; rt != nil {
		rt.record.LastSendError = message
		rt.record.LastSendErrorAt = now
	}
	a.mu.Unlock()
	_ = a.store.Update(func(d *session.Data) {
		for i := range d.Accounts {
			if d.Accounts[i].AccountID == accountID {
				d.Accounts[i].LastSendError = message
				d.Accounts[i].LastSendErrorAt = now
				return
			}
		}
	})
}

func (a *App) updateLoginRequest(action, result, errMsg string, ok bool) {
	now := time.Now().Format(time.RFC3339Nano)
	_ = a.store.Update(func(d *session.Data) {
		d.LastLoginAction = action
		d.LastLoginResult = result
		if errMsg != "" {
			d.LastLoginError = errMsg
			d.LastLoginErrorAt = now
		}
		if ok {
			d.LastLoginOKAt = now
			if errMsg == "" {
				d.LastLoginError = ""
				d.LastLoginErrorAt = ""
			}
		}
	})
}

func (a *App) updateAccountRequest(accountID, action, result, errMsg string, errCode int) {
	now := time.Now().Format(time.RFC3339Nano)
	a.mu.Lock()
	if rt := a.runtimes[accountID]; rt != nil {
		rt.record.LastRequestAction = action
		rt.record.LastRequestResult = result
		rt.record.LastRequestError = errMsg
		rt.record.LastRequestErrorCode = errCode
		rt.record.LastRequestAt = now
	}
	a.mu.Unlock()
	_ = a.store.Update(func(d *session.Data) {
		for i := range d.Accounts {
			if d.Accounts[i].AccountID != accountID {
				continue
			}
			d.Accounts[i].LastRequestAction = action
			d.Accounts[i].LastRequestResult = result
			d.Accounts[i].LastRequestError = errMsg
			d.Accounts[i].LastRequestErrorCode = errCode
			d.Accounts[i].LastRequestAt = now
			return
		}
	})
}
