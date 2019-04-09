package main

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func readPermissionsConfig() (Keys, error) {
	keys := make(Keys)
	err := viper.UnmarshalKey("keys", &keys)
	if err != nil {
		return keys, err
	}

	for key, permissions := range keys {
		// setting a profile trumps other custom settings
		if permissions.Profile != nil {
			permissions = Profiles[*permissions.Profile]
		} else {
			permissions.AllowedMethods = make(map[string]bool)
			permissions.DisallowedMethods = make(map[string]bool)
			for _, method := range permissions.AllowedMethodsList {
				permissions.AllowedMethods[method] = true
			}
			for _, method := range permissions.DisallowedMethodsList {
				permissions.DisallowedMethods[method] = true
			}

			if permissions.PayMaxSatoshis != nil {
				permissions.DisallowedMethods["sendpay"] = true
			}
		}
		keys[key] = permissions
	}

	return keys, err
}

type Keys map[string]PermissionSet

type PermissionSet struct {
	AllowedMethodsList    []string        `mapstructure:"allowed_methods"`
	DisallowedMethodsList []string        `mapstructure:"disallowed_methods"`
	AllowedMethods        map[string]bool `mapstructure:"-"`
	DisallowedMethods     map[string]bool `mapstructure:"-"`

	Profile *string `mapstructure:"profile"`

	PayMaxSatoshis *int `mapstructure:"pay_max_satoshis"`
}

func (ps PermissionSet) String() string {
	s := ""

	if len(ps.AllowedMethods) > 0 {
		methods := make([]string, len(ps.AllowedMethods))
		i := 0
		for method, _ := range ps.AllowedMethods {
			methods[i] = method
			i++
		}
		s += "allowed={" + strings.Join(methods, ",") + "}"
	} else if len(ps.DisallowedMethods) > 0 {
		methods := make([]string, len(ps.DisallowedMethods))
		i := 0
		for method, _ := range ps.DisallowedMethods {
			methods[i] = method
			i++
		}
		s += "disallowed={" + strings.Join(methods, ",") + "}"
	} else {
		s += "full-access"
	}

	if ps.PayMaxSatoshis != nil {
		s += fmt.Sprintf(" pay-max=%d", *ps.PayMaxSatoshis)
	}

	return s
}

var ReadOnly = PermissionSet{
	AllowedMethods: map[string]bool{
		"getinfo":        true,
		"listforwards":   true,
		"listfunds":      true,
		"listpayments":   true,
		"listinvoices":   true,
		"listnodes":      true,
		"listpeers":      true,
		"listchannels":   true,
		"getroute":       true,
		"feerates":       true,
		"waitinvoice":    true,
		"waitanyinvoice": true,
		"decodepay":      true,
		"paystatus":      true,
		"waitsendpay":    true,
	},
}

var Invoices = PermissionSet{
	AllowedMethods: map[string]bool{
		"invoice":     true,
		"waitinvoice": true,
	},
}

var MicroPayer = PermissionSet{
	AllowedMethods: map[string]bool{
		"decodepay":    true,
		"pay":          true,
		"paystatus":    true,
		"listpayments": true,
		"waitsendpay":  true,
	},
	PayMaxSatoshis: pointToInt(100000),
}

var Profiles = map[string]PermissionSet{
	"readonly":   ReadOnly,
	"invoices":   Invoices,
	"micropayer": MicroPayer,
}

func pointToInt(v int) *int { return &v }
