package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/fiatjaf/lightningd-gjson-rpc"
)

func handleRPC(w http.ResponseWriter, r *http.Request) {
	var req lightning.JSONRPCMessage
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	// check permissions
	if permissions, ok := r.Context().Value("permissions").(PermissionSet); ok {
		// if there are any allowed_methods, ignore disallowed_methods
		// setting and disallow anything that is not explicitly allowed.
		// otherwise only disallow what is explicitly disallowed.
		if len(permissions.AllowedMethods) > 0 {
			if _, allowed := permissions.AllowedMethods[req.Method]; !allowed {
				w.WriteHeader(401)
				return
			}
		} else {
			if _, disallowed := permissions.DisallowedMethods[req.Method]; disallowed {
				w.WriteHeader(401)
				return
			}
		}

		// special limits
		if req.Method == "pay" && permissions.PayMaxSatoshis != nil {
			var bolt11 string
			var msatoshi int
			var ok bool
			switch params := req.Params.(type) {
			case []interface{}:
				if len(params) > 0 {
					if bolt11, ok = params[0].(string); !ok {
						break
					}
					if len(params) > 1 {
						switch msats := params[1].(type) {
						case float64:
							msatoshi = int(msats)
						case string:
							msatoshi, _ = strconv.Atoi(msats)
						}
					}
				}
			case map[string]interface{}:
				if ibolt11, ok := params["bolt11"]; ok {
					bolt11, _ = ibolt11.(string)
				}
				if imsatoshi, ok := params["msatoshi"]; ok {
					fmsatoshi, _ := imsatoshi.(float64)
					msatoshi = int(fmsatoshi)
				}
			}

			var sats int
			inv, _ := ln.Call("decodepay", bolt11)
			pmsatoshi := inv.Get("msatoshi")
			if pmsatoshi.Exists() {
				sats = int(pmsatoshi.Int()) / 1000
			} else {
				sats = msatoshi / 1000
			}

			if sats > *permissions.PayMaxSatoshis {
				w.WriteHeader(401)
				return
			}
		}
	}

	// actually do the call
	respbytes, err := ln.CallMessageRaw(time.Second*30, req)
	if err != nil {
		log.NoticeF("Error calling RPC '%s': %s", req.Method, err)
		w.WriteHeader(500)

		if cmderr, ok := err.(lightning.ErrorCommand); ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(LightningError{
				Type:     "lightning",
				Name:     "LightningError",
				Message:  cmderr.Msg,
				Code:     cmderr.Code,
				FullType: "lightning",
			})
		}

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respbytes)
}

type LightningError struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Message  string `json:"message"`
	Code     int    `json:"code"`
	FullType string `json:"fullType"`
}
