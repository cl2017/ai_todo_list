# MCP Todo 服务器

这是一个基于MCP (Model Context Protocol) 协议的智能待办事项管理服务器，使用HTTP API而不是WebSocket协议。

## 功能特性

### 🎯 核心功能
- **内存数据库存储**: 使用内存存储，避免CGO依赖问题
- **HTTP API**: 完全基于HTTP协议的MCP服务器实现
- **智能分析**: AI驱动的任务分析和日程优化
- **数据导入**: 从data.json自动导入初始数据

### 🔧 MCP工具
- `list_todos`: 列出所有待办事项，支持过滤
- `create_todo`: 创建新的待办事项
- `update_todo`: 更新现有待办事项
- `delete_todo`: 删除待办事项
- `analyze_tasks`: 智能分析任务状态
- `optimize_schedule`: 优化工作日程
- `break_down_task`: 将复杂任务分解为子任务

## 技术栈

- **语言**: Go 1.22.2
- **数据库**: 内存存储 (避免CGO依赖)
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
# 方法1: 直接运行
go run .

# 方法2: 使用启动脚本
start.bat
```

服务器将在 `http://localhost:8081` 启动

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
- `POST /mcp/initialize` - 初始化MCP连接
- `GET /mcp/tools/list` - 获取可用工具列表
- `POST /mcp/tools/call` - 调用MCP工具

## MCP工具调用示例

### 创建待办事项
```bash
curl -X POST http://localhost:8081/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "create_todo",
    "method": "tools/call",
    "params": {
      "name": "create_todo",
      "arguments": {
        "title": "完成项目文档",
        "description": "编写项目技术文档",
        "priority": "high",
        "category": "work",
        "estimated_duration": "2小时"
      }
    }
  }'
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

### 内存数据库结构
- **todos**: 待办事项列表
- **userProfile**: 用户配置信息
- **线程安全**: 使用读写锁保证并发安全

### 数据流程
1. 启动时从data.json导入初始数据到内存
2. 所有CRUD操作通过内存数据库进行
3. MCP工具调用通过HTTP API处理
4. 数据在程序运行期间保持在内存中

## 项目结构

```
fydeos/
├── main.go              # 主程序入口
├── database_simple.go   # 内存数据库实现
├── mcp_server.go        # MCP服务器实现
├── database.go          # SQLite数据库实现(备用)
├── data.json            # 初始数据
├── start.bat            # 启动脚本
├── go.mod               # Go模块文件
└── README.md            # 项目说明
```

## 开发说明

### 主要改进
1. **移除WebSocket**: 完全改为HTTP API实现
2. **内存数据库**: 使用内存存储避免CGO依赖问题
3. **数据持久化**: 所有操作都在内存中进行
4. **MCP协议**: 实现标准的MCP协议工具

### 解决CGO问题
由于SQLite驱动需要CGO支持，我们提供了两种解决方案：
1. **内存数据库**: 默认使用，无需CGO
2. **SQLite数据库**: 需要启用CGO (`CGO_ENABLED=1`)

### 数据流程
1. 启动时从data.json导入初始数据到内存
2. 所有CRUD操作通过内存数据库进行
3. MCP工具调用通过HTTP API处理
4. 数据在程序运行期间保持在内存中

## 许可证

MIT License 