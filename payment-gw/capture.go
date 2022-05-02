package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"payment-gw/gateway"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"gopkg.in/validator.v2"
)

type captureResponse struct {
	AvailableToCapture string `json:"available_to_capture"`
	AvailableToRefund  string `json:"available_to_refund"`
	Currency           string `json:"currency,omitempty"`
	Error              string `json:"error,omitempty"`
}

func (a *App) capture(w http.ResponseWriter, r *http.Request) {
	lg := r.Context().Value("logger").(*zerolog.Logger)
	w.Header().Set("Content-Type", "application/json")
	req := struct {
		Amount string `json:"amount" validate:"regexp=^[0-9]{1\\,10}[.][0-9]{2}$"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lg.Debug().Msg(err.Error())
		respondWithError(w, http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}
	defer r.Body.Close()

	if err := validator.Validate(req); err != nil {
		lg.Debug().Msg(err.Error())
		respondWithError(w, http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	amount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		lg.Debug().Msg(err.Error())
		respondWithError(w, http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	payment, err := a.gateway.Capture(ctx, mux.Vars(r)["payment_id"], int(amount*100))
	res := createCaptureResponse(payment, err)
	if errors.Is(gateway.ErrPaymentIsCancelled, err) || errors.Is(gateway.ErrAlreadyRefunded, err) ||
		errors.Is(gateway.ErrAmountIsZero, err) || errors.Is(gateway.ErrCaptureToHigh, err) || errors.Is(gateway.ErrBasedOnCreditCardNumber, err) {
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

func createCaptureResponse(p gateway.Payment, err error) captureResponse {
	availableToRefund := float64(p.Captured-p.Refunded) / 100
	availableToCapture := float64(p.Authorized-p.Captured) / 100

	res := captureResponse{strconv.FormatFloat(availableToCapture, 'f', 2, 64), strconv.FormatFloat(availableToRefund, 'f', 2, 64), p.Currency, ""}
	if errors.Is(gateway.ErrAlreadyRefunded, err) {
		res.Error = err.Error()
		res.AvailableToCapture = "0.00"
		return res
	}
	if errors.Is(gateway.ErrPaymentIsCancelled, err) {
		res.Error = err.Error()
		res.AvailableToCapture = "0.00"
		res.AvailableToRefund = "0.00"
		return res
	}
	if errors.Is(gateway.ErrAmountIsZero, err) || errors.Is(gateway.ErrCaptureToHigh, err) || errors.Is(gateway.ErrBasedOnCreditCardNumber, err) {
		res.Error = err.Error()
		return res
	}

	return res
}
