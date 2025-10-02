@echo off
chcp 65001 >nul
REM build.bat (depends on dotnet, go,git)
REM Get target from the first argument
set "TARGET=%1"
REM Get mode from the second argument
set "MODE=%2"

if "%TARGET%"=="" (
    echo ERROR: Please specify a target: cli / gui / lib
    echo Example: build.bat cli release
    pause
    exit /b 1
)

if "%MODE%"=="" (
    echo ERROR: Please specify a mode: release or debug
    echo Example: build.bat cli release
    pause
    exit /b 1
)

echo Target: %TARGET%
echo Mode: %MODE%

REM Clean up Go modules
go mod tidy

REM Create build directory in project root if it does not exist
if not exist build (
    mkdir build
)

REM --- Decide extension based on GOOS ---
set "EXT=.exe"

if /i "%TARGET%"=="cli" (
    if /i "%GOOS%"=="linux" (
        set "EXT="
    )
)

if /i "%TARGET%"=="lib" (
    if /i "%GOOS%"=="darwin" (
        set "EXT=.dylib"
    ) else if /i "%GOOS%"=="linux" (
        set "EXT=.so"
    ) else (
        set "EXT=.dll"
    )
)

REM Output name by target
if /i "%TARGET%"=="cli" (
    set "OUT=build\eec%EXT%"
) else if /i "%TARGET%"=="gui" (
    set "OUT=build\geec.exe"
) else if /i "%TARGET%"=="lib" (
    set "OUT=build\libcengine%EXT%"
) else (
    echo ERROR: Invalid target "%TARGET%". Please specify cli / gui / lib.
    pause
    exit /b 1
)

REM Build process
if /i "%TARGET%"=="gui" (
    echo Building GUI C#...
    cd gui\csharp\GEEC
    if /i "%MODE%"=="release" (
        dotnet publish -c Release -r win-x64 --self-contained true -o ..\..\..\build\
    ) else (
        dotnet publish -c Debug -r win-x64 --self-contained true -o ..\..\..\build\
    )
    cd ..\..\..\
) else if /i "%TARGET%"=="cli" (
    cd cli
    if /i "%MODE%"=="release" (
        echo Building Release...
        go build -ldflags="-s -w -X github.com/m0090-dev/eec-go/core/types.BuildMode=release" -o "..\%OUT%"
    ) else if /i "%MODE%"=="debug" (
        echo Building Debug...
        go build -gcflags="all=-N -l" -ldflags="-X github.com/m0090-dev/eec-go/core/types.BuildMode=debug" -o "..\%OUT%"
    ) else (
        echo ERROR: Invalid mode "%MODE%". Please specify release or debug.
        pause
        exit /b 1
    )
    cd ..
) else if /i "%TARGET%"=="lib" (
    cd core\cexport\
    if /i "%MODE%"=="release" (
        echo Building Release...
        go build -buildmode=c-shared -ldflags="-s -w -X github.com/m0090-dev/eec-go/core/types.BuildMode=release" -o "..\..\%OUT%"
    ) else if /i "%MODE%"=="debug" (
        echo Building Debug...
        go build -buildmode=c-shared -gcflags="all=-N -l" -ldflags="-X github.com/m0090-dev/eec-go/core/types.BuildMode=debug" -o "..\..\%OUT%"
    ) else (
        echo ERROR: Invalid mode "%MODE%". Please specify release or debug.
        pause
        exit /b 1
    )
    cd ..\..\
)

REM If target is cli or gui, also build deleter
if /i "%TARGET%"=="cli" (
    call :BuildDeleter
) else if /i "%TARGET%"=="gui" (
    call :BuildDeleter
)

echo Build completed: %OUT%
pause
exit /b 0

:BuildDeleter
    echo Building deleter...
    set "DEL_OUT=build\eec-deleter%EXT%"
    cd deleter
    if /i "%MODE%"=="release" (
        go build -ldflags="-s -w -X github.com/m0090-dev/eec-go/core/types.BuildMode=release" -o "..\%DEL_OUT%"
    ) else (
        go build -gcflags="all=-N -l" -ldflags="-X github.com/m0090-dev/eec-go/core/types.BuildMode=debug" -o "..\%DEL_OUT%"
    )
    cd ..
    echo Build completed: %DEL_OUT%
    exit /b 0

