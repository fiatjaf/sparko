package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/lucsky/cuid"
	"github.com/mitchellh/go-homedir"
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
var accessKey string
var manifestKey string
var login string

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log = log.With().Timestamp().Logger()

	viper.SetConfigName("sparko")
	viper.AddConfigPath("$HOME/.config")

	viper.ReadInConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Fatal error reading config file.")
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	pflag.StringP("ln-path", "l", "~/.lightning", "path to c-lightning data directory")
	pflag.StringP("port", "p", "9737", "http(s) server port")
	pflag.StringP("host", "i", "localhost", "http(s) server listen address")
	pflag.StringP("login", "u", "generate random", "http basic auth login, \"username:password\" format")
	pflag.String("tls-path", "~/.spark-wallet/tls/", "directory to read key.pem and cert.pem for TLS")
	pflag.Bool("force-tls", false, "enable TLS even when binding on localhost")
	pflag.Bool("no-tls", false, "disable TLS for non-localhost hosts")
	pflag.Bool("no-webui", false, "run API server without serving client assets")
	pflag.Bool("no-test-conn", false, "skip testing access to c-lightning rpc")
	pflag.BoolP("print-key", "k", false, "print access key to console")
	pflag.BoolP("version", "v", false, "output version number")
	pflag.BoolP("help", "h", false, "output usage information")
	pflag.CommandLine.SortFlags = false
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	// --version
	if viper.GetBool("version") {
		fmt.Println(Version)
		os.Exit(0)
	}

	// --help
	if viper.GetBool("help") {
		fmt.Fprintf(os.Stderr, "\nA minimalistic wallet GUI for c-lightning repackaged\n\nUsage\n  $ sparko [options]\n\nOptions\n")
		pflag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n\nExample\n  $ sparko -l ~/.lightning\n\nAll options may also be specified as environment variables:\n  $ LN_PATH=/data/lightning PORT=8070 NO_TLS=1 sparko\nOr define a config file at ~/.config/sparko.toml or ~/.config/sparko.yaml")
		os.Exit(0)
	}

	// compute access key
	login = viper.GetString("login")
	if login == "generate random" {
		login = cuid.New() + ":" + cuid.New()
	}
	accessKey = hmacStr(login, "access-key")
	manifestKey = hmacStr(accessKey, "manifest-key")
	if viper.GetBool("print-key") {
		fmt.Println("Access key for remote API access: " + accessKey)
	}

	// start lightning client
	lnpath := viper.GetString("ln-path")
	lnpath, err = homedir.Expand(lnpath)
	if err != nil {
		log.Fatal().Err(err).Str("path", lnpath).Msg("cannot find home directory.")
	}
	ln = &lightning.Client{Path: path.Join(lnpath, "lightning-rpc")}
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
		addWebUI(router)
	}

	// start server
	host := viper.GetString("host")
	port := viper.GetString("port")
	srv := &http.Server{
		Handler: cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: false,
		}).Handler(router),
		Addr:         host + ":" + port,
		WriteTimeout: 25 * time.Second,
		ReadTimeout:  25 * time.Second,
	}
	if host == "localhost" && !viper.GetBool("force-tls") || viper.GetBool("no-tls") {
		log.Info().Str("addr", "http://"+srv.Addr+"/").Msg("HTTP server on")
		err = srv.ListenAndServe()
	} else {
		log.Info().Str("addr", "https://"+srv.Addr+"/").Msg("HTTPS server on")
		tlspath := viper.GetString("tls-path")
		tlspath, err = homedir.Expand(tlspath)
		if err != nil {
			log.Fatal().Err(err).Str("path", tlspath).Msg("cannot find home directory.")
		}
		err = srv.ListenAndServeTLS(path.Join(tlspath, "cert.pem"), path.Join(tlspath, "key.pem"))
	}
	if err != http.ErrServerClosed {
		log.Warn().Err(err).Msg("error starting server.")
	}
}
