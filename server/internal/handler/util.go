package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, BadRequest("invalid id format")
	}
	return id, nil
}
