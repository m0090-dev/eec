@echo off
chcp 65001
echo Setting tags...

REM call eec tag add powershell00 --config-file "%USER_EEC_CONFIG_DIR%\eec-config.toml" --program "powershell"  --program-args="-NoExit","-Command","Set-ExecutionPolicy RemoteSigned -Scope Process; checkitems %USER_EEC_CONFIG_DIR%\checkitems.csv"
REM call eec tag add powershell01 --config-file "%USER_EEC_CONFIG_DIR%\eec-config.toml" --program "powershell"  --program-args="-NoExit","-Command","Set-ExecutionPolicy RemoteSigned -Scope Process; checkitems %USER_EEC_CONFIG_DIR%\checkitems.csv; Set-Location -Path 'D:\win\program\'"
REM call eec tag add cmd00 --config-file "%USER_EEC_CONFIG_DIR%\eec-config.toml"  --program "cmd" --program-args="/K checkitems %USER_EEC_CONFIG_DIR%\checkitems.csv"
REM call eec tag add cmd01 --config-file "%USER_EEC_CONFIG_DIR%\eec-config.toml" --program "cmd" --program-args="/K cd D:\win\program && D: && checkitems %USER_EEC_CONFIG_DIR%\checkitems.csv"

set "USER_EEC_CONFIG_DIR=D:\win\program\go\main-project\eec\eec-configs"

call eec tag add android-studio00 --config-file "%USER_EEC_CONFIG_DIR%\android-studio.toml" --program "D:\win\dev-tools\android\android-studio\bin\studio64"
call eec tag add easy-up --program "D:\win\program\go\main-project\google-drive-easy-uploader\build\easy-up"
call eec tag add go-dev --config-file "%USER_EEC_CONFIG_DIR%\go-dev.toml" --import "%USER_EEC_CONFIG_DIR%\base-dev.toml" 

REM call eec tag add dev --import "%USER_EEC_CONFIG_DIR%\base-dev.toml" --import "%USER_EEC_CONFIG_DIR%\go-dev.toml" --import "%USER_EEC_CONFIG_DIR%\rust-dev.toml" --import "%USER_EEC_CONFIG_DIR%\java-dev.toml" --import "%USER_EEC_CONFIG_DIR%\r-dev.toml" --import "%USER_EEC_CONFIG_DIR%\ruby-dev.toml" --import "%USER_EEC_CONFIG_DIR%\python-dev.toml" --import "%USER_EEC_CONFIG_DIR%\free-basic-dev.toml" --import "%USER_EEC_CONFIG_DIR%\mingw-dev.toml" --import "%USER_EEC_CONFIG_DIR%\nim-dev.toml" --import "%USER_EEC_CONFIG_DIR%\dotnet-dev.toml" --import  "%USER_EEC_CONFIG_DIR%\use-tools-dev.toml" --import "%USER_EEC_CONFIG_DIR%\gnu-tools-dev.toml" 

REM Base configuration
call eec tag add dev-base --import "%USER_EEC_CONFIG_DIR%\base-dev.toml"

call eec tag add dev-tools --import "%USER_EEC_CONFIG_DIR%\use-tools-dev.toml" ^
                           --import "%USER_EEC_CONFIG_DIR%\gnu-tools-dev.toml"

REM Language-specific development setup
call eec tag add dev-lang --import "%USER_EEC_CONFIG_DIR%\go-dev.toml" ^
                          --import "%USER_EEC_CONFIG_DIR%\rust-dev.toml" ^
                          --import "%USER_EEC_CONFIG_DIR%\java-dev.toml" ^
                          --import "%USER_EEC_CONFIG_DIR%\r-dev.toml" ^
                          --import "%USER_EEC_CONFIG_DIR%\ruby-dev.toml" ^
                          --import "%USER_EEC_CONFIG_DIR%\python-dev.toml" ^
                          --import "%USER_EEC_CONFIG_DIR%\nim-dev.toml" ^
                          --import "%USER_EEC_CONFIG_DIR%\dotnet-dev.toml" ^
                          --import "%USER_EEC_CONFIG_DIR%\mingw-dev.toml" ^
                          --import "%USER_EEC_CONFIG_DIR%\nasm-dev.toml" ^

REM Finally, dev aggregates the intermediate categories
call eec tag add dev --import dev-base --import dev-lang --import dev-tools

call eec tag add dev-cmd00 --import dev --program cmd --program-args="/K checkitems %USER_EEC_CONFIG_DIR%\checkitems.csv"
call eec tag add dev-cmd01 --import dev --program cmd --program-args="/K cd D:\win\program && D: && checkitems %USER_EEC_CONFIG_DIR%\checkitems.csv"

call eec tag add dev-shell00 --import dev --program powershell  --program-args="-NoExit","-Command","Set-ExecutionPolicy RemoteSigned -Scope Process; checkitems %USER_EEC_CONFIG_DIR%\checkitems.csv"
call eec tag add dev-shell01 --import dev --program powershell --program-args="-NoExit","-Command","Set-ExecutionPolicy RemoteSigned -Scope Process; checkitems %USER_EEC_CONFIG_DIR%\checkitems.csv; Set-Location -Path 'D:\win\program\'"

echo Tag setup completed
REM Wait for key input
pause
