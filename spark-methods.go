// Methods required by spark-wallet client.
// https://github.com/shesek/spark-wallet/blob/master/src/cmd.js

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
	"github.com/tidwall/gjson"
)

var connectFund = plugin.RPCMethod{
	"connectfund",
	"peeruri satoshi feerate",
	"",
	"",
	func(p *plugin.Plugin, params plugin.Params) (resp interface{}, errCode int, err error) {
		peeruri := params.Get("peeruri").String()
		peerid := strings.Split(peeruri, "@")[0]
		p.Client.Call("connect", peeruri)

		satoshi := params.Get("satoshi").String()
		feerate := params.Get("feerate").String()
		res, err := p.Client.Call(peerid, satoshi, feerate)
		if err != nil {
			return nil, 37, errors.New("cannot open channel")
		}

		return getChannel(p, peerid, res.Get("channel_id").String())
	},
}

var closeGet = plugin.RPCMethod{
	"closeget",
	"peeruri chanid force timeout",
	"",
	"",
	func(p *plugin.Plugin, params plugin.Params) (resp interface{}, errCode int, err error) {
		peerid := params.Get("peerid").String()
		chanid := params.Get("chanid").String()
		force := params.Get("force").String()
		timeout := params.Get("timeout").String()

		res, err := p.Client.Call("close", chanid, force, timeout)
		if err != nil {
			return nil, 37, errors.New("cannot close channel")
		}

		retval, errCode, err := getChannel(p, peerid, res.Get("channel_id").String())
		if err != nil {
			return
		}

		retval["closing"] = res.Value()
		return retval, 0, nil
	},
}

var listpaysExt = plugin.RPCMethod{
	"listpaysext",
	"",
	"",
	"",
	func(p *plugin.Plugin, params plugin.Params) (resp interface{}, errCode int, err error) {
		res, err := p.Client.CallWithCustomTimeout(time.Second*30, "listpays")
		if err != nil {
			return nil, 37, errors.New("cannot listpays -- enable the pay plugin")
		}

		pays := res.Get("pays").Array()
		// more than 30, we filter
		if len(pays) > 30 {
			pays = pays[len(pays)-30:]
		}

		retval := make([]interface{}, len(pays))

		if len(pays) == 0 {
			return map[string]interface{}{"pays": retval}, 0, nil
		}

		// these are not currently available, but be prepared for when they are
		if pays[0].Get("payment_hash").Exists() && pays[0].Get("created_at").Exists() {
			for i, pay := range pays {
				retval[i] = pay.Value()
			}
		} else {
			filled := make(chan interface{}, len(pays))
			for _, pay := range pays {
				go fillPay(p, pay, filled)
			}
			i := 0
			for fpay := range filled {
				retval[i] = fpay
				i++
			}
		}

		return map[string]interface{}{"pays": retval}, 0, nil
	},
}

func fillPay(p *plugin.Plugin, pay gjson.Result, filled chan<- interface{}) {
	payv := pay.Value().(map[string]interface{})
	if pay.Get("status").String() != "complete" {
		filled <- payv
		return
	}

	preimage, _ := hex.DecodeString(pay.Get("preimage").String())
	hash := sha256.Sum256(preimage)
	hexhash := hex.EncodeToString(hash[:])
	payv["payment_hash"] = hexhash
	res, _ := p.Client.CallNamed("listsendpays", "payment_hash", hexhash)
	payv["created_at"] = res.Get("payments.0.created_at").Int()

	filled <- payv
}

func getChannel(p *plugin.Plugin, peerid string, channel_id string) (resp map[string]interface{}, errCode int, err error) {
	res, err := p.Client.Call("listpeers")
	if err != nil || len(res.Array()) == 0 {
		return nil, 38, errors.New("cannot find peer")
	}
	peer := res.Array()[0]

	var channel gjson.Result
	for _, channel = range peer.Get("channels").Array() {
		if channel.Get("channel_id").String() == channel_id {
			goto found
		}
	}
	return nil, 39, errors.New("cannot find channel")

found:
	ipeer := peer.Value().(map[string]interface{})
	delete(ipeer, "channels")

	return map[string]interface{}{
		"peer": ipeer,
		"chan": channel,
	}, 0, nil
}
