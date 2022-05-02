package main

import (
	"bytes"
	"fmt"
	"net/http"
	"payment-gw/gateway"
	"testing"

	jsonvalue "github.com/Andrew-M-C/go.jsonvalue"
	"github.com/stretchr/testify/assert"
)

type authorizationPayload struct {
	NameSurname string
	CardNumber  string
	ExpiryMonth string
	ExpiryYear  string
	CCV         string
	Amount      string
	Currency    string
}

func createAuthorizationPayload(p authorizationPayload) []byte {
	if p.NameSurname == "" {
		p.NameSurname = "Krystian Bednarczuk"
	}
	if p.CardNumber == "" {
		p.CardNumber = "5555555555554444"
	}
	if p.ExpiryMonth == "" {
		p.ExpiryMonth = "12"
	}
	if p.ExpiryYear == "" {
		p.ExpiryYear = "23"
	}
	if p.CCV == "" {
		p.CCV = "123"
	}
	if p.Amount == "" {
		p.Amount = "100.00"
	}
	if p.Currency == "" {
		p.Currency = "USD"
	}
	return []byte(fmt.Sprintf(`{"name_surname":"%s","card_number":"%s", "expiry_month":"%s",  "expiry_year":"%s", "CCV":"%s", "amount":"%s", "currency":"%s"}`,
		p.NameSurname, p.CardNumber, p.ExpiryMonth, p.ExpiryYear, p.CCV, p.Amount, p.Currency))
}

func sendAuthorizationRequest(pyaload authorizationPayload, merchantId, secretKey string) (responseCode int, errorMessage, paymentId, availableToCapture, availableToRefund string) {
	p := createAuthorizationPayload(pyaload)
	req, _ := http.NewRequest(http.MethodPost, "/merchant/"+merchantId+"/authorize", bytes.NewBuffer(p))
	req.Header.Set("Authorization", secretKey)
	response := executeRequest(req)

	j, _ := jsonvalue.Unmarshal(response.Body.Bytes())

	errorMessage, _ = j.GetString("error")
	paymentId, _ = j.GetString("payment_id")
	availableToCapture, _ = j.GetString("available_to_capture")
	availableToRefund, _ = j.GetString("available_to_refund")
	responseCode = response.Code
	return
}

func Test_InvalidAutorizationRequest(t *testing.T) {
	merchantId, secretKey := register(t)

	var authorizationRequestTest = []struct {
		payload []byte
	}{
		{createAuthorizationPayload(authorizationPayload{Amount: "112.999"})},
		{createAuthorizationPayload(authorizationPayload{CardNumber: "555555555a554444"})},
		{createAuthorizationPayload(authorizationPayload{ExpiryMonth: "xx"})},
		{createAuthorizationPayload(authorizationPayload{ExpiryYear: "112.999"})},
		{createAuthorizationPayload(authorizationPayload{Amount: "112.999"})},
		{createAuthorizationPayload(authorizationPayload{CCV: "XXX"})},
		{createAuthorizationPayload(authorizationPayload{Currency: "USD1"})},
		{createAuthorizationPayload(authorizationPayload{Amount: "00.00"})},
	}

	for _, tt := range authorizationRequestTest {
		req, _ := http.NewRequest(http.MethodPost, "/merchant/"+merchantId+"/authorize", bytes.NewBuffer(tt.payload))
		req.Header.Set("Authorization", secretKey)
		response := executeRequest(req)
		assert.Equal(t, http.StatusBadRequest, response.Code)
	}
}

func Test_AutorizationFailureCardNumber(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	responseCode, errorMessage, _, _, _ := sendAuthorizationRequest(authorizationPayload{CardNumber: authorizationFailureCardNumber}, merchantId, secretKey)

	assert.Equal(t, http.StatusBadRequest, responseCode)
	assert.Equal(t, gateway.ErrBasedOnCreditCardNumber.Error(), errorMessage)
}

func Test_AutorizationSuccess(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	responseCode, errorMessage, paymentId, availableToCapture, availableToRefund := sendAuthorizationRequest(authorizationPayload{Amount: "10.00"}, merchantId, secretKey)

	assert.Equal(t, http.StatusOK, responseCode)
	assert.Equal(t, "", errorMessage)
	assert.NotEmpty(t, paymentId)
	assert.Equal(t, "10.00", availableToCapture)
	assert.Equal(t, "0.00", availableToRefund)
}
