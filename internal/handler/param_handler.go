package handler

import (
	"encoding/json"
	"net/http"
	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/service"
)

type ParamHandler struct {
	service *service.ParamService
}

func NewParamHandler(service *service.ParamService) *ParamHandler {
	return &ParamHandler{service: service}
}

func (h *ParamHandler) GetParamsByGroup(w http.ResponseWriter, r *http.Request) {
	groupParam := r.URL.Query().Get("group_param")
	if groupParam == "" {
		http.Error(w, `{"error": "group_param query parameter is required"}`, http.StatusBadRequest)
		return
	}

	params, err := h.service.GetParamsByGroup(r.Context(), groupParam)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if params == nil {
		params = []*domain.Param{}
	}
	json.NewEncoder(w).Encode(params)
}
