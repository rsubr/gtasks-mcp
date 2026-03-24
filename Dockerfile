FROM gcr.io/distroless/static-debian13:nonroot

ARG TARGETARCH

WORKDIR /auth
COPY --chown=nobody:nobody --chmod=0755 dist/gtasks-mcp-linux-${TARGETARCH} /usr/local/bin/gtasks-mcp

USER nobody:nobody

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/gtasks-mcp"]
