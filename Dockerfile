FROM golang:1.18-alpine

RUN apk add --no-cache git

WORKDIR /app

COPY payment-gw/go.mod .
COPY payment-gw/go.sum .

RUN go mod tidy

COPY payment-gw/. .

RUN go build -o ./out/app .

EXPOSE 8080

CMD ["./out/app"]