package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

// ClaimsKey is the context key used to store validated JWT claims.
const ClaimsKey contextKey = "claims"

// JWTMiddleware validates JWT tokens on incoming requests.
type JWTMiddleware struct {
	enabled  bool
	issuer   string
	audience string
	keyFunc  jwt.Keyfunc
}

// NewJWTMiddleware creates a JWTMiddleware from the application's JWTConfig.
// When enabled, it fetches JWKS keys from the configured URL.
func NewJWTMiddleware(enabled bool, jwksURL, issuer, audience string) (*JWTMiddleware, error) {
	m := &JWTMiddleware{
		enabled:  enabled,
		issuer:   issuer,
		audience: audience,
	}

	if enabled {
		if jwksURL == "" {
			return nil, fmt.Errorf("jwt is enabled but jwksUrl is not set")
		}
		jwks := newJWKSFetcher(jwksURL)
		m.keyFunc = jwks.keyFunc
	}

	return m, nil
}

// NewJWTMiddlewareWithKey creates a JWTMiddleware using a static RSA public key.
// This is intended for testing.
func NewJWTMiddlewareWithKey(key *rsa.PublicKey, issuer, audience string) *JWTMiddleware {
	return &JWTMiddleware{
		enabled:  true,
		issuer:   issuer,
		audience: audience,
		keyFunc: func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return key, nil
		},
	}
}

// Handler returns an http.Handler that validates the JWT on each request.
func (m *JWTMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			ctx := context.WithValue(r.Context(), ClaimsKey, jwt.MapClaims{})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSONError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeJSONError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		tokenString := parts[1]

		parserOpts := []jwt.ParserOption{
			jwt.WithValidMethods([]string{"RS256"}),
		}
		if m.issuer != "" {
			parserOpts = append(parserOpts, jwt.WithIssuer(m.issuer))
		}
		if m.audience != "" {
			parserOpts = append(parserOpts, jwt.WithAudience(m.audience))
		}

		token, err := jwt.Parse(tokenString, m.keyFunc, parserOpts...)
		if err != nil || !token.Valid {
			writeJSONError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "invalid token claims")
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"_type": "error",
		"error": message,
	})
}

// jwksFetcher fetches and caches JWKS keys from a remote URL.
type jwksFetcher struct {
	url  string
	mu   sync.RWMutex
	keys map[string]*rsa.PublicKey
	last time.Time
	ttl  time.Duration
}

func newJWKSFetcher(url string) *jwksFetcher {
	return &jwksFetcher{
		url:  url,
		keys: make(map[string]*rsa.PublicKey),
		ttl:  5 * time.Minute,
	}
}

func (f *jwksFetcher) keyFunc(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}

	kid, _ := token.Header["kid"].(string)

	f.mu.RLock()
	if time.Since(f.last) < f.ttl {
		if key, ok := f.keys[kid]; ok {
			f.mu.RUnlock()
			return key, nil
		}
	}
	f.mu.RUnlock()

	// Refresh keys
	if err := f.refresh(); err != nil {
		return nil, fmt.Errorf("fetching JWKS: %w", err)
	}

	f.mu.RLock()
	defer f.mu.RUnlock()
	key, ok := f.keys[kid]
	if !ok {
		return nil, fmt.Errorf("key %q not found in JWKS", kid)
	}
	return key, nil
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (f *jwksFetcher) refresh() error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(f.url)
	if err != nil {
		return fmt.Errorf("fetching JWKS URL: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading JWKS response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS URL returned status %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("parsing JWKS response: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := parseRSAPublicKey(k)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}

	f.mu.Lock()
	f.keys = keys
	f.last = time.Now()
	f.mu.Unlock()

	return nil
}

func parseRSAPublicKey(k jwkKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decoding modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decoding exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
