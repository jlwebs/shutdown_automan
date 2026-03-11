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

响应 Content-Type 为 `application/json`。返回的结构将进程列表包裹在对象内，并包含系统近十分钟的平均网速估计。

```json
{
  "processes": [
    {
      "name": "explorer.exe",
      "status": "Running",
      "delay": 10,
      "is_not_responding": false
    },
    {
      "name": "notepad.exe",
      "status": "Not Started",
      "delay": 5,
      "is_not_responding": false
    }
  ],
  "network_speed_in_bps": 10240.5,
  "network_speed_out_bps": 2048.0
}
```

#### 字段说明
| 字段名称 | 类型 | 描述 |
| ------ | ------ | ------ |
| `processes` | array | 被监控进程的数组 |
| `name` | string | 进程名称（在配置列表中定义的名称） |
| `status` | string | 当前状态。通常为 `"Running"` (运行中), `"Not Responding"` (无响应) 或 `"Not Started"` (未运行/未知)。 |
| `delay` | int | 后端配置的该进程相关延迟时间 |
| `is_not_responding` | bool | 标记进程是否处于“无响应”状态 |
| `network_speed_in_bps` | float | 系统级下行网络速度（字节/秒 Bps），反映近10分钟内的平均值，用于作为是否正在下载更新/活跃玩家交互的参考指标。 |
| `network_speed_out_bps` | float | 系统级上行网络速度（字节/秒 Bps）。 |

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
