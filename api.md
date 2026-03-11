# Shutdown Automan - 前端对接 API 文档

本文档描述了 Shutdown Automan 服务端提供的 HTTP 接口。所有接口所在的服务器端口以系统配置文件为主（默认通常为 8080）。

## 鉴权说明 (Authentication)

如果后端的配置文件中设置了 `SecretKey`，那么在每次请求时，必须在 URL 中附加 `key` 作为 query 参数，以便后端进行权限验证。
- **示例**: `http://localhost:8080/process_status?key=你的密钥`
- **如果密钥不匹配**：所有受保护的接口均会返回 HTTP 状态码 `403 Forbidden` 和报错 `"Invalid Secret Key"`。

---

## 1. 进程状态查询接口

用于获取当前配置的进程信息及其运行状态。

- **URL**: `/process_status`
- **Method**: `GET`
- **Query 参数**:
  - `key` (可选): 用于鉴权的密钥（视服务端配置而定）。

### 成功响应示例 (200 OK)

响应 Content-Type 为 `application/json`。

```json
[
  {
    "name": "explorer.exe",
    "status": "Running",
    "delay": 10
  },
  {
    "name": "notepad.exe",
    "status": "Not Started",
    "delay": 5
  }
]
```

#### 字段说明
| 字段名称 | 类型 | 描述 |
| ------ | ------ | ------ |
| `name` | string | 进程名称（在配置列表中定义的名称） |
| `status` | string | 当前状态。通常为 `"Running"` (运行中) 或 `"Not Started"` (未运行/未知)。 |
| `delay` | int | 后端配置的该进程相关延迟时间 |

### 错误响应

- **403 Forbidden**: 提供或缺失的 `key` 不正确。
- **500 Internal Server Error**: 取回进程状态时系统发生错误（返回文本形式错误信息）。

---

## 2. 触发重启接口

由于后端设计兼容浏览器快速请求以及脚本调用，该接口既支持 `POST` 也支持 `GET` 方法。

- **URL**: `/restart`
- **Method**: `GET`, `POST`
- **Query 参数**:
  - `key` (可选): 用于鉴权的密钥（视服务端配置而定）。

### 成功响应示例 (200 OK)

响应 Content-Type 为 `text/plain`。

```text
Restart initiated
```
*(注意：该接口触发的是后端的异步操作，实际的进程结束和重启动作将在后端队列中执行)*

### 错误响应

- **403 Forbidden**: 提供或缺失的 `key` 不正确。
- **405 Method Not Allowed**: 使用了除 `GET` 或 `POST` 以外的请求方法。
