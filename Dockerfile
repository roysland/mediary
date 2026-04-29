# syntax=docker/dockerfile:1

FROM golang:1.25-bookworm AS build

WORKDIR /src

# CGO is required by github.com/mattn/go-sqlite3.
RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential pkg-config \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum* ./
RUN go mod download

COPY cmd ./cmd
COPY db ./db
COPY internal ./internal
COPY web/static ./web/static
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -trimpath -ldflags='-s -w' -o /out/server ./cmd/server

# NEW: Build whisper.cpp inside the container environment
FROM debian:bookworm-slim AS whisper-build
RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential cmake git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /whisper-src
RUN git clone --depth=1 https://github.com/ggerganov/whisper.cpp.git .

# The magic flag is -DBUILD_SHARED_LIBS=OFF
RUN cmake -B build \
    -DWHISPER_BUILD_EXAMPLES=ON \
    -DBUILD_SHARED_LIBS=OFF \
    -DGGML_OPENMP=OFF && \
    cmake --build build --config Release -j$(nproc)

# FINAL RUNTIME STAGE
FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl ffmpeg libgomp1 \
    && rm -rf /var/lib/apt/lists/*

ARG BUILD_VERSION=dev

WORKDIR /app

RUN groupadd --system app \
    && useradd --system --gid app --home-dir /app --create-home --shell /usr/sbin/nologin app

COPY --from=build /out/server /app/server
COPY --from=build /src/db /app/db
COPY --from=build /src/internal/views /app/internal/views
COPY --from=build /src/web/static /app/web/static
COPY --from=whisper-build /whisper-src/build/bin/whisper-cli /usr/local/bin/whisper-cli

RUN mkdir -p /app/data/audio /app/data/models \
    && chown -R app:app /app

ENV APP_ENV=production \
    BUILD_VERSION=${BUILD_VERSION} \
    LISTEN_ADDR=:8080 \
    DB_PATH=/app/data/app.db \
    AUDIO_STORAGE_DIR=/app/data/audio \
    WHISPER_BINARY_PATH=/usr/local/bin/whisper-cli \
    WHISPER_MODEL_PATH=/app/data/models/ggml-small.bin

EXPOSE 8080
VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl --fail --silent http://127.0.0.1:8080/healthz || exit 1

USER app

CMD ["/app/server"]
