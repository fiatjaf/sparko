package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/rs/cors"
)

var (
	err         error
	scookie     = securecookie.New(securecookie.GenerateRandomKey(32), nil)
	accessKey   string
	manifestKey string
	login       string
	ee          chan event
	keys        Keys
)

const DEFAULTPORT = "9737"

//go:embed spark-wallet/client/dist/*
var static embed.FS
var sparkWallet, _ = fs.Sub(static, "spark-wallet/client/dist")

func main() {
	p := plugin.Plugin{
		Name:    "sparko",
		Version: "v2.9",
		Options: []plugin.Option{
			{"sparko-host", "string", "127.0.0.1", "http(s) server listen address"},
			{"sparko-port", "string", DEFAULTPORT, "http(s) server port"},
			{"sparko-login", "string", nil, "http basic auth login, \"username:password\" format"},
			{"sparko-keys", "string", nil, "semicolon-separated list of key-permissions pairs"},
			{"sparko-tls-path", "string", nil, "directory to read/store key.pem and cert.pem for TLS (relative to your lightning directory)"},
			{"sparko-letsencrypt-email", "string", nil, "email in which LetsEncrypt will notify you and other things"},
			{"sparko-allow-cors", "bool", false, "allow CORS"},
		},
		RPCMethods: []plugin.RPCMethod{
			// required by spark-wallet
			connectFund,
			closeGet,
			listpaysExt,
		},
		Subscriptions: []plugin.Subscription{
			subscribeSSE("channel_opened"),
			subscribeSSE("channel_open_failed"),
			subscribeSSE("channel_state_changed"),
			subscribeSSE("channel_opened"),
			subscribeSSE("connect"),
			subscribeSSE("disconnect"),
			{
				"invoice_payment",
				func(p *plugin.Plugin, params plugin.Params) {
					// serve both events

					// our generic one
					subscribeSSE("invoice_payment").Handler(p, params)

					// and the one spark wants
					label := params.Get("invoice_payment.label").String()
					inv, err := p.Client.Call("waitinvoice", label)
					if err != nil {
						p.Logf("Failed to get invoice on inv-paid notification: %s", err)
						return
					}
					ee <- event{typ: "inv-paid", data: inv.String()}
				},
			},
			subscribeSSE("invoice_creation"),
			subscribeSSE("warning"),
			subscribeSSE("forward_event"),
			subscribeSSE("sendpay_success"),
			subscribeSSE("sendpay_failure"),
			subscribeSSE("sendpay_success"),
			subscribeSSE("coin_movement"),
			subscribeSSE("openchannel_peer_sigs"),
		},
		OnInit: func(p *plugin.Plugin) {
			// compute access key
			login, _ = p.Args.String("sparko-login")
			if login != "" {
				accessKey = hmacStr(login, "access-key")
				manifestKey = hmacStr(accessKey, "manifest-key")
				p.Log("Login credentials read: " + login + " (full-access key: " + accessKey + ")")
			}

			// permissions
			if keypermissions, err := p.Args.String("sparko-keys"); err == nil {
				keys, err = readPermissionsConfig(keypermissions)
				if err != nil {
					p.Log("Error reading permissions config: " + err.Error())
					return
				}
				message, nkeys := keys.Summary()
				p.Logf("%d keys read: %s", nkeys, message)
				if nkeys == 0 {
					p.Log("DANGER: All methods are free for anyone to call without authorization.")
				}
			}

			// start eventsource thing
			es := startStreams(p)

			// declare routes
			router := mux.NewRouter()

			router.Use(authMiddleware(p))
			router.Use(gziphandler.GzipHandler)

			router.Path("/stream").Methods("GET").Handler(
				checkStreamPermission(es),
			)
			router.Path("/rpc").Methods("POST").Handler(http.HandlerFunc(handleRPC))

			if login != "" {
				// web ui
				router.Path("/").Methods("GET").HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						indexb, err := fs.ReadFile(sparkWallet, "index.html")
						if err != nil {
							p.Log(err.Error())
							w.WriteHeader(404)
							return
						}
						indexb = bytes.Replace(indexb, []byte("{{accessKey}}"), []byte(accessKey), -1)
						indexb = bytes.Replace(indexb, []byte("{{manifestKey}}"), []byte(manifestKey), -1)
						w.Header().Set("Content-Type", "text/html")
						w.Write(indexb)
						return
					})
				router.PathPrefix("/").Methods("GET").Handler(
					http.FileServer(http.FS(sparkWallet)),
				)
			}

			// start server
			if p.Args.Get("sparko-allow-cors").Bool() {
				listen(p, cors.AllowAll().Handler(router))
			} else {
				listen(p, router)
			}
		},
	}

	p.Run()
}

func subscribeSSE(kind string) plugin.Subscription {
	return plugin.Subscription{
		kind,
		func(p *plugin.Plugin, params plugin.Params) {
			j, _ := json.Marshal(params)
			ee <- event{typ: kind, data: string(j)}
		},
	}
}
