# MCP Todo 服务器

这是一个基于MCP (Model Context Protocol) 协议的智能待办事项管理服务器，使用HTTP API实现。

## 功能特性

### 🎯 核心功能
- **SQLite数据库存储**: 使用SQLite3进行数据持久化
- **HTTP API**: 完全基于HTTP协议的API实现
- **智能分析**: AI驱动的任务分析和日程优化
- **数据导入**: 从data.json自动导入初始数据

### 🔧 MCP工具
- `list_todos`: 列出所有待办事项，支持过滤
- `create_todo`: 创建新的待办事项
- `update_todo`: 更新现有待办事项
- `delete_todo`: 删除待办事项
- `analyze_tasks`: 智能分析任务状态
- `optimize_schedule`: 优化工作日程

## 技术栈

- **语言**: Go 1.23
- **数据库**: SQLite3 (存储在当前目录)
- **Web框架**: Gorilla Mux
- **协议**: HTTP REST API
- **数据格式**: JSON

## 安装和运行

### 1. 安装依赖
```bash
go mod tidy
```

### 2. 运行服务器
```bash
# 直接运行
go run .
```

服务器将在 `http://localhost:8081` 启动，MCP SSE服务器将在 `http://localhost:8082` 启动

## API端点

### 基础API
- `GET /api/todos` - 获取所有待办事项
- `POST /api/todos` - 创建新待办事项
- `PUT /api/todos/{id}` - 更新待办事项
- `DELETE /api/todos/{id}` - 删除待办事项
- `GET /api/profile` - 获取用户配置

### AI分析API
- `GET /api/ai/analyze` - 智能分析任务
- `GET /api/ai/optimize` - 优化工作日程

### MCP API
- `GET /sse` - SSE（Server-Sent Events）连接端点
- `POST /message` - 发送消息到MCP服务器

## MCP工具调用示例

### 连接到SSE服务器
```javascript
// 前端JavaScript示例
const eventSource = new EventSource('http://localhost:8082/sse');

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('接收到消息:', data);
};

eventSource.onerror = (error) => {
  console.error('SSE连接错误:', error);
  eventSource.close();
};
```

### 发送工具调用请求
```javascript
// 创建待办事项示例
fetch('http://localhost:8082/message', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    type: 'call_tool',
    content: {
      name: 'create_todo',
      arguments: {
        title: '完成项目文档',
        description: '编写项目技术文档',
        priority: 'high',
        category: 'work',
        estimated_duration: '2小时'
      }
    }
  })
});
```

### 分析任务
```bash
curl -X POST http://localhost:8081/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "analyze_tasks",
    "method": "tools/call",
    "params": {
      "name": "analyze_tasks",
      "arguments": {
        "analysis_type": "priority"
      }
    }
  }'
```

### 优化日程
```bash
curl -X POST http://localhost:8081/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "optimize_schedule",
    "method": "tools/call",
    "params": {
      "name": "optimize_schedule",
      "arguments": {
        "time_horizon": "today",
        "work_hours": 8
      }
    }
  }'
```

## 数据存储

### SQLite数据库结构
- **todos表**: 存储待办事项列表
- **user_profile表**: 存储用户配置信息
- **持久化**: 数据存储在当前目录的todos.db文件中

### 数据流程
1. 启动时初始化SQLite数据库结构
2. 所有CRUD操作通过SQLite数据库进行
3. MCP工具调用通过SSE服务器处理
4. 数据持久化保存在todos.db文件中

## 项目结构

```
fydeos/
├── api/                # API处理函数
│   └── api.go           # API端点实现
├── db/                 # 数据库相关
│   └── sqlite.go        # SQLite数据库实现
├── mcp/                # MCP相关
│   └── mcp_server.go    # MCP服务器实现
├── static/             # 静态资源目录
├── main.go             # 主程序入口
├── data.json           # 初始数据
├── todos.db            # SQLite数据库文件
├── go.mod              # Go模块文件
├── go.sum              # Go模块依赖校验
└── README.md           # 项目说明
```

## 开发说明

### 主要组件
1. **SQLite数据库**: 使用SQLite3实现数据持久化存储
2. **REST API**: 基本的CRUD操作通过HTTP REST API实现
3. **MCP SSE服务器**: 使用Server-Sent Events实现MCP协议
4. **AI分析功能**: 提供智能任务分析和日程优化功能

### 关于CGO
本项目使用SQLite数据库，需要启用CGO支持：
1. 在Windows系统上，需要安装GCC编译器（例如通过MinGW或MSYS2）
2. 在编译时需要设置环境变量 `CGO_ENABLED=1`
3. 如果遇到CGO相关问题，请参考go-sqlite3文档：https://pkg.go.dev/github.com/mattn/go-sqlite3

### 技术依赖
- **github.com/gorilla/mux**: HTTP路由处理
- **github.com/mark3labs/mcp-go**: MCP协议Go实现
- **github.com/mattn/go-sqlite3**: SQLite3数据库驱动
- **github.com/rs/cors**: 跨域资源共享支持

## 许可证

MIT License 