package main

import (
	"encoding/json"
	"net/http"

	lightning "github.com/fiatjaf/lightningd-gjson-rpc"
)

func handleRPC(w http.ResponseWriter, r *http.Request) {
	if auth(r) != nil {
		w.WriteHeader(401)
		return
	}

	var req lightning.JSONRPCMessage
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	respbytes, err := ln.CallMessageRaw(lightning.DefaultTimeout, req)
	if err != nil {
		log.Error().Err(err).Msg("")
		w.WriteHeader(500)

		if cmderr, ok := err.(lightning.ErrorCommand); ok {
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

	w.Write(respbytes)
}

func auth(r *http.Request) error {
	return nil
}

type LightningError struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Message  string `json:"message"`
	Code     int    `json:"code"`
	FullType string `json:"fullType"`
}
