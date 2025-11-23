package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/yourname/pr-reviewer/internal/apperrors"
	"github.com/yourname/pr-reviewer/internal/model"
)

func writeError(w http.ResponseWriter, httpStatus int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	resp := map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func mapServiceError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	if se, ok := err.(*apperrors.ServiceError); ok {
		switch se.Code {
		case "TEAM_EXISTS":
			writeError(w, http.StatusBadRequest, se.Code, se.Message)
		case "PR_EXISTS":
			writeError(w, http.StatusConflict, se.Code, se.Message)
		case "PR_MERGED":
			writeError(w, http.StatusConflict, se.Code, se.Message)
		case "NOT_ASSIGNED":
			writeError(w, http.StatusConflict, se.Code, se.Message)
		case "NO_CANDIDATE":
			writeError(w, http.StatusConflict, se.Code, se.Message)
		case "NOT_FOUND":
			writeError(w, http.StatusNotFound, se.Code, se.Message)
		default:
			writeError(w, http.StatusInternalServerError, se.Code, se.Message)
		}
		return
	}
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return
	}
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
