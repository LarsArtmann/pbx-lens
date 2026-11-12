// Package oidctest provides a minimal Pocket-ID-shaped OpenID Connect
// provider for tests: a discovery document, a token endpoint issuing
// RSA-signed ID tokens, a JWKS endpoint, and a userinfo endpoint. It is a
// normal (non-_test) package so both the oidclogin suite and the web-layer
// e2e boot it against the real identity composition.
package oidctest

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
	"sync"
	"time"
)

// StubIdP is the stub provider. The IdP-side nonce is a settable field (the
// real IdP remembers it from the authorize request; the stub skips that
// hop). The token endpoint asserts the PKCE verifier's presence and reads
// the client id from HTTP Basic auth (oauth2's AuthStyleInHeader).
type StubIdP struct {
	server *httptest.Server
	key    *rsa.PrivateKey

	mu              sync.Mutex
	tokenClaims     map[string]any
	userinfoClaims  map[string]any // nil = serve tokenClaims
	nonce           string
	exchanges       int
	lastForm        url.Values
}

// NewStub starts the stub provider; Close stops it.
func NewStub(claims map[string]any) *StubIdP {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)

	idp := &StubIdP{key: key, tokenClaims: claims}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"issuer":                                idp.server.URL,
			"authorization_endpoint":                idp.server.URL + "/authorize",
			"token_endpoint":                        idp.server.URL + "/token",
			"userinfo_endpoint":                     idp.server.URL + "/userinfo",
			"jwks_uri":                              idp.server.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})

	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": keyID,
					"use": "sig",
					"n":   base64.RawURLEncoding.EncodeToString(idp.key.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString([]byte{0x01, 0x00, 0x01}), // 65537
				},
			},
		})
	})

	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		idp.mu.Lock()
		defer idp.mu.Unlock()

		idp.exchanges++

		if err := r.ParseForm(); err != nil {
			writeJSONError(w, http.StatusBadRequest, "parse form: "+err.Error())

			return
		}

		idp.lastForm = r.PostForm

		clientID, _, _ := r.BasicAuth()
		if clientID == "" {
			clientID = r.PostForm.Get("client_id")
		}

		if r.PostForm.Get("code_verifier") == "" {
			writeJSONError(w, http.StatusBadRequest, "missing code_verifier")

			return
		}

		idToken, err := signIDToken(idp.key, idp.server.URL, clientID, idp.tokenClaims, idp.nonce)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "sign: "+err.Error())

			return
		}

		writeJSON(w, map[string]any{
			"access_token": "at_" + fmt.Sprint(idp.exchanges),
			"token_type":   "Bearer",
			"id_token":     idToken,
		})
	})

	mux.HandleFunc("GET /userinfo", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			writeJSONError(w, http.StatusUnauthorized, "missing bearer")

			return
		}

		idp.mu.Lock()
		claims := idp.userinfoClaims
		if claims == nil {
			claims = idp.tokenClaims
		}
		idp.mu.Unlock()

		writeJSON(w, claims)
	})

	idp.server = httptest.NewServer(mux)

	return idp
}

// Close stops the stub server.
func (idp *StubIdP) Close() { idp.server.Close() }

// URL returns the issuer base URL (also the discovery base).
func (idp *StubIdP) URL() string { return idp.server.URL }

// SetNonce sets the nonce embedded in the next issued ID token.
func (idp *StubIdP) SetNonce(nonce string) {
	idp.mu.Lock()
	defer idp.mu.Unlock()

	idp.nonce = nonce
}

// SetTokenClaims replaces the claims embedded in issued ID tokens.
func (idp *StubIdP) SetTokenClaims(claims map[string]any) {
	idp.mu.Lock()
	defer idp.mu.Unlock()

	idp.tokenClaims = claims
}

// SetUserinfoClaims replaces the claims served at the userinfo endpoint
// (nil restores "same as the ID token").
func (idp *StubIdP) SetUserinfoClaims(claims map[string]any) {
	idp.mu.Lock()
	defer idp.mu.Unlock()

	idp.userinfoClaims = claims
}

// Exchanges returns the token-endpoint hit count.
func (idp *StubIdP) Exchanges() int {
	idp.mu.Lock()
	defer idp.mu.Unlock()

	return idp.exchanges
}

// LastForm returns the most recent token-endpoint form.
func (idp *StubIdP) LastForm() url.Values {
	idp.mu.Lock()
	defer idp.mu.Unlock()

	return idp.lastForm
}

const keyID = "test-key"

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.MarshalWrite(w, body)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	writeJSON(w, map[string]string{"error": msg})
}

// signIDToken issues a compact RS256 JWS whose payload merges the base
// claims with the protocol claims an ID token verifier requires.
func signIDToken(
	key *rsa.PrivateKey, issuer, audience string, claims map[string]any, nonce string,
) (string, error) {
	now := time.Now()
	payload := make(map[string]any, len(claims)+5)
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

	header, err := json.Marshal(map[string]string{"alg": "RS256", "kid": keyID, "typ": "JWT"})
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
