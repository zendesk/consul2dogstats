consul2dogstats
===============

Collect counts of Consul services by service name, status and tag,
and publishes them to Datadog.

How to build
------------

Just run `make bin` to make the program.  It will be placed in `bin/consul2dogstats`.

You can set the `GOOS` and `GOARCH` environment variables to cross-compile for
a foreign platform if you prefer.  See
https://golang.org/doc/install/source#environment for details on the permitted
values.

Configuration
-------------

The following environment variables can be used to configure `consul2dogstats`:

* `STATSD_ADDR`: Address of the local dogstatsd instance.
  Default: `127.0.0.1:8125`
* `C2D_LOCK_PATH`: Consul key to use for mutex.
  Default: `consul2dogstats/.lock`
* `C2D_COLLECT_INTERVAL`: Amount of time between each collection, expressed as
   a Go duration string.  Default: `10s`
* `CONSUL_HTTP_ADDR`: The address of the Consul agent (default: `127.0.0.1:8500`)
* `CONSUL_HTTP_SSL`: If set, connect to the server using TLS (default: unset/no TLS)
* `CONSUL_HTTP_TOKEN`: The API token used to authenticate to the Consul agent (optional, default: none)
* `CONSUL_CACERT`: Path to CA file to use for talking to Consul over TLS (default: none)
* `CONSUL_CAPATH`: Path to a directory of CA certs to use for talking to Consul over TLS (default: none)
* `CONSUL_CLIENT_CERT`: Path to a client cert file to use for talking to Consul over TLS (default: none)
* `CONSUL_CLIENT_KEY`: Path to a client key file to use for talking to Consul over TLS (default: none)
* `CONSUL_TLS_SERVER_NAME`: Server name to use as the SNI host when connecting via TLS (default: none)
* `CONSUL_HTTP_SSL_VERIFY`: If set to 0, disable TLS certificate verification (default: unset; perform verification)

Development
-----------

Run `make test` to run all tests.