package ilinkapi

import (
	"encoding/base64"
)

func encodeUploadedAESKey(uploaded *UploadedMedia) string {
	return base64.StdEncoding.EncodeToString([]byte(uploaded.AESKeyHex))
}
