package api

import (
	"log"
	"net/http"
)

func init() {
	RegisterRouteGroup(RegisterAgentUserWebhookRoutes)
}

// RegisterAgentUserWebhookRoutes adds user-session webhook endpoints to the mux.
func RegisterAgentUserWebhookRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /agents/{agent_id}/webhook", requireProfile(handleGetAgentWebhookForUser(deps)))
	mux.Handle("PUT /agents/{agent_id}/webhook", requireProfile(handlePutAgentWebhookForUser(deps)))
	mux.Handle("DELETE /agents/{agent_id}/webhook", requireProfile(handleDeleteAgentWebhookForUser(deps)))
}

func handlePutAgentWebhookForUser(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		var req setAgentWebhookRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		resp, opErr := setAgentWebhookCore(r.Context(), deps, agentID, req.URL, req.Token, req.Provider)
		if opErr != nil {
			RespondError(w, r, opErr.Status, opErr.Body)
			return
		}
		RespondJSON(w, http.StatusOK, resp)
	}
}

func handleGetAgentWebhookForUser(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		runTest := r.URL.Query().Get("test") == "true"
		resp, opErr := getAgentWebhookCore(r.Context(), deps, agentID, runTest)
		if opErr != nil {
			log.Printf("[%s] GetAgentWebhookForUser: %v", TraceID(r.Context()), opErr)
			RespondError(w, r, opErr.Status, opErr.Body)
			return
		}
		RespondJSON(w, http.StatusOK, resp)
	}
}

func handleDeleteAgentWebhookForUser(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		if opErr := deleteAgentWebhookCore(r.Context(), deps, agentID); opErr != nil {
			RespondError(w, r, opErr.Status, opErr.Body)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]bool{"cleared": true})
	}
}
