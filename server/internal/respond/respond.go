package respond

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// JSON encodes v as JSON and writes it with the given status code.
// Content-Type is set to application/json.
func JSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("respond: encode json", "error", err)
		return err
	}
	return nil
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error writes a structured JSON error response.
// Format: {"error":{"code":"...","message":"..."}}
func Error(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorBody{
		Error: errorDetail{Code: code, Message: message},
	}); err != nil {
		slog.Error("respond: encode error", "error", err)
	}
}

type paginatedResponse struct {
	Items  any `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Paginated writes a standard paginated JSON response.
// Format: {"items":[...],"total":N,"limit":N,"offset":N}
func Paginated(w http.ResponseWriter, items any, total, limit, offset int) error {
	return JSON(w, http.StatusOK, paginatedResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// Pagination holds parsed limit/offset query parameters.
type Pagination struct {
	Limit  int
	Offset int
}

// ParsePagination parses limit and offset from query params.
// Defaults: limit=defaultLimit, offset=0. Max limit=maxLimit.
func ParsePagination(q interface{ Get(string) string }, defaultLimit, maxLimit int) Pagination {
	limit := defaultLimit
	if l := q.Get("limit"); l != "" {
		var parsed int
		if _, err := fmt.Sscanf(l, "%d", &parsed); err == nil && parsed > 0 {
			if parsed > maxLimit {
				parsed = maxLimit
			}
			limit = parsed
		}
	}

	offset := 0
	if o := q.Get("offset"); o != "" {
		var parsed int
		if _, err := fmt.Sscanf(o, "%d", &parsed); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return Pagination{Limit: limit, Offset: offset}
}
