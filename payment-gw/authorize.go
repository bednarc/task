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

const (
	authorizationFailureCardNumber = "4000000000000119"
	captureFailureCardNumber       = "4000000000000259"
	refundFailureCardNumber        = "4000000000003238"
)

func (a *App) authorize(w http.ResponseWriter, r *http.Request) {
	lg := r.Context().Value("logger").(*zerolog.Logger)
	w.Header().Set("Content-Type", "application/json")
	req := struct {
		NameSurname string `json:"name_surname" validate:"regexp=^[A-Za-z]{1\\,16} [A-Za-z]{1\\,16}$"`
		CardNumber  string `json:"card_number" validate:"regexp=^[0-9]{16}$"`
		ExpiryMonth string `json:"expiry_month" validate:"regexp=^[0-9]{2}$"`
		ExpiryYear  string `json:"expiry_year" validate:"regexp=^[0-9]{2}$"`
		CCV         string `json:"CCV" validate:"regexp=^[0-9]{3}$"`
		Amount      string `json:"amount" validate:"regexp=^[0-9]{1\\,10}[.][0-9]{2}$"`
		Currency    string `json:"currency" validate:"regexp=^[A-Z]{3}$"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lg.Debug().Msg(err.Error())
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	if err := validator.Validate(req); err != nil {
		lg.Debug().Msg(err.Error())
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	amount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		lg.Debug().Msg(err.Error())
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	merchantId := mux.Vars(r)["merchant_id"]
	id, err := a.gateway.Authorize(ctx, int(amount*100), req.Currency, merchantId, getMockFailure(req.CardNumber))
	if errors.Is(gateway.ErrBasedOnCreditCardNumber, err) || errors.Is(gateway.ErrAmountIsZero, err) {
		lg.Debug().Msg(err.Error())
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		lg.Error().Msg(err.Error())
		respondWithError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	res := struct {
		Id                 string `json:"payment_id"`
		AvailableToCapture string `json:"available_to_capture"`
		AvailableToRefund  string `json:"available_to_refund"`
		Currency           string `json:"currency"`
	}{id, req.Amount, "0.00", req.Currency}

	respondWithJSON(w, http.StatusOK, res)
}

func getMockFailure(cardNumber string) gateway.MockFailure {
	switch cardNumber {
	case authorizationFailureCardNumber:
		return gateway.AuthorizationFailure
	case captureFailureCardNumber:
		return gateway.CaptureFailure
	case refundFailureCardNumber:
		return gateway.RefundFailure
	default:
		return gateway.NoFailure
	}
}
