package mcp

import (
	"context"
	"fmt"
	"fydeos/db"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func InitMCP() {
	s := server.NewMCPServer(
		"go-mcp-todo-list",
		"1.0.0",
		server.WithLogging(),
		server.WithRecovery(),
	)

	RegisterTodoTools(s, db.DB)

	if err := serveSSE(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func serveSSE(s *server.MCPServer) error {
	_, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := server.NewSSEServer(s)

	mux := http.NewServeMux()

	mux.Handle("/sse", srv)

	mux.Handle("/message", srv)

	httpServer := &http.Server{
		Addr:    "localhost:8082",
		Handler: mux,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	return nil
}

// 注册所有相关工具
func RegisterTodoTools(s *server.MCPServer, sqlite *db.SQLiteDatabase) {
	// list_todos
	s.AddTool(mcp.NewTool(
		"list_todos",
		mcp.WithDescription("列出所有待办事项，支持过滤"),
		// 无需参数定义
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		todo, _ := sqlite.GetAllTodos()
		return mcp.NewToolResultStructuredOnly(todo), nil
	})

	// create_todo
	s.AddTool(mcp.NewTool(
		"create_todo",
		mcp.WithDescription("创建新的待办事项"),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("标题"),
		),
		mcp.WithString("description",
			mcp.Description("描述"),
		),
		mcp.WithString("priority",
			mcp.Description("优先级（urgent/high/medium/low）"),
			mcp.Enum("urgent", "high", "medium", "low"),
		),
		mcp.WithString("category",
			mcp.Description("类别"),
		),
		mcp.WithString("estimated_duration",
			mcp.Description("预计耗时"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		todo := &db.Todo{
			Title:             req.GetString("title", ""),
			Description:       req.GetString("description", ""),
			Priority:          req.GetString("priority", ""),
			Category:          req.GetString("category", ""),
			Status:            "pending",
			CreatedDate:       time.Now(),
			LastUpdated:       time.Now(),
			EstimatedDuration: req.GetString("estimated_duration", ""),
		}
		if todo.Priority == "" {
			todo.Priority = "medium"
		}
		if todo.Category == "" {
			todo.Category = "personal"
		}

		if err := sqlite.CreateTodo(todo); err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(fmt.Sprintf("Created todo: %s (ID: %d)", todo.Title, todo.ID)), nil
	})

	// update_todo
	s.AddTool(mcp.NewTool(
		"update_todo",
		mcp.WithDescription("更新现有待办事项"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("待办事项ID"),
		),
		mcp.WithString("title",
			mcp.Description("标题"),
		),
		mcp.WithString("description",
			mcp.Description("描述"),
		),
		mcp.WithString("priority",
			mcp.Description("优先级"),
			mcp.Enum("urgent", "high", "medium", "low"),
		),
		mcp.WithString("status",
			mcp.Description("状态"),
			mcp.Enum("pending", "in_progress", "completed"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetFloat("id", 0)
		todo, err := sqlite.GetTodoByID(int(id))
		if err != nil {
			return nil, fmt.Errorf("todo with ID %d not found", id)
		}
		todo.Title = req.GetString("title", "")
		todo.Description = req.GetString("description", "")
		todo.Priority = req.GetString("priority", "")
		todo.Status = req.GetString("status", "")

		todo.LastUpdated = time.Now()
		if err := sqlite.UpdateTodo(todo); err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(fmt.Sprintf("Updated todo: %s (ID: %d)", todo.Title, todo.ID)), nil
	})

	// delete_todo
	s.AddTool(mcp.NewTool(
		"delete_todo",
		mcp.WithDescription("删除待办事项"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("待办事项ID"),
		)), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idFloat := req.GetFloat("id", 0)
		id := int(idFloat)
		todo, err := sqlite.GetTodoByID(id)
		if err != nil {
			return nil, fmt.Errorf("todo with ID %d not found", id)
		}
		if err := sqlite.DeleteTodo(id); err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(fmt.Sprintf("Deleted todo: %s (ID: %d)", todo.Title, todo.ID)), nil
	})
}
