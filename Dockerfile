FROM golang:1.26.1 AS builder

WORKDIR /src

COPY . .
RUN go mod download
RUN go build -ldflags="-s -w" -o /out/gtasks-mcp ./cmd/server

FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /auth
COPY --from=builder /out/gtasks-mcp /usr/local/bin/gtasks-mcp

VOLUME ["/auth"]
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/gtasks-mcp"]
