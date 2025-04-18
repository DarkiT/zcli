@echo off
setlocal

:: 获取当前时间
for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /format:list') do set DATETIME=%%I
set YEAR=%DATETIME:~0,4%
set MONTH=%DATETIME:~4,2%
set DAY=%DATETIME:~6,2%
set HOUR=%DATETIME:~8,2%
set MINUTE=%DATETIME:~10,2%
set SECOND=%DATETIME:~12,2%

set BUILD_TIME=%YEAR%-%MONTH%-%DAY% %HOUR%:%MINUTE%:%SECOND%

:: 获取Git信息（如果安装了Git）
where git >nul 2>&1
if %ERRORLEVEL% == 0 (
    for /f %%i in ('git rev-parse --short HEAD 2^>nul') do set GIT_COMMIT=%%i
    for /f %%i in ('git rev-parse --abbrev-ref HEAD 2^>nul') do set GIT_BRANCH=%%i
    for /f %%i in ('git describe --tags --abbrev=0 2^>nul') do set GIT_TAG=%%i
) else (
    set GIT_COMMIT=unknown
    set GIT_BRANCH=unknown
    set GIT_TAG=unknown
)

:: 设置输出文件名
set OUTPUT=zcli.exe

:: 显示构建信息
echo 正在构建 %OUTPUT%...
echo 构建时间: %BUILD_TIME%
echo Git提交: %GIT_COMMIT%
echo Git分支: %GIT_BRANCH%
echo Git标签: %GIT_TAG%

:: 使用ldflags注入编译时间和Git信息
:: 注意: 请将下面的包路径替换为您的实际包路径
go build -ldflags "-X 'github.com/你的用户名/你的项目/zcli.BuildTimeStr=%BUILD_TIME%' -X 'github.com/你的用户名/你的项目/zcli.GitCommit=%GIT_COMMIT%' -X 'github.com/你的用户名/你的项目/zcli.GitBranch=%GIT_BRANCH%' -X 'github.com/你的用户名/你的项目/zcli.GitTag=%GIT_TAG%'" -o %OUTPUT% main.go

echo 构建完成: %OUTPUT% 