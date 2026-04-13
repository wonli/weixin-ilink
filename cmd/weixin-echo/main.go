package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/wonli/weixin-ilink/internal/app"
	"github.com/wonli/weixin-ilink/internal/chatstate"
	"github.com/wonli/weixin-ilink/internal/config"
	"github.com/wonli/weixin-ilink/internal/ilinkapi"
	"github.com/wonli/weixin-ilink/internal/session"
	"github.com/wonli/weixin-ilink/internal/webui"
)

func main() {
	addr := flag.String("addr", ":8090", "HTTP listen address")
	sessionPath := flag.String("session", defaultSessionPath(), "session file path")
	configPath := flag.String("config", "", "config yaml path (defaults to root config.yaml / config.yml)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	store, err := session.NewStore(*sessionPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatal(err)
	}

	state := chatstate.New()
	client := ilinkapi.NewClient()
	application := app.New(client, store, state, cfg.AI)

	handler, err := webui.NewMux(application)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:    *addr,
		Handler: handler,
	}

	go func() {
		log.Printf("weixin echo bot listening on http://127.0.0.1%s using %s/%s (config: %s)", *addr, cfg.AI.Provider, cfg.AI.Model, cfg.Path)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	application.Shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func defaultSessionPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".weixin-echo-session.json"
	}
	return filepath.Join(home, ".weixin-echo", "session.json")
}
