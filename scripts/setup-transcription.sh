#!/usr/bin/env bash
set -e

mkdir -p tools
cd tools

if [ ! -d whisper.cpp ]; then
    git clone https://github.com/ggerganov/whisper.cpp
fi

cd whisper.cpp
make

if [ ! -f models/ggml-base.en.bin ]; then
    bash models/download-ggml-model.sh base.en
fi