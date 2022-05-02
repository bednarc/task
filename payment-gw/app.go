package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"payment-gw/gateway"
	"payment-gw/merchant"

	"time"

	"github.com/gorilla/mux"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	ErrForbidden    = errors.New("operation is forbidden")
	AdminKeyInvalid = errors.New("admin key is invalid")
)

type App struct {
	router   *mux.Router
	db       *mongo.Client
	lg       *zerolog.Logger
	gateway  gateway.GatewayRepository
	merchant merchant.MerchantRepository
	dbname   string
}

type Config struct {
	dbName     string
	dbUsername string
	dbPassword string
	dbPort     string
}

func (a *App) Initialize(c Config) {
	a.dbname = c.dbName
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	credential := options.Credential{
		Username: c.dbUsername,
		Password: c.dbPassword,
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:"+c.dbPort).SetAuth(credential))
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	a.db = client

	lg := log.With().Caller().Logger()
	a.lg = &lg

	a.router = mux.NewRouter()
	a.initializeRoutes()
	a.gateway = gateway.NewRepository(a.db.Database(a.dbname))
	a.merchant = merchant.NewRepository(a.db.Database(a.dbname))
}

func (a *App) Run(addr string) {
	log.Fatal().Err(http.ListenAndServe(addr, a.router))
	defer func() {
		if err := a.db.Disconnect(context.Background()); err != nil {
			panic(err)
		}
	}()
}

func (a *App) initializeRoutes() {
	xid := `.{20}`

	addLoggerRouter := a.router.NewRoute().Subrouter()
	addLoggerRouter.HandleFunc("/merchant/register", a.register).Methods(http.MethodPost)
	addLoggerRouter.Use(a.addLogger)

	needAuthenticationRouter := a.router.NewRoute().Subrouter()
	needAuthenticationRouter.HandleFunc("/merchant/{merchant_id:"+xid+"}/authorize", a.authorize).Methods(http.MethodPost)
	needAuthenticationRouter.Use(a.addLogger)
	needAuthenticationRouter.Use(a.needAuthentication)

	needAutorizationRouter := a.router.NewRoute().Subrouter()
	needAutorizationRouter.HandleFunc("/merchant/{merchant_id:"+xid+"}/capture/{payment_id:"+xid+"}", a.capture).Methods(http.MethodPost)
	needAutorizationRouter.HandleFunc("/merchant/{merchant_id:"+xid+"}/refund/{payment_id:"+xid+"}", a.refund).Methods(http.MethodPost)
	needAutorizationRouter.HandleFunc("/merchant/{merchant_id:"+xid+"}/void/{payment_id:"+xid+"}", a.void).Methods(http.MethodPost)
	needAutorizationRouter.Use(a.addLogger)
	needAutorizationRouter.Use(a.needAuthentication)
	needAutorizationRouter.Use(a.needAutorization)
}

func (a *App) collection(name string) *mongo.Collection {
	return a.db.Database(a.dbname).Collection(name)
}

func (a *App) needAuthentication(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lg := r.Context().Value("logger").(*zerolog.Logger)
		secretKey := r.Header.Get("Authorization")
		merchantId := mux.Vars(r)["merchant_id"]

		err := a.merchant.IsAuthenticated(r.Context(), merchantId, secretKey)
		if errors.Is(merchant.ErrMerchantNotFound, err) {
			lg.Debug().Msg(err.Error())
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		} else if errors.Is(merchant.ErrWrongSecretKey, err) {
			lg.Debug().Msg(err.Error())
			respondWithError(w, http.StatusForbidden, err.Error())
			return
		} else if err != nil {
			lg.Error().Msg(err.Error())
			respondWithError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *App) needAutorization(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lg := r.Context().Value("logger").(*zerolog.Logger)
		merchantId := mux.Vars(r)["merchant_id"]
		paymentId := mux.Vars(r)["payment_id"]

		merchantIdFromPayment, err := a.gateway.GetMerchantIdByPaymentId(r.Context(), paymentId)
		if err != nil {
			lg.Error().Msg(err.Error())
			respondWithError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}

		if merchantId != merchantIdFromPayment {
			lg.Debug().Msg(err.Error())
			respondWithError(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *App) addLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sublog := a.lg.With().Str("transaction_id", xid.New().String()).Logger()
		merchantId := mux.Vars(r)["merchant_id"]
		paymentId := mux.Vars(r)["payment_id"]

		if merchantId != "" {
			sublog = sublog.With().Str("merchant_id", merchantId).Logger()
		}

		if paymentId != "" {
			sublog = sublog.With().Str("payment_id", paymentId).Logger()
		}

		sublog.Debug().Str("method", r.Method).Str("url", r.URL.Path).Msg("")
		ctx := context.WithValue(r.Context(), "logger", &sublog)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
