# Multi-stage build — Final image ~20 MB
FROM golang:alpine AS builder

WORKDIR /app
COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o remote-wol .

# ---

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/remote-wol /usr/local/bin/remote-wol

EXPOSE 3000
VOLUME /app/data

ENV PORT=3000
ENV DB_PATH=/app/data/wol.db

CMD ["remote-wol"]
