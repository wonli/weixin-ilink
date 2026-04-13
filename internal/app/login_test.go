package app

import (
	"context"
	"testing"

	"github.com/wonli/weixin-ilink/internal/ai"
	"github.com/wonli/weixin-ilink/internal/ilinkapi"
	"github.com/wonli/weixin-ilink/internal/session"
)

func TestUpsertAccountReplacesOlderBotForSameUser(t *testing.T) {
	store := &session.Store{}
	app := New(ilinkapi.NewClient(), store, nil, ai.Config{})

	app.upsertAccount(&ilinkapi.QRStatusResponse{
		ILinkBotID:  "bot-old@im.bot",
		ILinkUserID: "user-1@im.wechat",
		BotToken:    "token-old",
		BaseURL:     ilinkapi.DefaultBaseURL,
	})

	app.mu.Lock()
	oldRuntime := app.runtimes["bot-old@im.bot"]
	oldCtx, oldCancel := context.WithCancel(context.Background())
	oldRuntime.cancel = oldCancel
	app.mu.Unlock()
	defer oldCancel()

	app.upsertAccount(&ilinkapi.QRStatusResponse{
		ILinkBotID:  "bot-new@im.bot",
		ILinkUserID: "user-1@im.wechat",
		BotToken:    "token-new",
		BaseURL:     ilinkapi.DefaultBaseURL,
	})

	app.mu.RLock()
	defer app.mu.RUnlock()

	if len(app.accountOrder) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(app.accountOrder))
	}
	if app.activeAccountID != "bot-new@im.bot" {
		t.Fatalf("expected active account to switch to new bot, got %q", app.activeAccountID)
	}

	replaced := app.runtimes["bot-old@im.bot"]
	if replaced == nil {
		t.Fatal("expected replaced runtime to exist")
	}
	if replaced.record.Status != "replaced" {
		t.Fatalf("expected replaced status, got %q", replaced.record.Status)
	}
	if replaced.record.ReplacedBy != "bot-new@im.bot" {
		t.Fatalf("expected replaced_by to point to new bot, got %q", replaced.record.ReplacedBy)
	}
	if replaced.record.Token != "" {
		t.Fatalf("expected replaced token to be cleared, got %q", replaced.record.Token)
	}
	if replaced.cancel != nil {
		t.Fatal("expected replaced poll loop to be cancelled")
	}
	if snapshot := replaced.state.Snapshot(); snapshot.LoggedIn {
		t.Fatal("expected replaced runtime snapshot to be logged out")
	}

	select {
	case <-oldCtx.Done():
	default:
		t.Fatal("expected old poll loop context to be cancelled")
	}
}
