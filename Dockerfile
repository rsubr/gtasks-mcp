FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/app /app
ENTRYPOINT ["/app"]
