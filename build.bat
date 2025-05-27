@echo off
rem ------------------------- 配置部分 -------------------------
rem 定义应用程序名称
set APP_NAME=jmeter

rem 定义输出目录
set OUTPUT_DIR=bin

rem 自定义 main.go 的路径
set MAIN_PATH=run.go

rem ------------------------- 检查并创建输出目录 -------------------------
rem 检查并创建输出目录
if not exist %OUTPUT_DIR% (
    mkdir %OUTPUT_DIR%
)

rem ------------------------- 编译不同平台 -------------------------

rem 编译为 Windows 可执行文件
echo 编译 Windows 可执行文件...
set GOOS=windows
set GOARCH=amd64
go build -o %OUTPUT_DIR%\%APP_NAME%-windows-amd64.exe %MAIN_PATH%

rem 编译为 Linux 可执行文件
echo 编译 Linux 可执行文件...
set GOOS=linux
set GOARCH=amd64
go build -o %OUTPUT_DIR%\%APP_NAME%-linux-amd64 %MAIN_PATH%

rem 编译为 macOS 可执行文件
echo 编译 macOS 可执行文件...
set GOOS=darwin
set GOARCH=amd64
go build -o %OUTPUT_DIR%\%APP_NAME%-darwin-amd64 %MAIN_PATH%

rem 编译为 ARM 架构 Linux 可执行文件
echo 编译 ARM Linux 可执行文件...
set GOOS=linux
set GOARCH=arm64
go build -o %OUTPUT_DIR%\%APP_NAME%-linux-arm64 %MAIN_PATH%

rem 编译为 32 位 Windows 可执行文件
echo 编译 32 位 Windows 可执行文件...
set GOOS=windows
set GOARCH=386
go build -o %OUTPUT_DIR%\%APP_NAME%-windows-386.exe %MAIN_PATH%

rem 完成
echo 所有平台的可执行文件已经生成。
pause
