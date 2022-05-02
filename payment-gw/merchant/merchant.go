package merchant

import (
	"context"
	"errors"
	"math/rand"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

const MerchantCol = "merchants"

var (
	ErrMerchantNotFound = errors.New("merchant with the given id not found")
	ErrWrongSecretKey   = errors.New("wrong secret key")
)

type merchant struct {
	HashedKey string `bson:"hashedkey"`
	Id        string `bson:"id"`
}

type MerchantRepository interface {
	Register(ctx context.Context) (string, string, error)
	IsAuthenticated(ctx context.Context, merchantId, secretKey string) error
}

type MongoMerchanyRepository struct {
	db *mongo.Database
}

func NewRepository(db *mongo.Database) MongoMerchanyRepository {
	return MongoMerchanyRepository{db: db}
}

func generateRandomKey(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func (g MongoMerchanyRepository) Register(ctx context.Context) (string, string, error) {
	lg := ctx.Value("logger").(*zerolog.Logger)
	merchantId := xid.New().String()
	secretKey := generateRandomKey(25)
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(secretKey), bcrypt.DefaultCost)
	if err != nil {
		lg.Error().Msg(err.Error())
		return "", "", err
	}
	merchant := merchant{HashedKey: string(hashedKey), Id: merchantId}
	_, err = g.db.Collection(MerchantCol).InsertOne(ctx, merchant)
	if err != nil {
		lg.Error().Msg(err.Error())
		return "", "", err
	}

	return merchantId, secretKey, nil
}

func (g MongoMerchanyRepository) IsAuthenticated(ctx context.Context, merchantId string, secretKey string) error {
	lg := ctx.Value("logger").(*zerolog.Logger)

	var result bson.M
	if err := g.db.Collection(MerchantCol).FindOne(ctx, bson.M{"id": merchantId}).Decode(&result); err == mongo.ErrNoDocuments {
		return ErrMerchantNotFound
	} else if err != nil {
		lg.Error().Msg(err.Error())
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(result["hashedkey"].(string)), []byte(secretKey)); err != nil {
		return ErrWrongSecretKey
	}

	return nil
}
