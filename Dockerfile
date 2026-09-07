FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache dependencies separately from source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary: no libc dependency, so it runs on a scratch/distroless base.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/api ./cmd/api

# ---- run ----
FROM alpine:3.20

RUN adduser -D -u 10001 appuser
COPY --from=builder /out/api /usr/local/bin/api

USER appuser
EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/api"]