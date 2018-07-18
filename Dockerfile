FROM alpine:3.8

# Adding ca-certificates for external communication, and openssh
# for attaching to remote nodes
RUN apk add --update curl ca-certificates openssh-client && \
    rm -rf /var/cache/apk/*

ADD ./build/nomad-helper-linux-amd64 /nomad-helper
RUN chmod +x /nomad-helper

ENTRYPOINT ["/nomad-helper"]
