@echo off
setlocal enabledelayedexpansion

REM Use the first argument as --program
set PROGRAM=%1
shift

REM Concatenate the remaining arguments with commas for --program-args
set ARGS=
:loop
if "%~1"=="" goto run
if defined ARGS (
  set ARGS=!ARGS!,%~1
) else (
  set ARGS=%~1
)
shift
goto loop

:run
eec run --tag powershell01
