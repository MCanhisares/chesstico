package main

import (
	"context"	
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"	
	"github.com/go-oauth2/oauth2/v4/errors"
	"github.com/go-oauth2/oauth2/v4/generates"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/golang-jwt/jwt"	
	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	pg "github.com/vgarvardt/go-oauth2-pg/v4"
	"github.com/vgarvardt/go-pg-adapter/pgx4adapter"
)



func startupEnv() (string, string) {
	godotenv.Load()

	portString := os.Getenv("PORT")
	if portString == "" {
		log.Fatal("PORT not set in environment")
	}
	fmt.Println("Port: ", portString)

	dbUrl := os.Getenv("DB_URL")
	if portString == "" {
		log.Fatal("DB_URL not set in environment")
	}

	return portString, dbUrl
}

func mountRouter(oauthSrv *server.Server) *chi.Mux {
	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	v1Router := chi.NewRouter()
	v1Router.Get("/healthz", handlerReadiness)
	v1Router.Get("/err", handlerErr)

	oAuthRouter := chi.NewRouter()
	oAuthRouter.Post("/auth", func(w http.ResponseWriter, r *http.Request) {
		err := oauthSrv.HandleAuthorizeRequest(w, r)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})
	oAuthRouter.Post("/token", func(w http.ResponseWriter, r *http.Request) {
		oauthSrv.HandleTokenRequest(w, r)
	})

	router.Mount("/v1", v1Router)
	router.Mount("/oauth", oAuthRouter)
	return router
}

func startupOauth(dbUrl string) *server.Server {

	manager := manage.NewDefaultManager()
	pgxConn, _ := pgx.Connect(context.TODO(), dbUrl)
	// use PostgreSQL token store with pgx.Connection adapter
	adapter := pgx4adapter.NewConn(pgxConn)
	tokenStore, _ := pg.NewTokenStore(adapter, pg.WithTokenStoreGCInterval(time.Minute))
	defer tokenStore.Close()

	clientStore, _ := pg.NewClientStore(adapter)	
	manager.MapAccessGenerate(generates.NewJWTAccessGenerate("", []byte("00000000"), jwt.SigningMethodHS512))
	manager.MapTokenStorage(tokenStore)
	manager.MapClientStorage(clientStore)
	manager.SetAuthorizeCodeTokenCfg(manage.DefaultAuthorizeCodeTokenCfg)
	manager.SetRefreshTokenCfg(manage.DefaultRefreshTokenCfg)

	srv := server.NewDefaultServer(manager)
	srv.SetAllowGetAccessRequest(true)
	srv.SetClientInfoHandler(server.ClientFormHandler)

	srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
		log.Println("Internal Error:", err.Error())
		return
	})

	srv.SetResponseErrorHandler(func(re *errors.Response) {
		log.Println("Response Error:", re.Error.Error())
	})

	return srv
}

func main() {
	portString, dbUrl := startupEnv()	
	oauthSrv := startupOauth(dbUrl)
	router := mountRouter(oauthSrv)

	srv := &http.Server{
		Handler: router,
		Addr:    ":" + portString,
	}
	log.Printf("Server starting on port: %v", portString)

	srvErr := srv.ListenAndServe()
	if srvErr != nil {
		log.Fatal(srvErr)
	}
}