@echo off
chcp 65001
REM set "eec_deleter=D:\win\program\go\main-project\eec\build\eec-deleter"
REM set "eec_exe=D:\win\program\go\main-project\eec\build\eec"
set "PATH=D:\win\program\go\main-project\eec\build\;%PATH%
set "eec_deleter=eec-deleter"
set "eec_exe=eec"

tasklist /FI "IMAGENAME eq %eec_deleter%" /NH | find /I "%eec_deleter%" >nul

if "%1"=="run" (
    rem エラーレベルが 0（すでに実行中）か確認
    if %ERRORLEVEL% equ 0 (
        echo [%eec_deleter%] は既に実行中です。
    ) else (
        echo [%eec_deleter%] を起動します…
        powershell -WindowStyle Normal -Command "Start-Process -FilePath '%eec_deleter%' -WindowStyle Hidden"
    )
)
%eec_exe% %*