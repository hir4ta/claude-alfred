package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/hir4ta/claude-alfred/internal/spec"
)

func (s *Server) handleGetReview(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !validateSlug(w, slug) {
		return
	}
	sd := &spec.SpecDir{ProjectPath: s.ds.ProjectPath(), TaskSlug: slug}
	if !sd.Exists() {
		writeError(w, http.StatusNotFound, "spec not found")
		return
	}
	review, err := sd.LatestReview()
	if err != nil {
		writeError(w, http.StatusNotFound, "no review found")
		return
	}
	writeJSON(w, http.StatusOK, review)
}

func (s *Server) handlePostReview(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !validateSlug(w, slug) {
		return
	}
	sd := &spec.SpecDir{ProjectPath: s.ds.ProjectPath(), TaskSlug: slug}
	if !sd.Exists() {
		writeError(w, http.StatusNotFound, "spec not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Status   string               `json:"status"`
		Comments []spec.ReviewComment `json:"comments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	status := spec.ReviewStatus(req.Status)
	if status != spec.ReviewApproved && status != spec.ReviewChangesRequested {
		writeError(w, http.StatusBadRequest, "status must be 'approved' or 'changes_requested'")
		return
	}

	review := spec.Review{
		Status:   status,
		Comments: req.Comments,
	}
	if err := sd.SaveReview(&review); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save review: "+err.Error())
		return
	}
	if err := spec.SetReviewStatus(s.ds.ProjectPath(), slug, status); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to set review status: "+err.Error())
		return
	}

	s.sse.Broadcast(SSEEvent{
		Type: "review_submitted",
		Data: map[string]any{"slug": slug, "status": string(status)},
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": string(status)})
}

func (s *Server) handleGetReviewHistory(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !validateSlug(w, slug) {
		return
	}
	sd := &spec.SpecDir{ProjectPath: s.ds.ProjectPath(), TaskSlug: slug}
	if !sd.Exists() {
		writeError(w, http.StatusNotFound, "spec not found")
		return
	}
	// LatestReview returns the most recent review; full history is via review files.
	// For now, return latest only. Full history can be added if spec package exposes it.
	review, err := sd.LatestReview()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"reviews": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reviews": []*spec.Review{review}})
}
