package handler

import (
	"encoding/json"
	"net/http"
)

type categoriesResponse struct {
	Categories []string `json:"categories"`
}

func ListCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categoriesResponse{Categories: ValidCategories})
}
