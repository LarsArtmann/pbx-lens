package oidclogin

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// stubIdP is a minimal Pocket-ID-shaped OpenID Connect provider for tests:
// a discovery document, a token endpoint issuing RSA-signed ID tokens, a
// JWKS endpoint, and a userinfo endpoint. The IdP-side nonce is a settable
// field (the real IdP remembers it from the authorize request; the stub
// skips that hop). The exchange asserts the PKCE verifier's presence.
type stubIdP struct {
	server         *httptest.Server
	key            *rsa.PrivateKey
	tokenClaims    map[string]any
	userinfoClaims map[string]any // nil = serve tokenClaims
	nonce          string
	exchanges      int
	lastForm       url.Values
}

func newStubIdP(t *testing.T, claims map[string]any) *stubIdP {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	idp := &stubIdP{key: key, tokenClaims: claims}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		stubWriteJSON(w, map[string]any{
			"issuer":                                 idp.server.URL,
			"authorization_endpoint":                 idp.server.URL + "/authorize",
			"token_endpoint":                         idp.server.URL + "/token",
			"userinfo_endpoint":                      idp.server.URL + "/userinfo",
			"jwks_uri":                               idp.server.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})

	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, _ *http.Request) {
		stubWriteJSON(w, stubJWKS(idp.key))
	})

	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		idp.exchanges++

		if err := r.ParseForm(); err != nil {
			stubWriteJSONError(w, http.StatusBadRequest, "parse form: "+err.Error())

			return
		}

		idp.lastForm = r.PostForm

		// The client credentials ride HTTP Basic auth (AuthStyleInHeader).
		clientID, _, _ := r.BasicAuth()
		if clientID == "" {
			clientID = r.PostForm.Get("client_id")
		}

		if r.PostForm.Get("code_verifier") == "" {
			stubWriteJSONError(w, http.StatusBadRequest, "missing code_verifier")

			return
		}

		idToken, err := stubSignIDToken(idp.key, idp.server.URL, clientID, idp.tokenClaims, idp.nonce)
		if err != nil {
			stubWriteJSONError(w, http.StatusInternalServerError, "sign: "+err.Error())

			return
		}

		stubWriteJSON(w, map[string]any{
			"access_token": "at_" + fmt.Sprint(idp.exchanges),
			"token_type":   "Bearer",
			"id_token":     idToken,
		})
	})

	mux.HandleFunc("GET /userinfo", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			stubWriteJSONError(w, http.StatusUnauthorized, "missing bearer")

			return
		}

		claims := idp.userinfoClaims
		if claims == nil {
			claims = idp.tokenClaims
		}

		stubWriteJSON(w, claims)
	})

	idp.server = httptest.NewServer(mux)
	t.Cleanup(idp.server.Close)

	return idp
}

func stubWriteJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.MarshalWrite(w, body)
}

func stubWriteJSONError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	stubWriteJSON(w, map[string]string{"error": msg})
}

// stubJWKS renders the signing key as a single-entry JWKS document.
func stubJWKS(key *rsa.PrivateKey) map[string]any {
	return map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": stubKeyID,
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString([]byte{0x01, 0x00, 0x01}), // 65537
			},
		},
	}
}

const stubKeyID = "test-key"

// stubSignIDToken issues a compact RS256 JWS whose payload merges the base
// claims with the protocol claims an ID token verifier requires.
func stubSignIDToken(
	key *rsa.PrivateKey, issuer, audience string, claims map[string]any, nonce string,
) (string, error) {
	now := time.Now()
	payload := make(map[string]any, len(claims)+6)
	for k, v := range claims {
		payload[k] = v
	}

	payload["iss"] = issuer
	payload["aud"] = audience
	payload["iat"] = now.Unix()
	payload["exp"] = now.Add(time.Hour).Unix()
	if nonce != "" {
		payload["nonce"] = nonce
	}

	header, err := json.Marshal(map[string]string{"alg": "RS256", "kid": stubKeyID, "typ": "JWT"})
	if err != nil {
		return "", err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(body)

	sum := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// parseQuery extracts the query of an authorization URL for assertions.
func parseQuery(t *testing.T, rawURL string) url.Values {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse authorize URL: %v", err)
	}

	return parsed.Query()
}

func sha256Sum(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}
