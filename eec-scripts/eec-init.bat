chcp 65001
@echo off
echo env-exec周りの初期化・ビルド中...
call .\env-exec-build.bat
call .\env-exec-deleter-build.bat
call .\env-exec-add-tags.bat 
echo 終了しました。
pause 