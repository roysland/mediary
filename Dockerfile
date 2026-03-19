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

# NEW: Build whisper.cpp inside the container environment
FROM debian:bookworm-slim AS whisper-build
RUN apt-get update && apt-get install -y build-essential cmake git
WORKDIR /whisper-src
# Clone a specific version or copy your local source
RUN git clone https://github.com/ggerganov/whisper.cpp.git .
RUN cmake -B build -DWHISPER_BUILD_EXAMPLES=ON && cmake --build build --config Release

# FINAL RUNTIME STAGE
FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates ffmpeg libstdc++6 libc6 libgomp1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /out/server /app/server
COPY --from=build /src/db /app/db
COPY --from=build /src/internal/views /app/internal/views
COPY --from=build /src/web/static /app/web/static
COPY --from=whisper-build /whisper-src/build/bin/whisper-cli /usr/local/bin/whisper-cli

RUN mkdir -p /app/data/audio

ENV APP_ENV=production \
    LISTEN_ADDR=:8080 \
    AUDIO_STORAGE_DIR=/app/data/audio

EXPOSE 8080
VOLUME ["/app/data"]

CMD ["/app/server"]
