package httpapi

import (
	"errors"
	"net/http"
	"strings"
)

type contextKey string

const (
	contextKeyUserID contextKey = "userID"
)

func keyByDeviceID(r *http.Request) (string, error) {
	deviceID := strings.TrimSpace(r.Header.Get("X-Device-Id"))
	if deviceID == "" {
		return "", errors.New("missing device id")
	}
	return "device:" + deviceID, nil
}
