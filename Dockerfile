FROM gcr.io/distroless/static-debian13

ARG TARGETARCH

WORKDIR /auth
COPY --chown=www-data:www-data --chmod=0755 dist/gtasks-mcp-linux-${TARGETARCH} /usr/local/bin/gtasks-mcp

USER nobody:nobody

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/gtasks-mcp"]
