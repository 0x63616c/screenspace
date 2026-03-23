package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testReportEnv struct {
	*testDB
	handler *ReportHandler
}

func newTestReportHandler(t *testing.T) *testReportEnv {
	t.Helper()
	tdb := newTestDB(t)
	h := NewReportHandler(tdb.Reports)
	return &testReportEnv{testDB: tdb, handler: h}
}

func TestReportWallpaper_Success(t *testing.T) {
	env := newTestReportHandler(t)
	u := env.createUser(t, "reporter@example.com", "user")
	wp := env.createApprovedWallpaper(t, "Report WP", "", u.ID)

	body := `{"reason":"inappropriate content"}`
	w, r := env.authRequest(t, http.MethodPost, "/wallpapers/"+wp.ID+"/report", body, u.ID, "user")
	r.SetPathValue("id", wp.ID)
	env.handler.Create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp reportResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID == "" {
		t.Fatal("expected non-empty report ID")
	}
	if resp.Reason != "inappropriate content" {
		t.Fatalf("expected reason 'inappropriate content', got %s", resp.Reason)
	}
	if resp.Status != "pending" {
		t.Fatalf("expected status 'pending', got %s", resp.Status)
	}
}

func TestReportWallpaper_MissingReason(t *testing.T) {
	env := newTestReportHandler(t)
	u := env.createUser(t, "reporter-noreason@example.com", "user")
	wp := env.createApprovedWallpaper(t, "Report NoReason WP", "", u.ID)

	body := `{}`
	w, r := env.authRequest(t, http.MethodPost, "/wallpapers/"+wp.ID+"/report", body, u.ID, "user")
	r.SetPathValue("id", wp.ID)
	env.handler.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReportWallpaper_ReasonTooLong(t *testing.T) {
	env := newTestReportHandler(t)
	u := env.createUser(t, "reporter-longreason@example.com", "user")
	wp := env.createApprovedWallpaper(t, "Report LongReason WP", "", u.ID)

	longReason := strings.Repeat("x", 501)
	body := fmt.Sprintf(`{"reason":"%s"}`, longReason)
	w, r := env.authRequest(t, http.MethodPost, "/wallpapers/"+wp.ID+"/report", body, u.ID, "user")
	r.SetPathValue("id", wp.ID)
	env.handler.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReportWallpaper_Unauthorized(t *testing.T) {
	env := newTestReportHandler(t)

	body := `{"reason":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/wallpapers/some-id/report", bytes.NewBufferString(body))
	req.SetPathValue("id", "some-id")
	w := httptest.NewRecorder()
	env.handler.Create(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
