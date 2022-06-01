## Description of the task
Description can be found in APIChallenge.pdf

## Implemented extra mile bonus points and more:


● Client (merchant) authentication

● Application logging with unique transaction id

● Containerization with a docker compose

● Client (merchant) authorization - only owner (merchant) of the transaction can perform actions on the transaction.

● Unit tests

● Handling multiple transactions at the same time using optimistic locking


## How to run application using docker-compose?
Run in the root directory:
```bash
docker-compose  up -d --build
```
The application runs on the port 8080 and database on the port 27020. This can be changed in the .env file in the root directory.

## How to run unit tests?
Unit tests needs the running mongo database instance. 
Run the database using 
```bash 
docker-compose  up -d --build mongodb_container
```
in the root directory. 

Then run
```bash 
MONGO_ROOT_USERNAME=root MONGO_ROOT_PASSWORD=rootpassword MONGO_PORT_NUMBER=27020 go test
```
in the root/payment-gw directory.


## A few examples of requests and responses

### Register
```bash
curl --location --request POST 'localhost:8080/merchant/register'
```
```bash
{
    "merchant_id": "c9nrc7r5g7ia69hskp30",
    "secret_key": "BpLnfgDsc2WD8F2qNfHK5a84j"
}
```

### Authorization
```bash
curl --location --request POST 'localhost:8080/merchant/c9nrc7r5g7ia69hskp30/authorize' \
--header 'Authorization: BpLnfgDsc2WD8F2qNfHK5a84j' \
--header 'Content-Type: application/json' \
--data-raw '{
    "name_surname": "Krystian Bednarczuk",
    "card_number": "1111222233334444",
    "expiry_month": "12",
    "expiry_year": "22",
    "CCV": "123",
    "amount": "99.99",
    "currency": "PLN"
}'
```
```bash
{
    "payment_id": "c9nrinj5g7ia69hskp40",
    "available_to_capture": "99.99",
    "available_to_refund": "0.00",
    "currency": "PLN"
}
```

### Invalid Capture
```bash
curl --location --request POST 'localhost:8080/merchant/c9nrc7r5g7ia69hskp30/capture/c9nrinj5g7ia69hskp40' \
--header 'Authorization: BpLnfgDsc2WD8F2qNfHK5a84j' \
--header 'Content-Type: text/plain' \
--data-raw '{
    "amount": "999.99"
}'
```
```bash
{
    "available_to_capture": "99.99",
    "available_to_refund": "0.00",
    "currency": "PLN",
    "error": "capture amount is higher than authorized"
}
```

### Capture
```bash
curl --location --request POST 'localhost:8080/merchant/c9nrc7r5g7ia69hskp30/capture/c9nrinj5g7ia69hskp40' \
--header 'Authorization: BpLnfgDsc2WD8F2qNfHK5a84j' \
--header 'Content-Type: text/plain' \
--data-raw '{
    "amount": "9.99"
}'
```
```bash
{
    "available_to_capture": "90.00",
    "available_to_refund": "9.99",
    "currency": "PLN"
}
```
### Refund
```bash
curl --location --request POST 'localhost:8080/merchant/c9nrc7r5g7ia69hskp30/refund/c9nrinj5g7ia69hskp40' \
--header 'Authorization: BpLnfgDsc2WD8F2qNfHK5a84j' \
--header 'Content-Type: text/plain' \
--data-raw '{
    "amount": "8.99"
}'
```
```bash
{
    "available_to_capture": "0.00",
    "available_to_refund": "1.00",
    "currency": "PLN"
}
```

### Void
```bash
curl --location --request POST 'localhost:8080/merchant/c9nrc7r5g7ia69hskp30/void/c9nrlgb5g7ia69hskp6g' \
--header 'Authorization: BpLnfgDsc2WD8F2qNfHK5a84j' \
--data-raw ''
```
```bash
{
    "available_to_capture": "0.00",
    "available_to_refund": "0.00",
    "currency": "PLN"
}
```
