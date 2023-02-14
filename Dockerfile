FROM alpine:3.14
COPY build/adapter /
ENTRYPOINT ["/adapter"]
