package app

import (
	"bytes"
	"os"
	"testing"

	qrcode "github.com/skip2/go-qrcode"
)

func TestBuildQRCodePosterPNGCanEncodePNG(t *testing.T) {
	content := "https://example.com/official-qrcode-content"
	if content == "" {
		t.Fatal("expected non-empty qr content")
	}

	png, err := qrcode.Encode(content, qrcode.Medium, 320)
	if err != nil {
		t.Fatalf("expected qr png to encode, got error: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("expected non-empty png bytes")
	}

	pngHeader := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	if !bytes.HasPrefix(png, pngHeader) {
		t.Fatalf("expected png header, got %x", png[:8])
	}
}

func TestWriteShareQRCodeCardPNGForDebug(t *testing.T) {
	png, err := buildQRCodePosterPNG("https://example.com/official-qrcode-content")
	if err != nil {
		t.Fatalf("build card png: %v", err)
	}

	outPath := "/tmp/weixin-share-qrcode-card-test.png"
	if err := os.WriteFile(outPath, png, 0o644); err != nil {
		t.Fatalf("write card png: %v", err)
	}
	t.Logf("wrote debug qrcode card png to %s (%d bytes)", outPath, len(png))
}
