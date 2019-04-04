package main

import (
	"encoding/json"
	"net/http"
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

	respbytes, err := ln.CallMessageRaw(time.Second*30, req)
	if err != nil {
		log.Error().Err(err).Str("method", req.Method).
			Msg("error calling rpc")
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
