package controllers

import (
	"errors"
	"net/http"

	"icctv-http-service/models"
	"icctv-http-service/services"

	"gorm.io/gorm"
)

type RTSPSourceController struct {
	service *services.RTSPSourceService
}

func NewRTSPSourceController(service *services.RTSPSourceService) *RTSPSourceController {
	return &RTSPSourceController{service: service}
}

func rtspSourceResult(w http.ResponseWriter, payload interface{}, err error, status int) {
	switch {
	case errors.Is(err, services.ErrRTSPSourceConflict), errors.Is(err, services.ErrRTSPSourceInUse):
		respondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, gorm.ErrRecordNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case err != nil:
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		respondData(w, status, payload)
	}
}

func (c *RTSPSourceController) List(w http.ResponseWriter, r *http.Request) {
	items, err := c.service.List(r.Context())
	rtspSourceResult(w, items, err, http.StatusOK)
}

func (c *RTSPSourceController) Create(w http.ResponseWriter, r *http.Request) {
	var payload models.RTSPSource
	if err := decodeJSON(r, &payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := c.service.Create(r.Context(), payload)
	rtspSourceResult(w, item, err, http.StatusCreated)
}

func (c *RTSPSourceController) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromQuery(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	var payload services.RTSPSourceUpdate
	if err := decodeJSON(r, &payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := c.service.Update(r.Context(), id, payload)
	rtspSourceResult(w, item, err, http.StatusOK)
}

func (c *RTSPSourceController) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromQuery(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	err = c.service.Delete(r.Context(), id)
	rtspSourceResult(w, map[string]bool{"deleted": true}, err, http.StatusOK)
}
