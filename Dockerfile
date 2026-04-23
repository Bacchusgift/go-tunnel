FROM alpine:3.19

COPY bin/go-tunnel-server /usr/local/bin/go-tunnel-server

EXPOSE 8080

ENTRYPOINT ["go-tunnel-server"]
CMD ["-addr", ":8080"]
