package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func generateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	return key
}

func createSignedToken(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}
	return signed
}

// okHandler is a simple handler that returns 200 OK.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	claims, _ := r.Context().Value(ClaimsKey).(jwt.MapClaims)
	if claims != nil {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusOK)
	}
})

func TestDisabledJWT_Passthrough(t *testing.T) {
	m := &JWTMiddleware{enabled: false}
	handler := m.Handler(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestDisabledJWT_ClaimsInContext(t *testing.T) {
	m := &JWTMiddleware{enabled: false}

	var gotClaims jwt.MapClaims
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims, _ = r.Context().Value(ClaimsKey).(jwt.MapClaims)
		w.WriteHeader(http.StatusOK)
	})

	handler := m.Handler(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotClaims == nil {
		t.Error("expected empty claims in context, got nil")
	}
}

func TestMissingAuthHeader_Returns401(t *testing.T) {
	key := generateTestKey(t)
	m := NewJWTMiddlewareWithKey(&key.PublicKey, "", "")
	handler := m.Handler(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON error body, got: %v", err)
	}
	if body["_type"] != "error" {
		t.Fatalf("expected _type=error, got %v", body["_type"])
	}
	if body["error"] != "missing authorization header" {
		t.Fatalf("unexpected error message: %v", body["error"])
	}
}

func TestInvalidTokenFormat_Returns401(t *testing.T) {
	key := generateTestKey(t)
	m := NewJWTMiddlewareWithKey(&key.PublicKey, "", "")
	handler := m.Handler(okHandler)

	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "notabearer token"},
		{"empty bearer", "Bearer "},
		{"garbage token", "Bearer not.a.valid.jwt"},
		{"basic auth", "Basic dXNlcjpwYXNz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tt.header)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", rec.Code)
			}
		})
	}
}

func TestValidToken_Returns200(t *testing.T) {
	key := generateTestKey(t)
	m := NewJWTMiddlewareWithKey(&key.PublicKey, "", "")
	handler := m.Handler(okHandler)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"iat": jwt.NewNumericDate(time.Now()),
	}
	tokenStr := createSignedToken(t, key, claims)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestValidToken_ClaimsInContext(t *testing.T) {
	key := generateTestKey(t)
	m := NewJWTMiddlewareWithKey(&key.PublicKey, "", "")

	var gotClaims jwt.MapClaims
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims, _ = r.Context().Value(ClaimsKey).(jwt.MapClaims)
		w.WriteHeader(http.StatusOK)
	})

	handler := m.Handler(inner)

	claims := jwt.MapClaims{
		"sub": "user456",
		"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"iat": jwt.NewNumericDate(time.Now()),
	}
	tokenStr := createSignedToken(t, key, claims)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotClaims == nil {
		t.Fatal("expected claims in context, got nil")
	}
	if gotClaims["sub"] != "user456" {
		t.Errorf("expected sub=user456, got %v", gotClaims["sub"])
	}
}

func TestExpiredToken_Returns401(t *testing.T) {
	key := generateTestKey(t)
	m := NewJWTMiddlewareWithKey(&key.PublicKey, "", "")
	handler := m.Handler(okHandler)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		"iat": jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
	}
	tokenStr := createSignedToken(t, key, claims)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestNotBeforeToken_Returns401(t *testing.T) {
	key := generateTestKey(t)
	m := NewJWTMiddlewareWithKey(&key.PublicKey, "", "")
	handler := m.Handler(okHandler)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
		"nbf": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"iat": jwt.NewNumericDate(time.Now()),
	}
	tokenStr := createSignedToken(t, key, claims)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestIssuerValidation(t *testing.T) {
	key := generateTestKey(t)

	t.Run("valid issuer", func(t *testing.T) {
		m := NewJWTMiddlewareWithKey(&key.PublicKey, "https://auth.example.com", "")
		handler := m.Handler(okHandler)

		claims := jwt.MapClaims{
			"sub": "user123",
			"iss": "https://auth.example.com",
			"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		}
		tokenStr := createSignedToken(t, key, claims)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("invalid issuer", func(t *testing.T) {
		m := NewJWTMiddlewareWithKey(&key.PublicKey, "https://auth.example.com", "")
		handler := m.Handler(okHandler)

		claims := jwt.MapClaims{
			"sub": "user123",
			"iss": "https://other.example.com",
			"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		}
		tokenStr := createSignedToken(t, key, claims)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})
}

func TestAudienceValidation(t *testing.T) {
	key := generateTestKey(t)

	t.Run("valid audience", func(t *testing.T) {
		m := NewJWTMiddlewareWithKey(&key.PublicKey, "", "my-api")
		handler := m.Handler(okHandler)

		claims := jwt.MapClaims{
			"sub": "user123",
			"aud": "my-api",
			"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		}
		tokenStr := createSignedToken(t, key, claims)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("invalid audience", func(t *testing.T) {
		m := NewJWTMiddlewareWithKey(&key.PublicKey, "", "my-api")
		handler := m.Handler(okHandler)

		claims := jwt.MapClaims{
			"sub": "user123",
			"aud": "other-api",
			"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		}
		tokenStr := createSignedToken(t, key, claims)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})
}

func TestWrongSigningKey_Returns401(t *testing.T) {
	signingKey := generateTestKey(t)
	verifyKey := generateTestKey(t)

	m := NewJWTMiddlewareWithKey(&verifyKey.PublicKey, "", "")
	handler := m.Handler(okHandler)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	tokenStr := createSignedToken(t, signingKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
