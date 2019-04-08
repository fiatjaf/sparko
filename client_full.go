// +build full

package main

import (
	"bytes"
	"net/http"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
)

var httpPublic = &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: ""}

func addWebUI(router *mux.Router) {
	router.Path("/").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			indexb, err := Asset("index.html")
			if err != nil {
				w.WriteHeader(404)
				return
			}
			indexb = bytes.Replace(indexb, []byte("{{accessKey}}"), []byte(accessKey), -1)
			indexb = bytes.Replace(indexb, []byte("{{manifestKey}}"), []byte(manifestKey), -1)
			w.Header().Set("Content-Type", "text/html")
			w.Write(indexb)
			return
		})
	router.PathPrefix("/").Methods("GET").Handler(http.FileServer(httpPublic))
}
