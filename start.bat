@echo off
setlocal enabledelayedexpansion

:: 设置颜色代码
set "GREEN=92"
set "BLUE=94"
set "RED=91"

:: 显示彩色输出的函数
call :print_blue "检查必要的命令..."

:: 检查命令是否存在
call :check_command "node"
call :check_command "npm"
call :check_command "python"
call :check_command "pip"

:: 安装 Node.js 依赖
call :print_blue "安装 Node.js 依赖..."
call npm install
if %ERRORLEVEL% neq 0 (
    call :print_red "Node.js 依赖安装失败！"
    exit /b 1
)
call :print_green "Node.js 依赖安装成功！"

:: 安装 Python 依赖
call :print_blue "安装 Python 依赖..."
:: 检查 requirements.txt 是否存在
if exist "requirements.txt" (
    pip install -r requirements.txt
) else if exist ".\public\requirements.txt" (
    pip install -r .\public\requirements.txt
) else (
    :: 如果没有 requirements.txt，安装常见的依赖
    call :print_blue "未找到 requirements.txt，安装常见的 Python 依赖..."
    pip install flask requests websockets
)

if %ERRORLEVEL% neq 0 (
    call :print_red "Python 依赖安装失败！"
    exit /b 1
)
call :print_green "Python 依赖安装成功！"

:: 检查配置文件是否存在
if not exist ".\public\claude2api_config.env" (
    call :print_red "错误: .\public\claude2api_config.env 文件不存在！"
    call :print_blue "请创建配置文件，内容示例:"
    echo ANTHROPIC_API_KEY=your_api_key_here
    exit /b 1
)

:: 启动所有服务
call :print_blue "并行启动所有服务..."

:: 启动 Node.js 服务
call :print_blue "启动 Node.js 服务..."
start /b cmd /c "npm run start"
call :print_green "Node.js 服务已启动"

:: 启动 Python 服务
call :print_blue "启动 Python 服务..."
start /b cmd /c "python .\public\app.py"
call :print_green "Python 服务已启动"

:: 启动 claude2api 服务
call :print_blue "启动 claude2api 服务..."
:: 读取环境变量文件
for /f "tokens=*" %%a in (.\public\claude2api_config.env) do set "%%a"
start /b cmd /c "cd .\public\claude2api && .\claude2api.exe"
call :print_green "claude2api 服务已启动"

call :print_green "所有服务已并行启动！"
call :print_blue "按 Ctrl+C 并输入 Y 停止所有服务..."

:: 等待用户按下任意键退出
pause > nul
call :print_blue "正在关闭所有服务..."

:: 结束所有启动的进程
taskkill /f /im node.exe 2>nul
taskkill /f /im python.exe 2>nul
taskkill /f /im claude2api.exe 2>nul

exit /b 0

:: 函数定义
:print_green
echo [%GREEN%m%~1[0m
exit /b 0

:print_blue
echo [%BLUE%m%~1[0m
exit /b 0

:print_red
echo [%RED%m%~1[0m
exit /b 0

:check_command
where %~1 >nul 2>nul
if %ERRORLEVEL% neq 0 (
    call :print_red "错误: %~1 未安装。请先安装 %~1。"
    exit /b 1
)
exit /b 0
