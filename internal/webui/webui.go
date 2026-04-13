package webui

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
)

//go:embed all:web
var webFS embed.FS

type App interface {
	State() any
	StartLogin(http.ResponseWriter, *http.Request)
	PollLogin(http.ResponseWriter, *http.Request)
	SendMessage(http.ResponseWriter, *http.Request)
	SendImageMessage(http.ResponseWriter, *http.Request)
	UpdateContactRemark(http.ResponseWriter, *http.Request)
	SelectAccount(http.ResponseWriter, *http.Request)
	QRFrame(http.ResponseWriter, *http.Request)
	QRImage(http.ResponseWriter, *http.Request)
	Favicon(http.ResponseWriter, *http.Request)
}

func NewMux(app App) (http.Handler, error) {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, app.State())
	})
	mux.HandleFunc("/api/account/select", app.SelectAccount)
	mux.HandleFunc("/api/login/start", app.StartLogin)
	mux.HandleFunc("/api/login/poll", app.PollLogin)
	mux.HandleFunc("/api/login/qr-image", app.QRImage)
	mux.HandleFunc("/api/send", app.SendMessage)
	mux.HandleFunc("/api/send-image", app.SendImageMessage)
	mux.HandleFunc("/api/contact/remark", app.UpdateContactRemark)
	mux.HandleFunc("/qr-frame", app.QRFrame)
	mux.HandleFunc("/favicon.ico", app.Favicon)
	return mux, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
