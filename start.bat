@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

cd /d %~dp0

set "SERVER_ADDR="
set "REMOTE_PORT="

for /f "tokens=1,* delims==" %%A in ('findstr /r /c:"^serverAddr[ ]*=" frpc.toml') do (
	set "SERVER_ADDR=%%~B"
)

for /f "tokens=1,* delims==" %%A in ('findstr /r /c:"^remotePort[ ]*=" frpc.toml') do (
	if not defined REMOTE_PORT set "REMOTE_PORT=%%~B"
)

set "SERVER_ADDR=!SERVER_ADDR: =!"
set "SERVER_ADDR=!SERVER_ADDR:"=!"
set "REMOTE_PORT=!REMOTE_PORT: =!"
set "REMOTE_PORT=!REMOTE_PORT:"=!"

echo(!REMOTE_PORT!| findstr /r "^[0-9][0-9]*$" >nul
if errorlevel 1 (
	echo frpc.toml 中 remotePort 不是有效数字
	exit /b 1
)

if !REMOTE_PORT! LSS 25565 (
	echo frpc.toml 中 remotePort 超出范围，要求 25565-25575
	exit /b 1
)

if !REMOTE_PORT! GTR 25575 (
	echo frpc.toml 中 remotePort 超出范围，要求 25565-25575
	exit /b 1
)

if defined SERVER_ADDR if defined REMOTE_PORT (
	echo 使用 !SERVER_ADDR!:!REMOTE_PORT! 连接服务器
) else (
	echo 未能从 frpc.toml 读取 serverAddr 或 remotePort
)

frpc.exe -c frpc.toml
set "FRPC_EXIT_CODE=!errorlevel!"

if not "!FRPC_EXIT_CODE!"=="0" (
	echo frpc 启动失败，退出码: !FRPC_EXIT_CODE!
	echo 启动失败原因: 请检查 frpc 日志中的 error 关键字。
	echo 常见原因: token 错误、serverAddr/serverPort 不可达、remotePort 被占用或服务端未放行。
	pause
	exit /b !FRPC_EXIT_CODE!
)

echo frpc 启动成功
pause