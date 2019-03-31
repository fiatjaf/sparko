package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var err error
var Version string
var ln *lightning.Client
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
var httpPublic = &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: ""}

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log = log.With().Timestamp().Logger()

	viper.SetConfigName("config")
	viper.AddConfigPath("$HOME/.spark-wallet")

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
	pflag.BoolP("version", "v", false, "output version number")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	// pretty.Log(viper.AllSettings())

	if viper.GetBool("version") {
		fmt.Println(Version)
		os.Exit(0)
	}

	// starting app
	ln = &lightning.Client{
		Path: path.Join(viper.GetString("ln-path"), "lightning-rpc"),
	}

	if !viper.GetBool("no-test-conn") {
		probeLightningd()
	}

	router := mux.NewRouter()

	router.Path("/rpc").Methods("POST").HandlerFunc(handleRPC)

	if !viper.GetBool("no-webui") {
		router.PathPrefix("/").Methods("GET").Handler(http.FileServer(httpPublic))
	}

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

func probeLightningd() {
	nodeinfo, err := ln.Call("getinfo")
	if err != nil {
		log.Warn().Err(err).Msg("can't talk to lightningd. retrying.")
		time.Sleep(time.Second * 5)
		probeLightningd()
		return
	}
	log.Info().
		Str("id", nodeinfo.Get("id").String()).
		Str("alias", nodeinfo.Get("alias").String()).
		Int64("channels", nodeinfo.Get("num_active_channels").Int()).
		Int64("blockheight", nodeinfo.Get("blockheight").Int()).
		Str("version", nodeinfo.Get("version").String()).
		Msg("lightning node connected")
}
