package main

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/tidwall/gjson"
	"gopkg.in/antage/eventsource.v1"
)

type event struct {
	typ  string
	data string
}

func startStreams() eventsource.EventSource {
	id := 1
	lastInvoiceIndex := 0

	res, err := ln.Call("listinvoices")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list invoices.")
	}
	indexes := res.Get("invoices.#.pay_index").Array()
	for _, indexr := range indexes {
		index := int(indexr.Int())
		if index > lastInvoiceIndex {
			lastInvoiceIndex = index
		}
	}
	log.Info().Int("pay_index", lastInvoiceIndex).Msg("got initial invoice pay_index.")

	es := eventsource.New(
		eventsource.DefaultSettings(),
		func(req *http.Request) [][]byte {
			return [][]byte{
				[]byte("X-Accel-Buffering: no"),
				[]byte("Cache-Control: no-cache"),
				[]byte("Content-Type: text/event-stream"),
				[]byte("Connection: keep-alive"),
			}
		},
	)

	ee := make(chan event)
	go pollRate(ee)
	go pollInvoice(lastInvoiceIndex, ee)

	go func() {
		for {
			select {
			case e := <-ee:
				es.SendEventMessage(e.data, e.typ, strconv.Itoa(id))
			}
			id++
		}
	}()

	return es
}

func pollRate(ee chan<- event) {
	defer pollRate(ee)

	resp, err := http.Get("https://www.bitstamp.net/api/v2/ticker/btcusd")
	if err != nil || resp.StatusCode >= 300 {
		log.Error().Err(err).Int("code", resp.StatusCode).Msg("error fetching BTC price.")
		return
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("error decoding BTC price.")
		return
	}

	lastRate := gjson.GetBytes(b, "last").String()
	ee <- event{typ: "btcusd", data: `"` + lastRate + `"`}

	time.Sleep(time.Minute)
}

func pollInvoice(lastInvoiceIndex int, ee chan<- event) {
	defer pollInvoice(lastInvoiceIndex, ee)

	inv, err := ln.CallWithCustomTimeout(time.Hour,
		"waitanyinvoice", strconv.Itoa(lastInvoiceIndex))
	if err != nil {
		if _, ok := err.(lightning.ErrorTimeout); ok {
			time.Sleep(time.Minute)
		}

		return
	}

	lastInvoiceIndex = int(inv.Get("pay_index").Int())
	ee <- event{typ: "inv-paid", data: inv.String()}
}
