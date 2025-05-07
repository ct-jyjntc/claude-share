#!/bin/bash

# 显示彩色输出的函数
print_green() {
    echo -e "\033[0;32m$1\033[0m"
}

print_blue() {
    echo -e "\033[0;34m$1\033[0m"
}

print_red() {
    echo -e "\033[0;31m$1\033[0m"
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_red "错误: $1 未安装。请先安装 $1。"
        exit 1
    fi
}

# 检查必要的命令是否存在
print_blue "检查必要的命令..."
check_command node
check_command npm
check_command python3
check_command pip3

# 安装 Node.js 依赖
print_blue "安装 Node.js 依赖..."
npm install
if [ $? -ne 0 ]; then
    print_red "Node.js 依赖安装失败！"
    exit 1
fi
print_green "Node.js 依赖安装成功！"

# 安装 Python 依赖
print_blue "安装 Python 依赖..."
# 检查 requirements.txt 是否存在
if [ -f "requirements.txt" ]; then
    pip3 install -r requirements.txt
elif [ -f "./public/requirements.txt" ]; then
    pip3 install -r ./public/requirements.txt
else
    # 如果没有 requirements.txt，安装常见的依赖
    print_blue "未找到 requirements.txt，安装常见的 Python 依赖..."
    pip3 install flask requests websockets
fi

if [ $? -ne 0 ]; then
    print_red "Python 依赖安装失败！"
    exit 1
fi
print_green "Python 依赖安装成功！"

# 检查配置文件是否存在
if [ ! -f "./public/claude2api_config.env" ]; then
    print_red "错误: ./public/claude2api_config.env 文件不存在！"
    print_blue "请创建配置文件，内容示例:"
    echo "ANTHROPIC_API_KEY=your_api_key_here"
    exit 1
fi

# 定义一个函数来启动服务
start_services() {
    # 启动所有服务
    print_blue "并行启动所有服务..."

    # 启动 Node.js 服务
    print_blue "启动 Node.js 服务..."
    npm run start &
    NODE_PID=$!
    print_green "Node.js 服务已启动，PID: $NODE_PID"

    # 启动 Python 服务
    print_blue "启动 Python 服务..."
    python3 ./public/app.py &
    PYTHON_PID=$!
    print_green "Python 服务已启动，PID: $PYTHON_PID"

    # 启动 claude2api 服务
    print_blue "启动 claude2api 服务..."
    (cd ./public/claude2api && env $(cat ../claude2api_config.env) ./claude2api) &
    CLAUDE_API_PID=$!
    print_green "claude2api 服务已启动，PID: $CLAUDE_API_PID"

    # 保存PID到全局变量
    export NODE_PID PYTHON_PID CLAUDE_API_PID
}

# 启动所有服务
start_services

print_green "所有服务已并行启动！"
print_blue "按 Ctrl+C 停止所有服务..."

# 捕获 Ctrl+C 信号，优雅地关闭所有服务
trap 'kill $NODE_PID $PYTHON_PID $CLAUDE_API_PID 2>/dev/null; print_blue "正在关闭所有服务..."; exit 0' INT

# 等待任意子进程退出
wait

# 如果有任何进程退出，杀死所有其他进程
print_red "一个服务已退出，正在关闭所有其他服务..."
kill $NODE_PID $PYTHON_PID $CLAUDE_API_PID 2>/dev/null
