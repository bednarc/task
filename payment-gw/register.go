package main

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	lg := r.Context().Value("logger").(*zerolog.Logger)
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	merchantId, secretKey, err := a.merchant.Register(ctx)
	if err != nil {
		lg.Error().Msg(err.Error())
		respondWithError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	res := struct {
		MerchantId string `json:"merchant_id"`
		SecretKey  string `json:"secret_key"`
	}{merchantId, secretKey}

	respondWithJSON(w, http.StatusCreated, res)
}
