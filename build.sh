#!/usr/bin/env bash
set -e

# build.sh (depends on dotnet, go, git)

TARGET=$1
MODE=$2

# --- Get current git commit hash (short) ---
BUILD_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

if [ -z "$TARGET" ]; then
    echo "ERROR: Please specify a target: cli / gui / lib"
    echo "Example: ./build.sh cli release"
    exit 1
fi

if [ -z "$MODE" ]; then
    echo "ERROR: Please specify a mode: release or debug"
    echo "Example: ./build.sh cli release"
    exit 1
fi

echo "Target: $TARGET"
echo "Mode: $MODE"
echo "BuildHash: $BUILD_HASH"

# Clean up Go modules
go mod tidy

# Create build directory if not exist
mkdir -p build

# --- Decide extension based on GOOS ---
EXT=".exe"
if [ "$TARGET" = "cli" ]; then
    if [ "$GOOS" = "linux" ]; then
        EXT=""
    fi
fi

if [ "$TARGET" = "lib" ]; then
    if [ "$GOOS" = "darwin" ]; then
        EXT=".dylib"
    elif [ "$GOOS" = "linux" ]; then
        EXT=".so"
    else
        EXT=".dll"
    fi
fi

# Output name by target
if [ "$TARGET" = "cli" ]; then
    OUT="build/eec$EXT"
elif [ "$TARGET" = "gui" ]; then
    OUT="build/geec.exe"
elif [ "$TARGET" = "lib" ]; then
    OUT="build/libcengine$EXT"
else
    echo "ERROR: Invalid target \"$TARGET\". Please specify cli / gui / lib."
    exit 1
fi

# Build process
if [ "$TARGET" = "gui" ]; then
    echo "Building GUI C#..."
    cd gui/csharp/GEEC
    if [ "$MODE" = "release" ]; then
        dotnet publish -c Release -r win-x64 --self-contained true -o ../../../build/
    else
        dotnet publish -c Debug -r win-x64 --self-contained true -o ../../../build/
    fi
    cd ../../..
elif [ "$TARGET" = "cli" ]; then
    cd cli
    if [ "$MODE" = "release" ]; then
        echo "Building Release..."
        go build -ldflags="-s -w -X github.com/m0090-dev/eec-go/core/types.BuildMode=release -X github.com/m0090-dev/eec-go/core/types.BuildHash=$BUILD_HASH" -o "../$OUT"
    elif [ "$MODE" = "debug" ]; then
        echo "Building Debug..."
        go build -gcflags="all=-N -l" -ldflags="-X github.com/m0090-dev/eec-go/core/types.BuildMode=debug -X github.com/m0090-dev/eec-go/core/types.BuildHash=$BUILD_HASH" -o "../$OUT"
    else
        echo "ERROR: Invalid mode \"$MODE\". Please specify release or debug."
        exit 1
    fi
    cd ..
elif [ "$TARGET" = "lib" ]; then
    cd core/cexport
    if [ "$MODE" = "release" ]; then
        echo "Building Release..."
        go build -buildmode=c-shared -ldflags="-s -w -X github.com/m0090-dev/eec-go/core/types.BuildMode=release -X github.com/m0090-dev/eec-go/core/types.BuildHash=$BUILD_HASH" -o "../../$OUT"
    elif [ "$MODE" = "debug" ]; then
        echo "Building Debug..."
        go build -buildmode=c-shared -gcflags="all=-N -l" -ldflags="-X github.com/m0090-dev/eec-go/core/types.BuildMode=debug -X github.com/m0090-dev/eec-go/core/types.BuildHash=$BUILD_HASH" -o "../../$OUT"
    else
        echo "ERROR: Invalid mode \"$MODE\". Please specify release or debug."
        exit 1
    fi
    cd ../..
fi

# If target is cli or gui, also build deleter
if [ "$TARGET" = "cli" ] || [ "$TARGET" = "gui" ]; then
    echo "Building deleter..."
    DEL_OUT="build/eec-deleter$EXT"
    cd deleter
    if [ "$MODE" = "release" ]; then
        go build -ldflags="-s -w -X github.com/m0090-dev/eec-go/core/types.BuildMode=release -X github.com/m0090-dev/eec-go/core/types.BuildHash=$BUILD_HASH" -o "../$DEL_OUT"
    else
        go build -gcflags="all=-N -l" -ldflags="-X github.com/m0090-dev/eec-go/core/types.BuildMode=debug -X github.com/m0090-dev/eec-go/core/types.BuildHash=$BUILD_HASH" -o "../$DEL_OUT"
    fi
    cd ..
    echo "Build completed: $DEL_OUT"
fi

echo "Build completed: $OUT"
