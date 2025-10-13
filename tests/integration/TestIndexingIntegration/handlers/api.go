package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

type APIHandler struct {
	server *server.Server
}

func NewAPIHandler(srv *server.Server) *APIHandler {
	return &APIHandler{
		server: srv,
	}
}

func (h *APIHandler) HandleAPIRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"message": "GET request handled",
		"time":    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *APIHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"received": data,
		"status":   "processed",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
