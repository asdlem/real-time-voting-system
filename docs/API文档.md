# 实时投票系统 API 文档 V3

本文档详细介绍实时投票系统的所有 API 接口，供前端开发人员使用。

## 基本信息

- 基础URL: `http://localhost:8090/api`
- 所有请求和响应均使用 JSON 格式
- 时间格式使用 ISO 8601 标准: `YYYY-MM-DDThh:mm:ssZ`
- 鉴权方式: 暂无，当前版本不需要认证，部分管理接口需要管理员密钥

## 目录

- [创建投票](#创建投票)
- [获取投票列表](#获取投票列表)
- [获取投票详情](#获取投票详情)
- [提交投票](#提交投票)
- [获取投票统计](#获取投票统计)
- [WebSocket连接](#websocket连接)
- [SSE连接（备用方案）](#sse连接备用方案)
- [管理接口](#管理接口)

## 接口详情

### 创建投票

创建一个新的投票活动。

- **URL**: `/polls`
- **方法**: `POST`
- **请求体**:

```json
{
  "title": "最喜欢的编程语言",
  "description": "选择你最喜欢的编程语言",
  "poll_type": 0,
  "options": ["Go", "Java", "Python", "JavaScript", "C++"],
  "end_time": "2023-07-30T16:00:00Z"
}
```

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| title | string | 是 | 投票标题 |
| description | string | 否 | 投票描述 |
| poll_type | integer | 是 | 投票类型：0=单选，1=多选 |
| options | string[] | 是 | 投票选项列表 |
| end_time | string | 是 | 投票结束时间（ISO 8601格式） |

- **成功响应** (状态码: 201):

```json
{
  "id": 123,
  "title": "最喜欢的编程语言",
  "description": "选择你最喜欢的编程语言",
  "poll_type": 0,
  "options": [
    {"id": 1, "text": "Go", "votes": 0},
    {"id": 2, "text": "Java", "votes": 0},
    {"id": 3, "text": "Python", "votes": 0},
    {"id": 4, "text": "JavaScript", "votes": 0},
    {"id": 5, "text": "C++", "votes": 0}
  ],
  "is_active": true,
  "end_time": "2023-07-30T16:00:00Z",
  "created_at": "2023-07-01T10:30:00Z"
}
```

- **错误响应** (状态码: 400):

```json
{
  "error": "请求参数无效",
  "details": ["标题不能为空", "必须提供至少两个选项"]
}
```

### 获取投票列表

获取分页的投票列表。

- **URL**: `/polls`
- **方法**: `GET`
- **查询参数**:

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| limit | int | 否 | 每页数量，默认10，最大50 |

- **成功响应** (状态码: 200):

```json
{
  "polls": [
    {
      "id": 123,
      "title": "最喜欢的编程语言",
      "description": "选择你最喜欢的编程语言",
      "poll_type": 0,
      "is_active": true,
      "end_time": "2023-07-30T16:00:00Z",
      "created_at": "2023-07-01T10:30:00Z",
      "total_votes": 42
    },
    // ...其他投票
  ],
  "pagination": {
    "current_page": 1,
    "total_pages": 5,
    "total_items": 42,
    "limit": 10
  }
}
```

- **错误响应** (状态码: 500):

```json
{
  "error": "服务器内部错误",
  "details": "数据库查询失败"
}
```

### 获取投票详情

获取特定投票的详细信息，包括所有选项。

- **URL**: `/polls/{id}`
- **方法**: `GET`
- **路径参数**:

| 参数 | 类型 | 描述 |
|------|------|------|
| id | int | 投票ID |

- **成功响应** (状态码: 200):

```json
{
  "id": 123,
  "title": "最喜欢的编程语言",
  "description": "选择你最喜欢的编程语言",
  "poll_type": 0,
  "options": [
    {"id": 1, "text": "Go", "votes": 15},
    {"id": 2, "text": "Java", "votes": 10},
    {"id": 3, "text": "Python", "votes": 8},
    {"id": 4, "text": "JavaScript", "votes": 7},
    {"id": 5, "text": "C++", "votes": 2}
  ],
  "is_active": true,
  "end_time": "2023-07-30T16:00:00Z",
  "created_at": "2023-07-01T10:30:00Z",
  "total_votes": 42
}
```

- **错误响应** (状态码: 404):

```json
{
  "error": "投票未找到",
  "details": "ID为123的投票不存在"
}
```

### 提交投票

向指定投票提交一个选项投票。

- **URL**: `/polls/{poll_id}/vote`
- **方法**: `POST`
- **路径参数**:

| 参数 | 类型 | 描述 |
|------|------|------|
| poll_id | int | 投票ID |

- **请求体 (单选)**:

```json
{
  "option_id": 3
}
```

- **请求体 (多选)**:

```json
{
  "option_ids": [1, 3, 5]
}
```

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| option_id | int | 单选必填 | 选择的选项ID |
| option_ids | int[] | 多选必填 | 选择的多个选项ID |

- **成功响应** (状态码: 200):

```json
{
  "success": true,
  "message": "投票成功",
  "updated_options": [
    {"id": 3, "votes": 9}
  ]
}
```

- **错误响应** (状态码: 400, 403, 409):

```json
{
  "error": "请求无效",
  "details": "必须提供有效的选项ID"
}
```

```json
{
  "error": "投票已结束",
  "details": "该投票已于2023-07-30 16:00:00结束"
}
```

```json
{
  "error": "重复投票",
  "details": "您已经参与过此投票"
}
```

### 获取投票统计

获取指定投票的实时统计数据。

- **URL**: `/polls/{poll_id}/stats`
- **方法**: `GET`
- **路径参数**:

| 参数 | 类型 | 描述 |
|------|------|------|
| poll_id | int | 投票ID |

- **成功响应** (状态码: 200):

```json
{
  "poll_id": 123,
  "total_votes": 42,
  "options": [
    {"id": 1, "text": "Go", "votes": 15, "percentage": 35.71},
    {"id": 2, "text": "Java", "votes": 10, "percentage": 23.81},
    {"id": 3, "text": "Python", "votes": 8, "percentage": 19.05},
    {"id": 4, "text": "JavaScript", "votes": 7, "percentage": 16.67},
    {"id": 5, "text": "C++", "votes": 2, "percentage": 4.76}
  ],
  "last_updated": "2023-07-15T14:30:45Z"
}
```

- **错误响应** (状态码: 404):

```json
{
  "error": "投票未找到",
  "details": "ID为123的投票不存在"
}
```

### WebSocket连接

建立WebSocket连接以接收实时投票更新（推荐使用）。

- **WebSocket URL**: `/api/polls/{poll_id}/ws`
- **路径参数**:

| 参数 | 类型 | 描述 |
|------|------|------|
| poll_id | int | 投票ID |

- **消息格式**:

1. **初始数据消息** - 连接建立后服务器发送的初始数据

```json
{
  "type": "initial",
  "timestamp": "2023-07-15T14:32:45Z",
  "data": {
    "poll_id": 123,
    "title": "最喜欢的编程语言",
    "options": [
      {"id": 1, "text": "Go", "votes": 15},
      {"id": 2, "text": "Java", "votes": 10},
      {"id": 3, "text": "Python", "votes": 8},
      {"id": 4, "text": "JavaScript", "votes": 7},
      {"id": 5, "text": "C++", "votes": 2}
    ],
    "total_votes": 42
  }
}
```

2. **更新消息** - 有新投票时服务器发送的更新

```json
{
  "type": "update",
  "timestamp": "2023-07-15T14:33:12Z",
  "data": {
    "poll_id": 123,
    "updated_options": [
      {"id": 1, "votes": 16}
    ],
    "total_votes": 43
  }
}
```

3. **心跳消息** - 保持连接活跃的心跳

```json
{
  "type": "heartbeat",
  "timestamp": "2023-07-15T14:34:00Z"
}
```

### SSE连接（备用方案）

作为备用的实时更新方案，系统也支持SSE连接。

- **SSE URL**: `/api/polls/{poll_id}/live`
- **路径参数**:

| 参数 | 类型 | 描述 |
|------|------|------|
| poll_id | int | 投票ID |

- **消息格式**与WebSocket类似，但通过HTTP流式传输。

## 管理接口

### 重置投票数据

重置指定投票的所有票数为0。

- **URL**: `/api/admin/polls/{poll_id}/reset`
- **方法**: `POST`
- **路径参数**:

| 参数 | 类型 | 描述 |
|------|------|------|
| poll_id | int | 投票ID |

- **请求体**:

```json
{
  "admin_key": "admin123"
}
```

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| admin_key | string | 是 | 管理员密钥 |

- **成功响应** (状态码: 200):

```json
{
  "success": true,
  "message": "投票数据已重置"
}
```

- **错误响应** (状态码: 400, 401, 404):

```json
{
  "error": "管理员密钥错误"
}
```

```json
{
  "error": "投票不存在"
}
```

### 清理Redis缓存

清理Redis中的缓存数据。

- **URL**: `/api/admin/cache/clean`
- **方法**: `POST`
- **请求体**:

```json
{
  "admin_key": "admin123",
  "patterns": ["poll:*", "vote_lock:*"]
}
```

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| admin_key | string | 是 | 管理员密钥 |
| patterns | array | 是 | 要清理的缓存键模式 |

- **成功响应** (状态码: 200):

```json
{
  "success": true,
  "message": "缓存已清理",
  "deleted_keys_count": 15
}
```

- **错误响应** (状态码: 400, 401):

```json
{
  "error": "管理员密钥错误"
}
```

```json
{
  "error": "清理缓存失败: Redis连接错误"
}
```

## 高并发测试结果

系统在高并发场景下表现优异，通过高并发测试得到以下结果：

### 高并发处理能力
- 每秒可稳定处理近100个请求（RPS: 94.43）
- 平均响应时间：0.1519秒（并发测试）/ 0.0146秒（持续测试）
- 95%响应时间：0.2811秒（并发测试）/ 0.0216秒（持续测试）

### 限流与防重复功能
- 同一IP限流成功率：99%
- 防重复投票成功率：100%

### 数据一致性保证
- 所有测试场景下，预期票数与数据库记录完全一致
- 分布式锁和事务处理确保了数据的完整性和准确性

## 客户端集成指南

### JavaScript集成示例

```javascript
// 提交投票
async function submitVote(pollId, optionId) {
  try {
    const response = await fetch(`http://localhost:8090/api/polls/${pollId}/vote`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ option_id: optionId })
    });
    
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || '投票失败');
    }
    
    return data;
  } catch (error) {
    console.error('投票错误:', error);
    throw error;
  }
}

// WebSocket连接示例
function connectToWebSocket(pollId) {
  const ws = new WebSocket(`ws://localhost:8090/api/polls/${pollId}/ws`);
  
  ws.onopen = () => {
    console.log('WebSocket连接已建立');
  };
  
  ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    
    if (message.type === 'initial') {
      // 处理初始数据
      initializePollData(message.data);
    } else if (message.type === 'update') {
      // 处理更新
      updatePollData(message.data);
    } else if (message.type === 'heartbeat') {
      // 响应心跳
      ws.send(JSON.stringify({ type: 'heartbeat_ack' }));
    }
  };
  
  ws.onclose = (event) => {
    console.log(`WebSocket连接已关闭：${event.code}`);
    // 重连逻辑
    setTimeout(() => connectToWebSocket(pollId), 3000);
  };
  
  ws.onerror = (error) => {
    console.error('WebSocket错误:', error);
  };
  
  return ws;
}
```

## 状态码说明

| 状态码 | 说明 |
|-------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求错误 |
| 403 | 禁止访问 |
| 404 | 资源不存在 |
| 429 | 请求频率过高 |
| 500 | 服务器内部错误 |

## 错误处理

所有错误响应格式统一为：

```json
{
  "error": "错误描述信息"
}
```

## 注意事项

1. 同一用户（按IP或会话ID识别）对同一投票只能投票一次
2. 投票必须在开始时间和结束时间之间，且状态为"active"才能参与
3. WebSocket连接可能因网络问题断开，客户端应实现自动重连逻辑
4. API请求可能会受到限流保护，请合理控制请求频率

## 扩展信息

### 投票状态

| 状态 | 说明 |
|------|------|
| draft | 草稿，尚未开始 |
| active | 进行中，可以投票 |
| paused | 已暂停，临时不能投票 |
| completed | 已结束，不能再投票 |

#### 系统接口
- `GET /api/health` - 健康检查接口
- `GET /api/status` - 系统状态接口
- `GET /api/metrics` - 系统性能指标接口（以日志格式）
- `POST /api/admin/polls/:id/reset` - 重置投票数据（管理接口）
- `POST /api/admin/cache/clean` - 清理缓存（管理接口） 