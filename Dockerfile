FROM alpine

# we need ca-certificates for any external https communication
RUN apk --update upgrade && \
    apk add curl ca-certificates && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*

ADD ./build/nomad-helper-linux-amd64 /nomad-helper
ENTRYPOINT ["/nomad-helper"]