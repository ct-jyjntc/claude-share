# Claude2 API 使用文档

## 准备工作

### 1. 下载并解压文件

首先，您需要下载并解压 Claude2 API 的安装包：

```bash
# 下载文件
wget https://github.com/ct-jyjntc/claude-share/releases/download/claude2api/claude2api.zip

# 解压文件
unzip claude2api.zip
```

或者您也可以直接通过浏览器访问链接下载：
https://github.com/ct-jyjntc/claude-share/releases/download/claude2api/claude2api.zip

### 2. 环境要求

在开始之前，请确保您的系统已安装以下软件：

- **Node.js**：JavaScript 运行环境
- **pip3**：Python 包管理工具

可以通过以下命令检查是否已安装：

```bash
# 检查 Node.js 是否安装
node -v

# 检查 pip3 是否安装
pip3 -v
```

如果未安装，请根据您的操作系统安装这些工具。

## 安装步骤

### 1. 替换可执行文件

根据您的操作系统，从仓库中下载对应的可执行文件，并替换解压后的 claude2api 文件夹中的原始 claude2api 文件：

- 对于 Linux 系统：下载 Linux 版本的可执行文件
- 对于 macOS 系统：下载 macOS 版本的可执行文件
- 对于 Windows 系统：下载 Windows 版本的可执行文件

```bash
# 确保新下载的可执行文件具有执行权限
chmod +x /path/to/new/claude2api

# 替换原文件
mv /path/to/new/claude2api /path/to/claude2api/folder/claude2api
```

### 2. 启动服务

完成上述步骤后，您可以通过运行启动脚本来启动 Claude2 API 服务：

```bash
# 进入解压后的目录
cd claude2api

# 启动服务
./start.sh
```

## 使用说明

启动服务后，您可以通过 API 端点与 Claude2 进行交互。默认情况下，服务会在本地的 3000 端口启动。

### API 端点

- **基础 URL**: `http://localhost:3000`
- **聊天接口**: `POST /v1/chat/completions`

### 示例请求

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-2",
    "messages": [
      {"role": "user", "content": "你好，请介绍一下自己。"}
    ],
    "max_tokens": 1000
  }'
```

## 常见问题

1. **服务无法启动**
   - 检查 Node.js 和 pip3 是否正确安装
   - 确保可执行文件具有执行权限
   - 查看日志文件了解详细错误信息

2. **API 请求失败**
   - 确认服务是否正在运行
   - 检查请求格式是否正确
   - 验证网络连接是否正常

3. **性能问题**
   - 考虑增加服务器资源
   - 优化请求频率和大小

## 注意：如果使用反向代理，请修改src文件夹下的App.jsx文件的47行为对应的地址

## 支持与反馈

如果您在使用过程中遇到任何问题，请通过以下方式获取支持：

- 在 GitHub 仓库提交 Issue
- 联系技术支持团队
