#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
mkdir -p bin
go build -trimpath -o bin/nnx ./cmd/nnx

# 言語設定のない環境ではサーバーがロケールから実効言語を決めるので、E2E は
# 英語に固定する。日本語ロケールの開発機でも画面の文言が変わらないようにする。
LC_ALL=en_US.UTF-8
LC_MESSAGES=en_US.UTF-8
LANG=en_US.UTF-8
export LC_ALL LC_MESSAGES LANG

exec bin/nnx serve --demo --addr 127.0.0.1:0
