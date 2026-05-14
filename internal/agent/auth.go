package agent

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	headerNode      = "X-Wol-Relay-Node"
	headerTimestamp = "X-Wol-Relay-Timestamp"
	headerSignature = "X-Wol-Relay-Signature"
)

func signRequest(req *http.Request, nodeName, secret string, body []byte) {
	now := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set(headerNode, nodeName)
	req.Header.Set(headerTimestamp, now)
	req.Header.Set(headerSignature, signature(secret, now, body))
}

func verifyRequest(req *http.Request, secret string, body []byte) error {
	if secret == "" {
		return nil
	}
	ts := req.Header.Get(headerTimestamp)
	sig := req.Header.Get(headerSignature)
	if ts == "" || sig == "" {
		return errors.New("missing authentication headers")
	}
	unix, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}
	if delta := time.Since(time.Unix(unix, 0)); delta > 5*time.Minute || delta < -5*time.Minute {
		return errors.New("timestamp is outside allowed clock skew")
	}
	want := signature(secret, ts, body)
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(sig)), []byte(want)) != 1 {
		return errors.New("invalid signature")
	}
	return nil
}

func signature(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("\n"))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
