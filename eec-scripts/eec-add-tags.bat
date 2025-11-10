@echo off
chcp 65001
echo Setting tags...

REM Set base configuration directory
set "USER_EEC_CONFIG_DIR=D:\win\program\go\main-project\eec-configs"

REM IDE-specific tags
call eec tag add android-studio00 -c "%USER_EEC_CONFIG_DIR%\ide\android-studio\android-studio.toml" -p "D:\win\dev-tools\android\android-studio\bin\studio64"
REM Example Unity tag (uncomment if needed)
REM call eec tag add unity00 -c "%USER_EEC_CONFIG_DIR%\ide\unity\unity-dev.toml" -p "D:\Program Files\Unity\Hub\Editor\2023.1.0f1\Editor\Unity.exe"

REM Tool-specific tags
call eec tag add easy-up -c "%USER_EEC_CONFIG_DIR%\others\easy-uploader.toml" -p "D:\win\program\go\main-project\google-drive-easy-uploader\build\easy-up"

REM Base development configuration
call eec tag add dev-base -i "%USER_EEC_CONFIG_DIR%\base\base-dev.toml"

REM General tool development
call eec tag add dev-tools -i "%USER_EEC_CONFIG_DIR%\tools\use-tools-dev.toml,%USER_EEC_CONFIG_DIR%\tools\gnu-tools-dev.toml"

REM Language-specific development setup
call eec tag add dev-lang -i "%USER_EEC_CONFIG_DIR%\lang\go\go-dev.toml,%USER_EEC_CONFIG_DIR%\lang\rust\rust-dev.toml,%USER_EEC_CONFIG_DIR%\lang\java\java-dev.toml,%USER_EEC_CONFIG_DIR%\lang\r\r-dev.toml,%USER_EEC_CONFIG_DIR%\lang\ruby\ruby-dev.toml,%USER_EEC_CONFIG_DIR%\lang\python\python-dev.toml,%USER_EEC_CONFIG_DIR%\lang\nim\nim-dev.toml,%USER_EEC_CONFIG_DIR%\lang\dotnet\dotnet-dev.toml,%USER_EEC_CONFIG_DIR%\lang\mingw\mingw-dev.toml,%USER_EEC_CONFIG_DIR%\lang\nasm\nasm-dev.toml,%USER_EEC_CONFIG_DIR%\lang\haxe\haxe-dev.toml,%USER_EEC_CONFIG_DIR%\lang\haxe\.env,%USER_EEC_CONFIG_DIR%\lang\swift\swift-dev.toml"

REM Aggregate all development configurations
call eec tag add dev -i "dev-base,dev-lang,dev-tools"

REM Command prompt dev setups
call eec tag add dev-cmd00 -i dev -p cmd -a "/K checkitems %USER_EEC_CONFIG_DIR%\checkitems.csv"
call eec tag add dev-cmd01 -i dev -p cmd -a "/K cd D:\win\program && D: && checkitems %USER_EEC_CONFIG_DIR%\checkitems.csv"

REM PowerShell dev setups
call eec tag add dev-shell00 -i dev -p powershell -a "-NoExit,-Command,Set-ExecutionPolicy RemoteSigned -Scope Process; checkitems %USER_EEC_CONFIG_DIR%\checkitems.csv"
call eec tag add dev-shell01 -i dev -p powershell -a "-NoExit,-Command,Set-ExecutionPolicy RemoteSigned -Scope Process; checkitems %USER_EEC_CONFIG_DIR%\checkitems.csv; Set-Location -Path 'D:\win\program\'"

echo Tag setup completed
pause
