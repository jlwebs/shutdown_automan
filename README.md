# Remote Restart Service (远程重启服务)

这是一个轻量级的 Windows 系统托盘程序，用于远程触发系统重启，并支持自定义重启前的预处理逻辑（如终止指定进程）。

## 功能特性

- **HTTP 服务端**：监听指定端口（默认 8080），提供 `/restart` 接口触发重启。
- **自定义重启流程**：
  1. 依次终止配置的进程列表。
  2. 每个进程终止后等待指定延迟时间。
  3. 执行系统重启 (`shutdown /r /t 0`)。
- **自动监控**：可配置定期检查进程是否存在，若缺失则自动触发重启流程。
- **系统托盘**：程序运行在后台，通过托盘图标进行管理（启动/停止服务、设置）。
- **配置持久化**：所有配置保存于 `config.json`。

## 构建说明 (Build Instructions)

本项目专为 Windows 设计，使用了 `walk` GUI 库。

### 前置要求

1.  安装 Go (1.16+).
2.  仅支持 Windows 系统进行构建和运行（或者在 Linux/Mac 上使用交叉编译，但 `walk` 库包含 CGO/WinAPI 调用，建议在 Windows 上构建）。

### 构建步骤

#### 方式一：使用构建脚本 (推荐 Mac/Linux 用户)

本项目提供了自动化的构建脚本 `build_windows.sh`，适用于 macOS 和 Linux 环境交叉编译。

1.  确保已安装 Go (1.16+)。
2.  在终端运行以下命令：

```bash
chmod +x build_windows.sh
./build_windows.sh
```

脚本会自动下载依赖、编译 Windows 可执行文件，并生成 `release` 目录，其中包含 `RemoteRestartService.exe` 和配置文件。

#### 方式二：手动构建 (Windows 用户)

在 Windows 环境下，打开终端（PowerShell 或 CMD）：

```bash
# 1. 下载依赖
go mod tidy

# 2. 生成资源文件 (可选，如果需要嵌入 manifest 和图标)
# 需要安装 rsrc: go install github.com/akavel/rsrc@latest
# rsrc -manifest app.manifest -o rsrc.syso

# 3. 编译
# -ldflags="-H=windowsgui" 用于隐藏控制台窗口
go build -ldflags="-H=windowsgui" -o "RemoteRestartService.exe"
```

如果未嵌入 manifest，请确保 `app.manifest` 文件与 `.exe` 文件在同一目录下，以获取管理员权限。

## 使用说明

1.  **运行**：以**管理员身份**运行 `RemoteRestartService.exe`。
2.  **托盘图标**：右下角会出现应用图标。
3.  **启动服务**：右键点击图标 -> "Start Service"。
4.  **触发重启**：
    -   **HTTP**: 发送 POST 请求到 `http://localhost:8080/restart`。
    -   **监控**: 若启用了监控，当指定进程消失时会自动触发。
5.  **配置**：
    -   右键点击图标 -> "Settings" 修改端口和监控设置。
    -   **进程列表**：目前请直接编辑生成的 `config.json` 文件来配置进程列表。格式如下：

    ```json
    {
      "port": "8080",
      "process_list": [
        {
          "name": "notepad.exe",
          "delay": 5
        },
        {
          "name": "calc.exe",
          "delay": 2
        }
      ],
      "monitor_enabled": false,
      "monitor_interval": 60
    }
    ```

## 注意事项

-   程序必须以管理员权限运行才能终止进程和重启系统。
-   HTTP 服务默认监听所有接口（`:` + 端口），请注意防火墙设置。
-   监控功能在服务启动后才生效。
