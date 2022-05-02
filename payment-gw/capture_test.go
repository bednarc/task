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

func sendCaptureRequest(amount, merchantId, paymentId, secretKey string) (responseCode int, errorMessage, availableToCapture, availableToRefund string) {
	payload := []byte(fmt.Sprintf(`{"amount":"%s"}`, amount))
	req, _ := http.NewRequest(http.MethodPost, "/merchant/"+merchantId+"/capture/"+paymentId, bytes.NewBuffer(payload))
	req.Header.Set("Authorization", secretKey)
	response := executeRequest(req)
	j, _ := jsonvalue.Unmarshal(response.Body.Bytes())

	errorMessage, _ = j.GetString("error")
	availableToCapture, _ = j.GetString("available_to_capture")
	availableToRefund, _ = j.GetString("available_to_refund")
	responseCode = response.Code

	return
}

func Test_CaptureFullAmount(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)
	_, _, availableToCapture, availableToRefund := sendCaptureRequest("99.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "99.00", availableToRefund)
}

func Test_CaptureSmallAmount(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)
	_, _, availableToCapture, availableToRefund := sendCaptureRequest("00.01", merchantId, paymentId, secretKey)

	assert.Equal(t, "98.99", availableToCapture)
	assert.Equal(t, "0.01", availableToRefund)
}

func Test_CaptureTwoTimes(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "100.00"}, merchantId, secretKey)
	sendCaptureRequest("10.00", merchantId, paymentId, secretKey)
	_, _, availableToCapture, availableToRefund := sendCaptureRequest("50.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "40.00", availableToCapture)
	assert.Equal(t, "60.00", availableToRefund)
}

func Test_CaptureMoreThanAuthorized(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "100.00"}, merchantId, secretKey)
	responseCode, errorMessage, availableToCapture, availableToRefund := sendCaptureRequest("150.00", merchantId, paymentId, secretKey)

	assert.Equal(t, http.StatusBadRequest, responseCode)
	assert.Equal(t, gateway.ErrCaptureToHigh.Error(), errorMessage)
	assert.Equal(t, "100.00", availableToCapture)
	assert.Equal(t, "0.00", availableToRefund)
}

func Test_CaptureFailureCardNumber(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)

	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{CardNumber: captureFailureCardNumber, Amount: "10.00"}, merchantId, secretKey)

	responseCode, errorMessage, availableToCapture, availableToRefund := sendCaptureRequest("10.00", merchantId, paymentId, secretKey)
	assert.Equal(t, http.StatusBadRequest, responseCode)
	assert.Equal(t, gateway.ErrBasedOnCreditCardNumber.Error(), errorMessage)
	assert.Equal(t, "10.00", availableToCapture)
	assert.Equal(t, "0.00", availableToRefund)
}

func Test_CaptureZeroAmount(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "100.00"}, merchantId, secretKey)
	responseCode, errorMessage, availableToCapture, availableToRefund := sendCaptureRequest("00.00", merchantId, paymentId, secretKey)

	assert.Equal(t, http.StatusBadRequest, responseCode)
	assert.Equal(t, gateway.ErrAmountIsZero.Error(), errorMessage)
	assert.Equal(t, "100.00", availableToCapture)
	assert.Equal(t, "0.00", availableToRefund)
}

func Test_CaptureVoidedTransaction(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)

	responseCode, errorMessage, availableToCapture, availableToRefund := sendVoidRequest(merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "0.00", availableToRefund)
	assert.Equal(t, "", errorMessage)
	assert.Equal(t, http.StatusOK, responseCode)

	responseCode, errorMessage, availableToCapture, availableToRefund = sendCaptureRequest("90.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "0.00", availableToRefund)
	assert.Equal(t, gateway.ErrPaymentIsCancelled.Error(), errorMessage)
	assert.Equal(t, http.StatusBadRequest, responseCode)
}

func Test_CaptureRefundedTransaction(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "30.00"}, merchantId, secretKey)
	sendCaptureRequest("10.00", merchantId, paymentId, secretKey)
	sendRefundRequest("5.00", merchantId, paymentId, secretKey)

	responseCode, errorMessage, availableToCapture, availableToRefund := sendCaptureRequest("10.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "5.00", availableToRefund)
	assert.Equal(t, gateway.ErrAlreadyRefunded.Error(), errorMessage)
	assert.Equal(t, http.StatusBadRequest, responseCode)
}
