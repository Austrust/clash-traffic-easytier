package mesh

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const authHeader = "X-Clash-Traffic-Auth"
const nonceHeader = "X-Clash-Traffic-Nonce"
const timeHeader = "X-Clash-Traffic-Time"
const responseHeader = "X-Clash-Traffic-Response"

func mac(secret, text string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

func requestMAC(secret string, r *http.Request) string {
	return mac(secret, strings.Join([]string{"request-v1", r.Method, r.Host, r.URL.RequestURI(), r.Header.Get(timeHeader), r.Header.Get(nonceHeader)}, "\n"))
}

// Bind signatures to the destination as well as the request. A discovered
// untrusted peer cannot reuse a probe's proof against a different device.
func signRequest(secret string, r *http.Request) error {
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	r.Header.Set(nonceHeader, hex.EncodeToString(nonce[:]))
	r.Header.Set(timeHeader, strconv.FormatInt(time.Now().Unix(), 10))
	r.Header.Set(authHeader, requestMAC(secret, r))
	return nil
}

func validRequest(secret string, r *http.Request) bool {
	local, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok || local.String() != r.Host {
		return false
	}
	ts, err := strconv.ParseInt(r.Header.Get(timeHeader), 10, 64)
	now := time.Now().Unix()
	if err != nil || ts < now-60 || ts > now+60 {
		return false
	}
	nonce, err := hex.DecodeString(r.Header.Get(nonceHeader))
	if err != nil || len(nonce) != 32 {
		return false
	}
	return hmac.Equal([]byte(r.Header.Get(authHeader)), []byte(requestMAC(secret, r)))
}

func responseMAC(secret string, r *http.Request, body []byte) string {
	sum := sha256.Sum256(body)
	return mac(secret, "response-v1\n"+r.Header.Get(authHeader)+"\n"+hex.EncodeToString(sum[:]))
}

func verifyResponse(secret string, r *http.Request, resp *http.Response, body []byte) error {
	if !hmac.Equal([]byte(resp.Header.Get(responseHeader)), []byte(responseMAC(secret, r, body))) {
		return errors.New("EasyTier: response authentication failed; check the sync key")
	}
	return nil
}
