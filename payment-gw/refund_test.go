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

func sendRefundRequest(amount, merchantId, paymentId, secretKey string) (responseCode int, errorMessage, availableToCapture, availableToRefund string) {
	payload := []byte(fmt.Sprintf(`{"amount":"%s"}`, amount))
	req, _ := http.NewRequest(http.MethodPost, "/merchant/"+merchantId+"/refund/"+paymentId, bytes.NewBuffer(payload))
	req.Header.Set("Authorization", secretKey)
	response := executeRequest(req)
	j, _ := jsonvalue.Unmarshal(response.Body.Bytes())

	errorMessage, _ = j.GetString("error")
	availableToCapture, _ = j.GetString("available_to_capture")
	availableToRefund, _ = j.GetString("available_to_refund")
	responseCode = response.Code

	return
}

func Test_RefundfullAmount(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)
	_, _, availableToCapture, availableToRefund := sendCaptureRequest("99.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "99.00", availableToRefund)

	responseCode, errorMessage, availableToCapture, availableToRefund := sendRefundRequest("90.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "9.00", availableToRefund)
	assert.Equal(t, "", errorMessage)
	assert.Equal(t, http.StatusOK, responseCode)

	responseCode, errorMessage, availableToCapture, availableToRefund = sendRefundRequest("9.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "0.00", availableToRefund)
	assert.Equal(t, "", errorMessage)
	assert.Equal(t, http.StatusOK, responseCode)
}

func Test_RefundMoreThanCaptured(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)
	_, _, availableToCapture, availableToRefund := sendCaptureRequest("99.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "99.00", availableToRefund)

	responseCode, errorMessage, availableToCapture, availableToRefund := sendRefundRequest("90.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "9.00", availableToRefund)
	assert.Equal(t, "", errorMessage)
	assert.Equal(t, http.StatusOK, responseCode)

	responseCode, errorMessage, availableToCapture, availableToRefund = sendRefundRequest("9.01", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "9.00", availableToRefund)
	assert.Equal(t, gateway.ErrRefundToHigh.Error(), errorMessage)
	assert.Equal(t, http.StatusBadRequest, responseCode)
}

func Test_RefundMoreThanCaptured2(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)
	_, _, availableToCapture, availableToRefund := sendCaptureRequest("1.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "98.00", availableToCapture)
	assert.Equal(t, "1.00", availableToRefund)

	responseCode, errorMessage, availableToCapture, availableToRefund := sendRefundRequest("9.01", merchantId, paymentId, secretKey)

	assert.Equal(t, "98.00", availableToCapture)
	assert.Equal(t, "1.00", availableToRefund)
	assert.Equal(t, gateway.ErrRefundToHigh.Error(), errorMessage)
	assert.Equal(t, http.StatusBadRequest, responseCode)
}

func Test_RefundNotCapturedTransaction(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)

	responseCode, errorMessage, availableToCapture, availableToRefund := sendRefundRequest("90.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "99.00", availableToCapture)
	assert.Equal(t, "0.00", availableToRefund)
	assert.Equal(t, gateway.ErrNotCaptured.Error(), errorMessage)
	assert.Equal(t, http.StatusBadRequest, responseCode)
}

func Test_RefundVoidedTransaction(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)

	responseCode, errorMessage, availableToCapture, availableToRefund := sendVoidRequest(merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "0.00", availableToRefund)
	assert.Equal(t, "", errorMessage)
	assert.Equal(t, http.StatusOK, responseCode)

	responseCode, errorMessage, availableToCapture, availableToRefund = sendRefundRequest("90.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "0.00", availableToRefund)
	assert.Equal(t, gateway.ErrPaymentIsCancelled.Error(), errorMessage)
	assert.Equal(t, http.StatusBadRequest, responseCode)
}

func Test_RefundAmountZerot(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{Amount: "99.00"}, merchantId, secretKey)
	_, _, availableToCapture, availableToRefund := sendCaptureRequest("99.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "99.00", availableToRefund)

	responseCode, errorMessage, availableToCapture, availableToRefund := sendRefundRequest("0.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "99.00", availableToRefund)
	assert.Equal(t, gateway.ErrAmountIsZero.Error(), errorMessage)
	assert.Equal(t, http.StatusBadRequest, responseCode)

	responseCode, errorMessage, availableToCapture, availableToRefund = sendRefundRequest("90.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "9.00", availableToRefund)
	assert.Equal(t, "", errorMessage)
	assert.Equal(t, http.StatusOK, responseCode)
}

func Test_RefundFailureCardNumber(t *testing.T) {
	clearTable()
	merchantId, secretKey := register(t)
	_, _, paymentId, _, _ := sendAuthorizationRequest(authorizationPayload{CardNumber: refundFailureCardNumber, Amount: "99.00"}, merchantId, secretKey)
	_, _, availableToCapture, availableToRefund := sendCaptureRequest("99.00", merchantId, paymentId, secretKey)

	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "99.00", availableToRefund)

	responseCode, errorMessage, availableToCapture, availableToRefund := sendRefundRequest("90.00", merchantId, paymentId, secretKey)
	assert.Equal(t, "0.00", availableToCapture)
	assert.Equal(t, "99.00", availableToRefund)
	assert.Equal(t, gateway.ErrBasedOnCreditCardNumber.Error(), errorMessage)
	assert.Equal(t, http.StatusBadRequest, responseCode)
}
