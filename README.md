consul2dogstats
===============

Collect counts of Consul services by service name, status and tag,
and publishes them to Datadog.

How to build
------------

```
make
```

Configuration
-------------

The following environment variables can be used to configure `consul2dogstats`:

* `STATSD_ADDR`: Address of the local dogstatsd instance.  Default: `127.0.0.1:8125`
