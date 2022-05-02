package main

import (
	"bytes"
	"net/http"
	"payment-gw/merchant"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_InvalidSecretKey(t *testing.T) {
	clearTable()
	merchantId, _ := register(t)
	payload := createAuthorizationPayload(authorizationPayload{})
	req, _ := http.NewRequest(http.MethodPost, "/merchant/"+merchantId+"/authorize", bytes.NewBuffer(payload))
	req.Header.Set("Authorization", "InvalidSecretKey")
	response := executeRequest(req)
	assert.Equal(t, http.StatusForbidden, response.Code)
}

func Test_ForbiddenPaymentId(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)
	sendCaptureRequest("50.00", merchantId, paymentId, secretKey)

	_, secretKey = register(t)
	responseCode, errorMessage, availableToCapture, availableToRefund := sendCaptureRequest("10.00", merchantId, paymentId, secretKey)
	assert.Equal(t, http.StatusForbidden, responseCode)
	assert.Equal(t, merchant.ErrWrongSecretKey.Error(), errorMessage)
	assert.Equal(t, "", availableToCapture)
	assert.Equal(t, "", availableToRefund)
}

func Test_InvalidMerchantId(t *testing.T) {
	clearTable()
	_, secretKey := register(t)
	responseCode, errorMessage, _, availableToCapture, availableToRefund := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, "11111222223333344444", secretKey)

	assert.Equal(t, http.StatusBadRequest, responseCode)
	assert.Equal(t, merchant.ErrMerchantNotFound.Error(), errorMessage)
	assert.Equal(t, "", availableToCapture)
	assert.Equal(t, "", availableToRefund)
}
