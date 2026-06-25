package ctgexchange

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

// SHA256Hex returns the hex SHA-256 of a request body.
// SHA256Hex("") is the hash used for every body-less request.
func SHA256Hex(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

// RESTCanonicalString builds the four-field canonical string a REST
// signature covers:
//
//	<ts>\n<METHOD>\n<request-uri>\n<hex sha256 of body>
//
// requestURI is the path plus query string exactly as sent on the
// request line, e.g. /api/v1/me/orders/CTGUSDT?limit=50.
func RESTCanonicalString(ts int64, method, requestURI, body string) string {
	return strings.Join([]string{
		strconv.FormatInt(ts, 10),
		strings.ToUpper(method),
		requestURI,
		SHA256Hex(body),
	}, "\n")
}

func hmacHex(secret, msg string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// SignREST returns the lowercase-hex HMAC-SHA256 of the REST canonical
// string, keyed by the API key secret.
func SignREST(secret string, ts int64, method, requestURI, body string) string {
	return hmacHex(secret, RESTCanonicalString(ts, method, requestURI, body))
}

// RESTHeaders returns the three signed headers a private REST request
// must carry. ts is Unix seconds; the server rejects timestamps outside
// its signature window (default 30s), so keep the local clock in sync.
func RESTHeaders(keyID, secret, method, requestURI, body string, ts int64) map[string]string {
	return map[string]string{
		"X-API-Key":   keyID,
		"X-Timestamp": strconv.FormatInt(ts, 10),
		"X-Signature": SignREST(secret, ts, method, requestURI, body),
	}
}

// SignWSAuth returns the lowercase-hex HMAC-SHA256 over "ws-auth\n<ts>".
func SignWSAuth(secret string, ts int64) string {
	return hmacHex(secret, "ws-auth\n"+strconv.FormatInt(ts, 10))
}

// WSAuthMessage builds the signed "auth" frame to send as the first
// frame on the private WebSocket stream.
func WSAuthMessage(keyID, secret string, ts int64) map[string]any {
	return map[string]any{
		"op":   "auth",
		"args": []any{keyID, ts, SignWSAuth(secret, ts)},
	}
}
