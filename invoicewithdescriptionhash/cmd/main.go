package main

import (
	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
	"github.com/fiatjaf/sparko/invoicewithdescriptionhash"
)

func main() {
	p := plugin.Plugin{
		Name:    "invoicewithdescriptionhash",
		Version: "v1.2",
		Options: []plugin.Option{},
		RPCMethods: []plugin.RPCMethod{
			invoicewithdescriptionhash.InvoiceWithDescriptionHashMethod,
		},
	}
	p.Run()
}
