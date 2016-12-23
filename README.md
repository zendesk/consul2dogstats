consul2dogstats
===============

Collect counts of Consul services by service name, status and tag,
and publishes them to Datadog.

How to build
------------

Just run `make` to make the program.  It will be placed in `bin/consul2dogstats`.

You can set the `GOOS` and `GOARCH` environment variables to cross-compile for
a foreign platform if you prefer.
See https://golang.org/doc/install/source#environment for details on the
permitted values.

Configuration
-------------

The following environment variables can be used to configure `consul2dogstats`:

* `STATSD_ADDR`: Address of the local dogstatsd instance.
  Default: `127.0.0.1:8125`
* `C2D_LOCK_PATH`: Consul key to use for mutex.
  Default: `consul2dogstats/.lock`
* `C2D_COLLECT_INTERVAL`: Amount of time between each collection, expressed as
   a Go duration string.  Default: `10s`
