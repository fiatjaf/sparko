package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/apsdehal/go-logger"
	"github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/lucsky/cuid"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var err error
var Version string
var ln *lightning.Client
var log *logger.Logger
var scookie = securecookie.New([]byte("ilsvfoisg7rils3g4fo8segzr"), []byte("OHAOHDP4BLAKBDPAS3BÇSF"))
var accessKey string
var manifestKey string
var login string
var keys Keys

func main() {
	log, err = logger.New("main", 1, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error instantiating logger on stderr")
		os.Exit(-1)
	}
	log.SetFormat("[%{level}] %{message}")

	viper.SetConfigName("sparko")
	viper.AddConfigPath("$HOME/.config")

	viper.ReadInConfig()
	if err != nil {
		log.Warning("Fatal error reading config file.")
		os.Exit(-2)
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
	pflag.BoolP("print-key", "k", false, "print access keys to console")
	pflag.BoolP("verbose", "V", false, "display debugging information")
	pflag.BoolP("help", "h", false, "output usage information")
	pflag.BoolP("version", "v", false, "output version number")
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

	// log level
	if viper.GetBool("verbose") {
		log.SetLogLevel(logger.DebugLevel)
	}

	// permissions
	keys, err = readPermissionsConfig()
	if err != nil {
		log.WarningF("Error reading permissions config: %s.", err)
		os.Exit(-3)
	}

	// compute access key
	login = viper.GetString("login")
	if login == "generate random" {
		login = cuid.New() + ":" + cuid.New()
	}
	accessKey = hmacStr(login, "access-key")
	manifestKey = hmacStr(accessKey, "manifest-key")
	if viper.GetBool("print-key") {
		log.Info("Access keys:")
		fmt.Fprintf(os.Stderr, "  %s (default/login): full-access", accessKey)
		for key, permissions := range keys {
			fmt.Fprintf(os.Stderr, "\n  %s: %s", key, permissions)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// start lightning client
	lnpath := viper.GetString("ln-path")
	lnpath, err = homedir.Expand(lnpath)
	if err != nil {
		log.ErrorF("Cannot find home directory (on %s): %s.", lnpath, err)
		os.Exit(-4)
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
		Handler:      router,
		Addr:         host + ":" + port,
		WriteTimeout: 25 * time.Second,
		ReadTimeout:  25 * time.Second,
	}
	if host == "localhost" && !viper.GetBool("force-tls") || viper.GetBool("no-tls") {
		log.Info("HTTP server on http://" + srv.Addr + "/")
		err = srv.ListenAndServe()
	} else {
		log.Info("HTTPS server on https://" + srv.Addr + "/")
		tlspath := viper.GetString("tls-path")
		tlspath, err = homedir.Expand(tlspath)
		if err != nil {
			log.ErrorF("Cannot find home directory (on %s): %s.", tlspath, err)
			os.Exit(-5)
		}
		err = srv.ListenAndServeTLS(path.Join(tlspath, "cert.pem"), path.Join(tlspath, "key.pem"))
	}
	if err != http.ErrServerClosed {
		log.ErrorF("Error starting HTTP server: %s.", err)
	}
}
