package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/tidwall/gjson"
)

type sseclient struct {
	closed  bool
	invoice chan gjson.Result
	rate    chan string
}

var streamingStarted bool
var lastRate string
var lastInvoiceIndex string

func startStreams() {
	if streamingStarted {
		return
	}

	go pollRate()

	res, err := ln.Call("listinvoices")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list invoices.")
	}
	invoices := res.Get("invoices").Array()
	lastInvoiceIndex = invoices[len(invoices)-1].Get("pay_index").String()
	go pollInvoice()
}

func pollRate() {
	defer pollRate()

	if len(broker) > 0 {
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

		lastRate = gjson.GetBytes(b, "last").String()
		for _, client := range broker {
			if !client.closed {
				client.rate <- lastRate
			}
		}
	}

	time.Sleep(time.Minute)
}

func pollInvoice() {
	defer pollInvoice()

	inv, err := ln.CallWithCustomTimeout(time.Minute*5, "waitanyinvoice", lastInvoiceIndex)
	if err != nil {
		log.Error().Err(err).Str("index", lastInvoiceIndex).Msg("error waiting invoice.")
		return
	}

	lastInvoiceIndex = inv.Get("pay_index").String()
	for _, client := range broker {
		if !client.closed {
			client.invoice <- inv
		}
	}
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(500)
		return
	}

	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	write := func(data string) {
		w.Write([]byte(data + "\n\n"))
		flusher.Flush()
	}

	// init client
	client := sseclient{
		closed:  false,
		invoice: make(chan gjson.Result),
		rate:    make(chan string),
	}
	broker = append(broker, client)
	if lastRate != "" {
		client.rate <- lastRate
	}

	write("retry: 3000")
	closed := w.(http.CloseNotifier).CloseNotify()

	go func() {
		select {
		case <-closed:
			return
		case <-time.After(time.Second * 250):
			write(": keepalive")
			return
		}
	}()

	go func() {
		for {
			select {
			case <-closed:
				return
			case inv := <-client.invoice:
				write("event:inv-paid\ndata:" + inv.String())
			}
		}
	}()

	go func() {
		for {
			select {
			case <-closed:
				return
			case rate := <-client.rate:
				write(fmt.Sprintf(`event:btcusd\ndata:"%s"`, rate))
			}
		}
	}()

	go func() {
		x := <-closed
		log.Print("closed", x)
		client.closed = true
	}()
}
