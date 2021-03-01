FROM --platform=$TARGETPLATFORM golang:1.13.5-stretch as devel
ARG BUILD_ARGS
COPY / /go/src/
RUN cd /go/src/baetyl-remote-object && make build-local BUILD_ARGS=$BUILD_ARGS

FROM --platform=$TARGETPLATFORM busybox
COPY --from=devel /go/src/baetyl-remote-object/pem/*.pem /etc/ssl/certs/
COPY --from=devel /go/src/baetyl-remote-object/baetyl-remote-object /bin/
ENTRYPOINT ["baetyl-remote-object"]
