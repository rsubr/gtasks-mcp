FROM golang:1.26.1 AS builder

WORKDIR /src

COPY . .
RUN go mod download
RUN go build -ldflags="-s -w" -o /out/gtasks-mcp ./cmd/server

FROM debian:bookworm-slim

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/* \
	&& mkdir -p /auth \
	&& chown nobody:nogroup /auth

WORKDIR /auth
COPY --from=builder /out/gtasks-mcp /usr/local/bin/gtasks-mcp

USER nobody:nogroup

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/gtasks-mcp"]
