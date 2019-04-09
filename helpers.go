package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"time"
)

var nonLetters = regexp.MustCompile(`\W+`)

func hmacStr(key, data string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	b64 := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return nonLetters.ReplaceAllString(b64, "")
}

func probeLightningd() {
	nodeinfo, err := ln.Call("getinfo")
	if err != nil {
		log.WarningF("Can't talk to lightningd (%s). Retrying.", err)
		time.Sleep(time.Second * 10)
		probeLightningd()
		return
	}
	log.InfoF("Connected to lightningd node id %s, alias %s. Running version %s with %d channels on blockheight %d.",
		nodeinfo.Get("id").String(),
		nodeinfo.Get("alias").String(),
		nodeinfo.Get("version").String(),
		nodeinfo.Get("num_active_channels").Int(),
		nodeinfo.Get("blockheight").Int(),
	)
}
