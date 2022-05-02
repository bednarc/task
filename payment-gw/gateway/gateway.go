package gateway

import (
	"context"
	"errors"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const PaymentsCol = "payments"

var (
	ErrCaptureToHigh           = errors.New("capture amount is higher than authorized")
	ErrRefundToHigh            = errors.New("refund amount is higher than authorized")
	ErrAlreadyRefunded         = errors.New("cannot perfom this operation because payment was already refunded")
	ErrAlreadyVoided           = errors.New("cannot perfom this operation because payment was already voided")
	ErrAmountIsZero            = errors.New("amount should be higher than 0.0")
	ErrBasedOnCreditCardNumber = errors.New("error based on credit card number")
	ErrAlreadyCaptured         = errors.New("cannot perfom this operation because payment was already captured")
	ErrPaymentIsCancelled      = errors.New("payment is cancelled")
	ErrNotCaptured             = errors.New("cannot refund non-captured transaction")
)

type MockFailure uint8

const (
	NoFailure            MockFailure = 0
	AuthorizationFailure             = 1
	CaptureFailure                   = 2
	RefundFailure                    = 3
)

type Payment struct {
	Id         string      `bson:"id"`
	Authorized int         `bson:"auhtorized"`
	Captured   int         `bson:"captured"`
	Refunded   int         `bson:"refunded"`
	Currency   string      `bson:"currency"`
	MerchantId string      `bson:"merchantid"`
	Failure    MockFailure `bson:"mockfailure"`
	Version    int         `bson:"version"`
	Voided     bool        `bson:"voided"`
}

type GatewayRepository interface {
	Authorize(ctx context.Context, amount int, currency, merchantId string, failure MockFailure) (string, error)
	GetMerchantIdByPaymentId(ctx context.Context, paymentId string) (string, error)
	Capture(ctx context.Context, paymentId string, amount int) (Payment, error)
	Refund(ctx context.Context, paymentId string, amount int) (Payment, error)
	Void(ctx context.Context, paymentId string) (Payment, error)
}

type MongoGatewayRepository struct {
	db *mongo.Database
}

func NewRepository(db *mongo.Database) MongoGatewayRepository {
	return MongoGatewayRepository{db: db}
}

func (g MongoGatewayRepository) Authorize(ctx context.Context, amount int, currency, merchantId string, failure MockFailure) (string, error) {
	lg := ctx.Value("logger").(*zerolog.Logger)

	if failure == AuthorizationFailure {
		return "", ErrBasedOnCreditCardNumber
	}
	if amount <= 0 {
		return "", ErrAmountIsZero
	}

	paymentId := xid.New().String()
	payment := Payment{Currency: currency, Authorized: amount, Id: paymentId, Failure: failure, MerchantId: merchantId}
	_, err := g.db.Collection(PaymentsCol).InsertOne(ctx, payment)
	if err != nil {
		lg.Error().Msg(err.Error())
		return "", err
	}

	return paymentId, nil
}

func (g MongoGatewayRepository) Capture(ctx context.Context, paymentId string, amount int) (Payment, error) {
	lg := ctx.Value("logger").(*zerolog.Logger)
	result := Payment{}
	if err := g.db.Collection(PaymentsCol).FindOne(ctx, bson.M{"id": paymentId}).Decode(&result); err != nil {
		lg.Error().Msg(err.Error())
		return Payment{}, err
	}

	if result.Voided {
		return result, ErrPaymentIsCancelled
	}

	if result.Failure == CaptureFailure {
		return result, ErrBasedOnCreditCardNumber
	}

	if amount <= 0 {
		return result, ErrAmountIsZero
	}

	if result.Authorized < result.Captured+amount {
		return result, ErrCaptureToHigh
	}

	if result.Refunded > 0 {
		return result, ErrAlreadyRefunded
	}
	result.Captured += amount

	filter := bson.M{"id": paymentId, "version": result.Version}
	result.Version++

	updateResult, err := g.db.Collection(PaymentsCol).ReplaceOne(ctx, filter, result)
	if err != nil {
		lg.Error().Msg(err.Error())
		return Payment{}, err
	}

	if updateResult.ModifiedCount == 0 {
		if err := g.db.Collection(PaymentsCol).FindOne(ctx, bson.M{"id": paymentId}).Decode(&result); err != nil {
			lg.Error().Msg(err.Error())
			return Payment{}, err
		}
		err := errors.New("optimistic locking: could not update document")
		lg.Debug().Msg(err.Error())
		return result, err
	}

	return result, nil
}

func (g MongoGatewayRepository) Refund(ctx context.Context, paymentId string, amount int) (Payment, error) {
	lg := ctx.Value("logger").(*zerolog.Logger)
	result := Payment{}
	if err := g.db.Collection(PaymentsCol).FindOne(ctx, bson.M{"id": paymentId}).Decode(&result); err != nil {
		lg.Error().Msg(err.Error())
		return Payment{}, err
	}

	if result.Voided {
		return result, ErrPaymentIsCancelled
	}

	if result.Captured == 0 {
		return result, ErrNotCaptured
	}

	if result.Failure == RefundFailure {
		return result, ErrBasedOnCreditCardNumber
	}

	if amount <= 0 {
		return result, ErrAmountIsZero
	}

	if result.Captured < result.Refunded+amount {
		return result, ErrRefundToHigh
	}

	result.Refunded += amount

	filter := bson.M{"id": paymentId, "version": result.Version}
	result.Version++

	updateResult, err := g.db.Collection(PaymentsCol).ReplaceOne(ctx, filter, result)
	if err != nil {
		lg.Error().Msg(err.Error())
		return result, err
	}

	if updateResult.ModifiedCount == 0 {
		if err := g.db.Collection(PaymentsCol).FindOne(ctx, bson.M{"id": paymentId}).Decode(&result); err != nil {
			lg.Error().Msg(err.Error())
			return Payment{}, err
		}
		err := errors.New("optimistic locking: could not update document")
		lg.Debug().Msg(err.Error())
		return result, err
	}
	return result, nil
}

func (g MongoGatewayRepository) Void(ctx context.Context, paymentId string) (Payment, error) {
	lg := ctx.Value("logger").(*zerolog.Logger)
	result := Payment{}
	if err := g.db.Collection(PaymentsCol).FindOne(ctx, bson.M{"id": paymentId}).Decode(&result); err != nil {
		lg.Error().Msg(err.Error())
		return Payment{}, err
	}
	if result.Voided {
		return result, ErrAlreadyVoided
	}

	if result.Refunded != 0 {
		return result, ErrAlreadyRefunded
	}
	if result.Captured != 0 {
		return result, ErrAlreadyCaptured
	}
	filter := bson.M{"id": paymentId, "version": result.Version}
	result.Version++
	result.Voided = true
	updateResult, err := g.db.Collection(PaymentsCol).ReplaceOne(ctx, filter, result)
	if err != nil {
		lg.Error().Msg(err.Error())
		return result, err
	}

	if updateResult.ModifiedCount == 0 {
		if err := g.db.Collection(PaymentsCol).FindOne(ctx, bson.M{"id": paymentId}).Decode(&result); err != nil {
			lg.Error().Msg(err.Error())
			return Payment{}, err
		}
		err := errors.New("optimistic locking: could not update document")
		lg.Debug().Msg(err.Error())
		return result, err
	}
	return result, nil
}

func (g MongoGatewayRepository) GetMerchantIdByPaymentId(ctx context.Context, paymentId string) (string, error) {
	lg := ctx.Value("logger").(*zerolog.Logger)
	result := bson.M{}
	if err := g.db.Collection(PaymentsCol).FindOne(ctx, bson.M{"id": paymentId}).Decode(&result); err != nil {
		lg.Error().Msg(err.Error())
		return "", err
	}

	return result["merchantid"].(string), nil
}
