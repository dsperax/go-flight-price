package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sperax/flight-price-service/internal/auth"
)

const (
	testSecret   = "test-secret"
	testUsername = "admin"
	testPassword = "admin123"
	testExpiry   = 60
)

// --- JWT unit tests ---

func TestGenerateToken_Valid(t *testing.T) {
	token, err := auth.GenerateToken(testSecret, testExpiry)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestValidateToken_Valid(t *testing.T) {
	token, _ := auth.GenerateToken(testSecret, testExpiry)
	if err := auth.ValidateToken(token, testSecret); err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, _ := auth.GenerateToken(testSecret, testExpiry)
	if err := auth.ValidateToken(token, "wrong-secret"); err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	// expiration of 0 minutes results in an already-expired token
	token, _ := auth.GenerateToken(testSecret, 0)
	if err := auth.ValidateToken(token, testSecret); err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	if err := auth.ValidateToken("not.a.valid.token", testSecret); err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}

// --- Handler unit tests ---

func newHandler() *auth.Handler {
	return auth.NewHandler(testUsername, testPassword, testSecret, testExpiry)
}

func TestLogin_Success(t *testing.T) {
	h := newHandler()
	body := `{"username":"admin","password":"admin123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["access_token"] == "" {
		t.Fatal("expected access_token in response")
	}
	if resp["token_type"] != "Bearer" {
		t.Fatalf("expected token_type Bearer, got %v", resp["token_type"])
	}
}

func TestLogin_WrongCredentials(t *testing.T) {
	h := newHandler()
	body := `{"username":"admin","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLogin_InvalidBody(t *testing.T) {
	h := newHandler()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- Middleware unit tests ---

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestMiddleware_MissingToken(t *testing.T) {
	mw := auth.Middleware(testSecret)
	handler := mw(http.HandlerFunc(dummyHandler))

	req := httptest.NewRequest(http.MethodGet, "/flights/search", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_InvalidFormat(t *testing.T) {
	mw := auth.Middleware(testSecret)
	handler := mw(http.HandlerFunc(dummyHandler))

	req := httptest.NewRequest(http.MethodGet, "/flights/search", nil)
	req.Header.Set("Authorization", "InvalidFormatToken")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	mw := auth.Middleware(testSecret)
	handler := mw(http.HandlerFunc(dummyHandler))

	req := httptest.NewRequest(http.MethodGet, "/flights/search", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_ValidToken(t *testing.T) {
	mw := auth.Middleware(testSecret)
	handler := mw(http.HandlerFunc(dummyHandler))

	token, _ := auth.GenerateToken(testSecret, testExpiry)
	req := httptest.NewRequest(http.MethodGet, "/flights/search", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
