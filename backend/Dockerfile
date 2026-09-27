FROM golang:1.27-alpine AS build
WORKDIR /src

# go.mod/go.sum: Docker cachea la descarga de dependencias
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed


FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H app
WORKDIR /app
COPY --from=build /out/server /out/seed ./
USER app

ENV GIN_MODE=release APP_PORT=8080 DATA_DIR=/app/data
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=5 \
  CMD wget -qO- http://localhost:8080/health || exit 1
CMD ["./server"]
