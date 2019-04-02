package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/lucsky/cuid"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var err error
var Version string
var ln *lightning.Client
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
var scookie = securecookie.New([]byte("ilsvfoisg7rils3g4fo8segzr"), []byte("OHAOHDP4BLAKBDPAS3BÇSF"))
var httpPublic = &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: ""}
var accessKey string
var login string

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log = log.With().Timestamp().Logger()

	viper.SetConfigName("spark")
	viper.AddConfigPath("$HOME/.config")

	viper.ReadInConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Fatal error reading config file.")
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	pflag.StringP("ln-path", "l", "/home/fiatjaf/.lightning", "path to c-lightning data directory")
	pflag.StringP("port", "p", "9737", "http(s) server port")
	pflag.StringP("host", "i", "localhost", "http(s) server listen address")
	pflag.StringP("login", "u", "generate random", "http basic auth login, \"username:password\" format")
	pflag.BoolP("print-key", "k", false, "print access key to console")
	pflag.Bool("no-webui", false, "run API server without serving client assets")
	pflag.Bool("no-test-conn", false, "skip testing access to c-lightning rpc")
	pflag.BoolP("version", "v", false, "output version number")
	pflag.BoolP("help", "h", false, "output usage information")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	// --version
	if viper.GetBool("version") {
		fmt.Println(Version)
		os.Exit(0)
	}

	// --help
	if viper.GetBool("help") {
		fmt.Fprintf(os.Stderr, "\nA minimalistic wallet GUI for c-lightning repackaged\n\nUsage\n  $ spark [options]\n\nOptions\n")
		pflag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n\nExample\n  $ spark -l ~/.lightning\n\nAll options may also be specified as environment variables:\n  $ LN_PATH=/data/lightning PORT=8070 NO_TLS=1 spark\nOr define a config file at ~/.config/spark.toml or ~/.config/spark.yaml")
		os.Exit(0)
	}

	// compute access key
	login = viper.GetString("login")
	if login == "generate random" {
		login = cuid.New() + ":" + cuid.New()
	}
	accessKey = hmacStr(login, "access-key")
	if viper.GetBool("print-key") {
		fmt.Println("Access key for remote API access: " + accessKey)
	}

	// start lightning client
	ln = &lightning.Client{
		Path: path.Join(viper.GetString("ln-path"), "lightning-rpc"),
	}
	if !viper.GetBool("no-test-conn") {
		probeLightningd()
	}

	// start eventsource thing
	es := startStreams()

	// declare routes
	router := mux.NewRouter()
	router.Use(authMiddleware)
	router.Path("/rpc").Methods("POST").HandlerFunc(handleRPC)
	router.Path("/stream").Methods("GET").Handler(es)
	if !viper.GetBool("no-webui") {
		router.Path("/").Methods("GET").HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				indexb, err := Asset("index.html")
				if err != nil {
					w.WriteHeader(404)
					return
				}
				indexb = bytes.Replace(indexb, []byte("{{accessKey}}"), []byte(accessKey), 1)
				w.Header().Set("Content-Type", "text/html")
				w.Write(indexb)
				return
			})
		router.PathPrefix("/").Methods("GET").Handler(http.FileServer(httpPublic))
	}

	// start server
	log.Info().Str("port", viper.GetString("port")).Msg("listening.")
	srv := &http.Server{
		Handler: cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: false,
		}).Handler(router),
		Addr:         viper.GetString("host") + ":" + viper.GetString("port"),
		WriteTimeout: 25 * time.Second,
		ReadTimeout:  25 * time.Second,
	}
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Warn().Err(err).Msg("listenAndServe")
	}
}
