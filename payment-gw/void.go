package main

import (
	"context"
	"net/http"
	"payment-gw/gateway"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

type voidResponse struct {
	AvailableToCapture string `json:"available_to_capture,omitempty"`
	AvailableToRefund  string `json:"available_to_refund,omitempty"`
	Currency           string `json:"currency,omitempty"`
	Error              string `json:"error,omitempty"`
}

func (a *App) void(w http.ResponseWriter, r *http.Request) {
	lg := r.Context().Value("logger").(*zerolog.Logger)
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	payment, err := a.gateway.Void(ctx, mux.Vars(r)["payment_id"])

	res := createVoidResponse(payment, err)
	if err == gateway.ErrAlreadyCaptured || err == gateway.ErrAlreadyRefunded || err == gateway.ErrAlreadyVoided {
		lg.Debug().Msg(err.Error())
		respondWithJSON(w, http.StatusBadRequest, res)
		return
	}
	if err != nil {
		lg.Error().Msg(err.Error())
		respondWithError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}
	respondWithJSON(w, http.StatusOK, res)
}

func createVoidResponse(p gateway.Payment, err error) voidResponse {
	availableToCapture_f := float64(p.Authorized-p.Captured) / 100
	availableToRefund_f := float64(p.Captured-p.Refunded) / 100
	res := voidResponse{strconv.FormatFloat(availableToCapture_f, 'f', 2, 64), strconv.FormatFloat(availableToRefund_f, 'f', 2, 64), p.Currency, ""}

	if err == gateway.ErrAlreadyCaptured {
		res.Error = err.Error()
		return res
	}

	if err == gateway.ErrAlreadyRefunded {
		res.AvailableToCapture = "0.00"
		res.Error = err.Error()
		return res
	}

	if err == gateway.ErrAlreadyVoided {
		res.AvailableToCapture = "0.00"
		res.AvailableToRefund = "0.00"
		res.Error = err.Error()
		return res
	}

	res.AvailableToCapture = "0.00"
	res.AvailableToRefund = "0.00"
	return res
}
