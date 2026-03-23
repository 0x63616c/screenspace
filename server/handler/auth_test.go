package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterHandler_Success(t *testing.T) {
	h := newTestAuthHandler(t)

	body := `{"email":"register-success@example.com","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp authResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if resp.Role != "user" {
		t.Fatalf("expected role user, got %s", resp.Role)
	}
}

func TestRegisterHandler_MissingFields(t *testing.T) {
	h := newTestAuthHandler(t)

	body := `{"email":"","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRegisterHandler_DuplicateEmail(t *testing.T) {
	h := newTestAuthHandler(t)

	body := `{"email":"dup-handler@example.com","password":"secret123"}`

	req1 := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	w1 := httptest.NewRecorder()
	h.Register(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first register: expected 201, got %d: %s", w1.Code, w1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	w2 := httptest.NewRecorder()
	h.Register(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("second register: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestLoginHandler_Success(t *testing.T) {
	h := newTestAuthHandler(t)

	body := `{"email":"login-success@example.com","password":"secret123"}`
	regReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	regW := httptest.NewRecorder()
	h.Register(regW, regReq)
	if regW.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", regW.Code)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	loginW := httptest.NewRecorder()
	h.Login(loginW, loginReq)

	if loginW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", loginW.Code, loginW.Body.String())
	}

	var resp authResponse
	if err := json.NewDecoder(loginW.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	h := newTestAuthHandler(t)

	regBody := `{"email":"wrong-pw@example.com","password":"secret123"}`
	regReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(regBody))
	regW := httptest.NewRecorder()
	h.Register(regW, regReq)
	if regW.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", regW.Code)
	}

	loginBody := `{"email":"wrong-pw@example.com","password":"wrongpassword"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(loginBody))
	loginW := httptest.NewRecorder()
	h.Login(loginW, loginReq)

	if loginW.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", loginW.Code)
	}
}

func TestLoginHandler_NonexistentUser(t *testing.T) {
	h := newTestAuthHandler(t)

	body := `{"email":"nonexistent@example.com","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
