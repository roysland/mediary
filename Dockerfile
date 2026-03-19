# syntax=docker/dockerfile:1

FROM golang:1.25-bookworm AS build

WORKDIR /src

# CGO is required by github.com/mattn/go-sqlite3.
RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential pkg-config \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -trimpath -ldflags='-s -w' -o /out/server ./cmd/server

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates ffmpeg \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /out/server /app/server
COPY --from=build /src/db /app/db
COPY --from=build /src/internal/views /app/internal/views
COPY --from=build /src/web/static /app/web/static

RUN mkdir -p /app/data/audio

ENV APP_ENV=production \
    LISTEN_ADDR=:8080 \
    AUDIO_STORAGE_DIR=/app/data/audio \

EXPOSE 8080
VOLUME ["/app/data"]

CMD ["/app/server"]
