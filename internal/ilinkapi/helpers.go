package ilinkapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/wonli/apic/v2"
)

func randomWechatUIN() (string, error) {
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint32(raw[:])
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", n))), nil
}

func buildHeaders(token string) (apic.Params, error) {
	uin, err := randomWechatUIN()
	if err != nil {
		return nil, err
	}
	headers := apic.Params{
		"Content-Type":      "application/json",
		"AuthorizationType": "ilink_bot_token",
		"X-WECHAT-UIN":      uin,
	}
	if strings.TrimSpace(token) != "" {
		headers["Authorization"] = "Bearer " + strings.TrimSpace(token)
	}
	return headers, nil
}

func stringsTrimRightSlash(s string) string {
	return strings.TrimRight(s, "/")
}
