#!/bin/bash
# -*- coding: utf-8 -*-

# 第一引数でモード取得
MODE="$1"

if [ -z "$MODE" ]; then
    echo "ERROR: モードを指定してください release or debug"
    echo "例: ./build.sh release"
    exit 1
fi

echo "モード: $MODE"

# モジュール整理
go mod tidy

# build ディレクトリ作成（存在しない場合）
if [ ! -d "build" ]; then
    mkdir build
fi

# ビルド処理
if [ "$MODE" = "release" ]; then
    echo "リリースビルド中..."
    go build -ldflags="-s -w -X main/ext.BuildMode=release" -o build/eec main.go
elif [ "$MODE" = "debug" ]; then
    echo "デバッグビルド中..."
    go build -gcflags="all=-N -l" -ldflags="-X main/ext.BuildMode=debug" -o build/eec main.go
else
    echo "ERROR: 無効なモード \"$MODE\". release または debug を指定してください。"
    exit 1
fi

echo "ビルド完了: build/eec"
