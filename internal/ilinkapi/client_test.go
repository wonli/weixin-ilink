package ilinkapi

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestContainsContextDeadlineExceeded(t *testing.T) {
	if !containsContextDeadlineExceeded(context.DeadlineExceeded) {
		t.Fatal("expected context deadline exceeded to be recognized")
	}
	if !containsContextDeadlineExceeded(errors.New("upstream: context deadline exceeded")) {
		t.Fatal("expected wrapped deadline exceeded text to be recognized")
	}
	if containsContextDeadlineExceeded(errors.New("boom")) {
		t.Fatal("did not expect unrelated error to be recognized")
	}
}

func TestValidateRemoteMediaURLRejectsPrivateIPByDefault(t *testing.T) {
	client := NewClient()
	if _, err := client.validateRemoteMediaURL("https://127.0.0.1/file.png"); err == nil {
		t.Fatal("expected private ip to be rejected")
	}
}

func TestDownloadURLBytesHonorsMaxSize(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("12345"))
	}))
	defer server.Close()
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	client := NewClient(
		WithHTTPClient(server.Client()),
		WithRemoteMediaAllowPrivateHosts(),
		WithRemoteMediaAllowedHosts(parsed.Host),
		WithRemoteMediaMaxBytes(4),
	)

	_, _, err = client.downloadURLBytes(context.Background(), server.URL)
	if !errors.Is(err, errRemoteMediaTooLarge) {
		t.Fatalf("expected errRemoteMediaTooLarge, got %v", err)
	}
}

func TestDownloadImageRejectsInvalidEncryptedKey(t *testing.T) {
	client := NewClient()
	item := &ImageItem{
		Media: &CDNMedia{
			EncryptQueryParam: "enc",
			AESKey:            base64.StdEncoding.EncodeToString([]byte("not-hex-32-bytes")),
		},
		AESKeyHex: "zz",
	}

	_, err := client.DownloadImage(context.Background(), DefaultCDNBaseURL, item)
	if err == nil {
		t.Fatal("expected invalid aes key to fail")
	}
}

func TestNewClientUsesProvidedHTTPClient(t *testing.T) {
	expectedTransport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec
	httpClient := &http.Client{Transport: expectedTransport}
	client := NewClient(WithHTTPClient(httpClient))
	if client.apiClient.HTTPClient() != httpClient {
		t.Fatal("expected custom http client to be provided by apic client")
	}
}
