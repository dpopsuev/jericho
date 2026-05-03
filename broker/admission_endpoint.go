package broker

import (
	"encoding/json"
	"log/slog"
	"net/http"

	tangle "github.com/dpopsuev/tangle"
)

// AdmissionRequest is the HTTP body for POST /admission.
type AdmissionRequest struct {
	Role        string   `json:"role"`
	CallbackURL string   `json:"callback_url"`
	Skills      []string `json:"skills,omitempty"`
	Model       string   `json:"model,omitempty"`
}

// AdmissionResponse is the HTTP response for POST /admission.
type AdmissionResponse struct {
	EntityID uint64 `json:"entity_id"`
	Status   string `json:"status"`
}

// AdmissionHandler returns an http.HandlerFunc that admits agents via
// the Lobby. Mount on the HTTPTransport mux at POST /admission.
func AdmissionHandler(lobby *Lobby) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req AdmissionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		config := tangle.AgentConfig{
			Role:        req.Role,
			CallbackURL: req.CallbackURL,
			Skills:      req.Skills,
			Model:       req.Model,
		}

		id, err := lobby.Admit(r.Context(), config)
		if err != nil {
			slog.WarnContext(r.Context(), "admission endpoint rejected",
				slog.String(logKeyReason, req.Role),
			)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AdmissionResponse{ //nolint:errcheck // best-effort response
			EntityID: uint64(id),
			Status:   "admitted",
		})
	}
}
