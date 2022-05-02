package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"payment-gw/gateway"
	"payment-gw/merchant"
	"testing"

	jsonvalue "github.com/Andrew-M-C/go.jsonvalue"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

var a App

func TestMain(m *testing.M) {
	a = App{}

	dbUsername := os.Getenv("MONGO_ROOT_USERNAME")
	dbPassword := os.Getenv("MONGO_ROOT_PASSWORD")
	dbPortNumber := os.Getenv("MONGO_PORT_NUMBER")

	c := Config{
		dbName:     "test",
		dbUsername: dbUsername,
		dbPassword: dbPassword,
		dbPort:     dbPortNumber,
	}
	a.Initialize(c)

	code := m.Run()
	os.Exit(code)
}

func clearTable() {
	a.collection(merchant.MerchantCol).DeleteMany(context.Background(), bson.D{})
	a.collection(gateway.PaymentsCol).DeleteMany(context.Background(), bson.D{})
}

func register(t *testing.T) (string, string) {
	req, _ := http.NewRequest(http.MethodPost, "/merchant/register", nil)
	response := executeRequest(req)
	assert.Equal(t, http.StatusCreated, response.Code)

	j, err := jsonvalue.Unmarshal(response.Body.Bytes())
	assert.NoError(t, err)

	secretKey, _ := j.GetString("secret_key")
	assert.NotEqual(t, len(secretKey), 0)

	merchantId, _ := j.GetString("merchant_id")
	assert.NotEqual(t, len(merchantId), 0)
	return merchantId, secretKey
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	a.router.ServeHTTP(rr, req)
	return rr
}

func makeErrorResponse(err error) string {
	return `{"error":"` + err.Error() + `"}`
}
