package main

import (
	"net/http"
	"payment-gw/gateway"
	"testing"

	jsonvalue "github.com/Andrew-M-C/go.jsonvalue"
	"github.com/stretchr/testify/assert"
)

func sendVoidRequest(merchantId, paymentId, secretKey string) (responseCode int, errorMessage, availableToCapture, availableToRefund string) {
	req, _ := http.NewRequest(http.MethodPost, "/merchant/"+merchantId+"/void/"+paymentId, nil)
	req.Header.Set("Authorization", secretKey)
	response := executeRequest(req)
	j, _ := jsonvalue.Unmarshal(response.Body.Bytes())

	errorMessage, _ = j.GetString("error")
	availableToCapture, _ = j.GetString("available_to_capture")
	availableToRefund, _ = j.GetString("available_to_refund")
	responseCode = response.Code

	return
}

func Test_Void(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)
	responseCode, errorMessage, availableToCapture, availableToRefund := sendVoidRequest(merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "0.00", availableToRefund)
	assert.Equal(t, "", errorMessage)
	assert.Equal(t, http.StatusOK, responseCode)
}

func Test_VoidAfterCapture(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)
	_, _, availableToCapture, availableToRefund := sendCaptureRequest("50.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "49.00", availableToCapture)
	assert.Equal(t, "50.00", availableToRefund)

	responseCode, errorMessage, availableToCapture, availableToRefund := sendVoidRequest(merchantId, paymentId, secretKey)

	assert.Equal(t, "49.00", availableToCapture)
	assert.Equal(t, "50.00", availableToRefund)
	assert.Equal(t, gateway.ErrAlreadyCaptured.Error(), errorMessage)
	assert.Equal(t, http.StatusBadRequest, responseCode)
}

func Test_VoidAfterVoided(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)
	responseCode, errorMessage, availableToCapture, availableToRefund := sendVoidRequest(merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "0.00", availableToRefund)
	assert.Equal(t, "", errorMessage)
	assert.Equal(t, http.StatusOK, responseCode)

	responseCode, errorMessage, availableToCapture, availableToRefund = sendVoidRequest(merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "0.00", availableToRefund)
	assert.Equal(t, gateway.ErrAlreadyVoided.Error(), errorMessage)
	assert.Equal(t, http.StatusBadRequest, responseCode)
}

func Test_VoidAfterRefund(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)
	_, _, availableToCapture, availableToRefund := sendCaptureRequest("50.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "49.00", availableToCapture)
	assert.Equal(t, "50.00", availableToRefund)

	_, _, availableToCapture, availableToRefund = sendRefundRequest("10.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "40.00", availableToRefund)

	responseCode, errorMessage, availableToCapture, availableToRefund := sendVoidRequest(merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "40.00", availableToRefund)
	assert.Equal(t, gateway.ErrAlreadyRefunded.Error(), errorMessage)
	assert.Equal(t, http.StatusBadRequest, responseCode)
}
