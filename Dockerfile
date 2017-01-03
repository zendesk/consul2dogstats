FROM busybox
ADD bin/consul2dogstats /bin/
USER nobody
ENTRYPOINT ["/bin/consul2dogstats"]
