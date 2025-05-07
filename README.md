# Docker 部署指南

本项目由两个组件组成：
1. Web前端（React + Node.js）
2. Go API后端（claude2api）

两个组件都使用Docker容器化，可以通过Docker Compose一起部署。

## 前提条件

- Docker
- Docker Compose

## 快速开始

1. 克隆仓库
2. 进入项目目录
3. 运行以下命令：

```bash
git clone https://github.com/ct-jyjntc/claude-share.git

cd claude-share

cp public/data/sessionKeys.json.example public/data/sessionKeys.json

cp public/claude2api/.env.example public/claude2api/.env

docker-compose up -d
```

这将在后台启动前端和后端服务。

## 访问应用

- Web前端：http://你的服务器IP:5173
- 后端API：http://你的服务器IP:8080

## 配置

### 环境变量

您可以通过修改`.env`文件或在`docker-compose.yml`文件中设置环境变量来自定义部署。

### 会话密钥

会话密钥存储在`public/data/sessionKeys.json`中。这个文件在前端和后端容器之间共享。

## 停止应用

要停止应用，运行：

```bash
docker-compose down
```

## 重建应用

如果您对代码进行了更改，需要重新构建Docker镜像：

```bash
docker-compose build
docker-compose up -d
```

## 日志

查看容器日志：

```bash
# 查看所有容器的日志
docker-compose logs

# 查看特定容器的日志
docker-compose logs frontend
docker-compose logs backend

# 实时跟踪日志
docker-compose logs -f
```

## 故障排除

### 端口冲突

如果遇到端口冲突，可以在`docker-compose.yml`文件中更改端口映射。

### 卷权限

如果遇到共享卷的权限问题，请确保`public/data`目录具有正确的权限。

### 容器通信

前端和后端容器通过`claude-network` Docker网络进行通信。确保正确创建此网络。

### 文件共享问题

如果后端无法读取`sessionKeys.json`文件，请检查：

1. 文件是否存在于`public/data`目录中
2. Docker卷是否正确挂载
3. 文件权限是否正确

在我们的配置中，前端容器将`./public/data`挂载到`/app/public/data`，后端容器将`./public/data`挂载到`/app/data`。这样两个容器都可以访问同一个文件。
