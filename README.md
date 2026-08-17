# up-down

`up-down` 是一个 Python 3 文件上传服务与命令行客户端。服务端接收 `POST /upload`，文件通过 `GET /files/<filename>` 访问，健康检查为 `GET /healthz`。

## 配置原则

服务端和客户端使用不同的 `.env` 文件：

| 运行位置 | 配置文件 | 必填变量 | 用途 |
| --- | --- | --- | --- |
| 上传服务所在机器 | 项目根目录 `.env` | `LISTEN_ADDR`、`UPLOAD_DIR`、`MIN_FREE_BYTES`、`MAX_UPLOAD_BYTES` | 启动和管理上传服务 |
| 使用 `up` 的机器 | 可执行文件同目录 `.env` | `SERVER_URL` | 告诉客户端上传到哪个服务端 |

不要把客户端的 `SERVER_URL` 配置当成服务端配置，也不要依赖服务端 `.env` 被复制到客户端。`SERVER_URL` 推荐填写可长期使用的域名；它必须是完整的 `http://` 或 `https://` URL，且不应以 `/` 结尾。

## 服务端 `.env`

在部署服务端的项目根目录创建配置：

```sh
cp .env.server.example .env
```

编辑 `.env`：

```dotenv
# 监听地址。:8080 表示监听所有网络接口的 8080 端口。
LISTEN_ADDR=:8080

# 上传文件所在目录。生产环境建议使用绝对路径。
UPLOAD_DIR=/var/lib/up-down/uploads

# 低于该可用空间时，删除最旧上传文件。
MIN_FREE_BYTES=1073741824

# 单个 multipart 上传请求允许的最大字节数。
MAX_UPLOAD_BYTES=10737418240
```

启动服务：

```sh
python3 cmd/up-server.py
```

后台运行和管理：

```sh
chmod +x scripts/up-server.sh
./scripts/up-server.sh start
./scripts/up-server.sh status
./scripts/up-server.sh restart
./scripts/up-server.sh stop
```

脚本会先检查 `LISTEN_ADDR` 是否可用，并将 PID 和日志写入 `run/`。服务启动后可验证：

```sh
curl -i http://localhost:8080/healthz
```

## 客户端 `.env`

构建客户端：

```sh
make build-client
```

构建会将项目根目录 `.env` 中的 `SERVER_URL` 编译进 `dist/up`。也可在构建时显式指定：

```sh
make build-client SERVER_URL=https://upload.example.com
```

上传文件：

```sh
./dist/up ./example.png
```

成功后，客户端会打印文件的公开访问 URL。

## 客户端地址优先级

客户端按以下优先级确定服务端地址：

1. `--server` 命令行参数。
2. 环境变量 `SERVER_URL`。
3. 当前工作目录的 `.env`。
4. `up` 可执行文件同目录的 `.env`。
5. 构建时嵌入的 `SERVER_URL`。

示例：

```sh
./dist/up --server https://upload.example.com ./example.png
SERVER_URL=https://upload.example.com ./dist/up ./example.png
```

从源码运行时，当前目录的 `.env` 同样生效：

```sh
SERVER_URL=https://upload.example.com python3 cmd/up.py ./example.png
```

## 安装客户端

将源码客户端安装为命令启动器：

```sh
make install
up ./example.png
```

默认安装目录为 `~/.local/bin`。可使用 `BIN_DIR=/your/bin` 指定安装目录，使用 `COMMAND=upload` 指定命令名。安装命令会复制已编译的客户端，并内嵌构建时的服务端地址。
