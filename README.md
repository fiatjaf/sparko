# The `sparko` plugin.

The famous [Spark wallet](https://github.com/shesek/spark-wallet) repackaged as a single-binary lightningd plugin.

This works either as a personal wallet with a nice UI (see link above) or as a **full-blown HTTP-RPC bridge to your node that can be used to develop apps**.

It has some differences (advantages?) over the original Spark wallet:

  * Single binary: No dependencies to manage, just grab the manage and throw it in your `lightningd` plugins folder.
  * Runs as a plugin: this means you don't have to manage the server, it will be managed by `lightningd` and will always be running as long as your node is running.
  * Multiple keys with fine-grained permissions: create keys that can only call some methods.
  * Centralized options management: since it runs as a plugin all options are read from your `lightningd` config file.
  * Written in Go: lean, fast, relatively low on memory, doesn't require installing Node.js and a ton of dependencies.
  * Unrestricted: any method can be called through the HTTP/JSON-RPC interface, including any methods provided by plugins you might have active in your node.
  * Event streaming: makes it easy to write remote Lightning apps by exposing a [SSE stream](#listen-to-events) of all events (payments sent, received etc.) that happen on the node.
  * No default login: you don't have to expose "super user" credentials over your node. You can have only access-keys to specific methods. But you can define a login an password too, of course.

# How to install

This is distributed as a single binary for your delight (or you can compile it yourself with `go get`, or ask me for binaries for other systems if you need them).

[Download it](https://github.com/fiatjaf/sparko/releases), call `chmod +x <binary>` and put it in `~/.lightning/plugins` (create that directory if it doesn't exist).

You only need the binary you can get in [the releases page](https://github.com/fiatjaf/sparko/releases), nothing else.

# How to use

Just configure the options you want in you `~/.lightning/config` file, like the following:

```shell
sparko-host=0.0.0.0
sparko-port=9737

# the tls path is just the directory where your self-signed key and certificate are.
# (see below for code snippets that generate them on Linux)
# the path is relative to your lightning-dir, so "sparko-tls" will translate to "~/.lightning/bitcoin/sparko-tls/"
# (you can also use an absolute path)
# if not specified the app will run without TLS (as http://)
sparko-tls-path=sparko-tls

# login credentials for using the wallet app.
# under the hood these are translated into an access key with full access.
# the default login is none, which doesn't allow you to use the wallet app,
#   but you can still use the /rpc endpoint with other keys specified at sparko-keys=
sparko-login=mywalletusername:mywalletpassword

# a list of semicolon-separated pairs of keys:permissions
#   - each possible callable RPC method is a permission.
#   - 'stream' is a special method that gives access to the SSE stream at /stream.
#   - just writing the key and nothing else means that key has all permissions.
#   - keys must be secret and random.
sparko-keys=masterkeythatcandoeverything; secretaccesskeythatcanreadstuff: getinfo, listchannels, listnodes; verysecretkeythatcanpayinvoices: pay; keythatcanlistentoallevents: stream
# for the example above the initialization logs (mixed with lightningd logs) should print something like
2019/09/27 00:48:46 plugin-sparko Keys read: masterkeythatcandoeverything (full-access), secretaccesskeythatcanreadstuff (3 permission), verysecretkeythatcanpayinvoices(1 permission), keythatcanlistentoallevents (1 permission)
```

To use TLS with a self-signed certificate (`https://`), generate your certificate first:

```
mkdir ~/.lightning/bitcoin/sparko-tls
cd ~/.lightning/bitcoin/sparko-tls/
openssl genrsa -out key.pem 2048
openssl req -new -x509 -sha256 -key key.pem -out cert.pem -days 3650
```

To use a certificate signed by LetsEncrypt, you must be able to bind to ports 80 and 443, which generally requires running as root. Specify options like the following:

```shell
sparko-host=sparko.mydomain.com
sparko-tls-path=sparko-letsencrypt
sparko-letsencrypt-email=myemail@gmail.com
```

Then try to visit `http://sparko.mydomain.com/`. If all is well you should get redirected to the `https://` page, if something is wrong it should appear on the logs.

To expose Sparko over CORS (who knows why), add `sparko-allow-cors=true` to the config file.

## Errors

When starting `lightningd`, check the logs for errors regarding `sparko` initialization, they will be prefixed with `"plugin-sparko"`.

## Call the HTTP RPC

Replace the following with your actual values:

```
curl -k https://0.0.0.0:9737/rpc -d '{"method": "pay", "params": ["lnbc..."]}' -H 'X-Access: masterkeythatcandoeverything'
```

See also [a list of client libraries](#client-libraries).

### `Range` headers

You can also limit the number of things you're returning. For example, `listinvoices` and `listsendpays` tend to get out of hand quickly and you may not want to return all your invoices and payments. You can add a `Range` header to solve this issue:

```
curl -k https://0.0.0.0:9737/rpc -d '{"method": "listsendpays"}' -H 'X-Access: masterkeythatcandoeverything' -H 'Range: payments=0-99'
```

The above means that `sparko` will take the response it gets from `lightningd` and slice the array contained in the key `"payments"` to get values between 0 and 99, i.e., the first 100 payments. You could get the last 50 payments, for example, by passing `-H 'Range: payments=-50'` and so on. This is method-agnostic (that's why you must supply the `payments=` parameter), so you can use it on other methods and even methods provided by other plugins.

## Listen to events

Sparko exposes a [SSE](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) endpoint at `/stream` that emits [all events](https://lightning.readthedocs.io/PLUGINS.html#event-notifications) a plugin may receive, in raw format given by lightningd. In some cases that's what you want when developing applications that must talk to a Lightning node remotely, better than webhooks. There are libraries for listening to Server-Sent Events in all languages. The `/stream` endpoint requires the `stream` permission to be accessed.

## Client libraries

 * [JavaScript](https://github.com/fiatjaf/sparko-client) (Node.js and the browser)
 * [Go](https://pkg.go.dev/github.com/fiatjaf/lightningd-gjson-rpc?tab=doc#Client) (initialize it with the Spark URL and key/token instead of a lightning-rpc socket path)

## Polar integration

 * There is an arm64 CLN w/ sparko plugin polar image [here](https://hub.docker.com/r/chongjin/polar-cln-sparko)
 * The instruction for installation & usage is on the page

## Open the wallet UI

This is the same code used in [Spark wallet](https://github.com/shesek/spark-wallet).

Visit `https://0.0.0.0:9737/`. Only available if `sparko-login` is provided.

## Built with [github.com/fiatjaf/lightningd-gjson-rpc](https://pkg.go.dev/github.com/fiatjaf/lightningd-gjson-rpc/plugin?tab=doc)
