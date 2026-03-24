package respond_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/0x63616c/screenspace/server/internal/respond"
)

func init() {
	slog.SetDefault(slog.New(slog.DiscardHandler))
}

func TestJSON(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	respond.JSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}

func TestError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	respond.Error(w, http.StatusNotFound, "not_found", "wallpaper not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Errorf("expected not_found, got %q", body.Error.Code)
	}
	if body.Error.Message != "wallpaper not found" {
		t.Errorf("unexpected message: %q", body.Error.Message)
	}
}

func TestPaginated(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	items := []string{"a", "b", "c"}
	respond.Paginated(w, items, 100, 20, 0)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body struct {
		Items  []string `json:"items"`
		Total  int      `json:"total"`
		Limit  int      `json:"limit"`
		Offset int      `json:"offset"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode paginated body: %v", err)
	}
	if body.Total != 100 || body.Limit != 20 || body.Offset != 0 {
		t.Errorf("unexpected pagination: total=%d limit=%d offset=%d", body.Total, body.Limit, body.Offset)
	}
	if len(body.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(body.Items))
	}
}

func TestParsePagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      url.Values
		wantLimit  int
		wantOffset int
	}{
		{"defaults", url.Values{}, 20, 0},
		{"custom limit", url.Values{"limit": {"50"}}, 50, 0},
		{"limit capped at max", url.Values{"limit": {"200"}}, 100, 0},
		{"custom offset", url.Values{"offset": {"40"}}, 20, 40},
		{"invalid limit uses default", url.Values{"limit": {"abc"}}, 20, 0},
		{"negative offset clamped", url.Values{"offset": {"-5"}}, 20, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pg := respond.ParsePagination(tc.query, 20, 100)
			if pg.Limit != tc.wantLimit {
				t.Errorf("limit: got %d, want %d", pg.Limit, tc.wantLimit)
			}
			if pg.Offset != tc.wantOffset {
				t.Errorf("offset: got %d, want %d", pg.Offset, tc.wantOffset)
			}
		})
	}
}
