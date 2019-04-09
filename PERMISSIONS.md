Permissions
===========

In your config file at `~/.config/sparko.{yaml,toml,json}` you can have an entry called `keys`. Each subkey there can be an access key of your choosing (be sure to generate a fairly random key), and each of these subkeys can have

  * `disallowed_methods`: a list of RPC methods this key will be able to call. All other are allowed.
  * `allowed_methods`: a list of RPC methods this key will be able to call. If there's any method listed here only the allowed methods will be allowed and all other will be disallowed (thus the `disallowed_methods` list is ignored).
  * `pay_max_satoshis`: if set, this setting automatically disallows `sendpay` (which can be _ by listing it in `allowed_methods`, but you don't want that mess) and causes calls to `pay` that exceed this limit to be denied.
  * `profile`: the name of a predefined permissions profile. You can find a list of the profiles at [permissions.go](permissions.go).

Example
-------

Having a config file like this at `~/.config/sparko.yaml`:

```yaml
login: user:password
keys:
  veryrandomsecret1234:
    disallowed_methods:
      - fundchannel
      - close
      - newaddr
      - withdraw
  superrandomsecret987:
    allowed_methods:
      - invoice
      - waitinvoice
      - decodepay
      - pay
      - listpayments
    pay_max_satoshis: 50000
  randomsecret456:
    profile: readonly
print-key: true
```

Should print on start-up:

```
[INFO] Access keys:
  DHu6nFxzaO7HFtenx4locFcV8vqHePCjUjixoVAMEQ (default/login): full-access
  verysecretrandomkey1234noblockchain: disallowed={newaddr,withdraw,fundchannel,close}
  superrandomsecret78910basicln: allowed={listpayments,invoice,waitinvoice,decodepay,pay} pay-max=50000
  keythatshouldberandom223344readonly: allowed={listchannels,feerates,waitanyinvoice,waitsendpay,getinfo,listpayments,getroute,decodepay,paystatus,waitinvoice,listfunds,listinvoices,listnodes,listpeers}
```
