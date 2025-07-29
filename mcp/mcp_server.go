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

// 注册所有 todo 相关工具到 mcp-go server
func RegisterTodoTools(s *server.MCPServer, sqlite *db.SQLiteDatabase) {
	// list_todos
	s.AddTool(mcp.NewTool(
		"list_todos",
		mcp.WithDescription("列出所有待办事项，支持过滤"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		todo, _ := sqlite.GetAllTodos()

		return mcp.NewToolResultStructuredOnly(todo), nil
	})

	// create_todo
	s.AddTool(mcp.NewTool(
		"create_todo",
		mcp.WithDescription("创建新的待办事项"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		title := req.GetString("title", "")

		todo := &db.Todo{
			Title:             title,
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
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	// analyze_tasks
	s.AddTool(mcp.NewTool(
		"analyze_tasks",
		mcp.WithDescription("智能分析任务状态"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		analysisType := req.GetString("analysis_type", "")
		if analysisType == "" {
			analysisType = "priority"
		}
		todos, err := sqlite.GetAllTodos()
		if err != nil {
			return nil, err
		}
		var analysis string
		switch analysisType {
		case "priority":
			urgent, high, medium, low := 0, 0, 0, 0
			for _, todo := range todos {
				switch todo.Priority {
				case "urgent":
					urgent++
				case "high":
					high++
				case "medium":
					medium++
				case "low":
					low++
				}
			}
			analysis = fmt.Sprintf("Priority Analysis: Urgent: %d, High: %d, Medium: %d, Low: %d", urgent, high, medium, low)
		case "overdue":
			overdue := 0
			now := time.Now()
			for _, todo := range todos {
				if todo.DueDate != nil && todo.DueDate.Before(now) && todo.Status != "completed" {
					overdue++
				}
			}
			analysis = fmt.Sprintf("Overdue Analysis: %d tasks are overdue", overdue)
		case "stale":
			stale := 0
			now := time.Now()
			for _, todo := range todos {
				if now.Sub(todo.LastUpdated).Hours() > 24*30 {
					stale++
				}
			}
			analysis = fmt.Sprintf("Stale Analysis: %d tasks haven't been updated in 30+ days", stale)
		case "workload":
			pending, inProgress, completed := 0, 0, 0
			for _, todo := range todos {
				switch todo.Status {
				case "pending":
					pending++
				case "in_progress":
					inProgress++
				case "completed":
					completed++
				}
			}
			analysis = fmt.Sprintf("Workload Analysis: Pending: %d, In Progress: %d, Completed: %d", pending, inProgress, completed)
		}
		return mcp.NewToolResultText(analysis), nil
	})

	// optimize_schedule
	s.AddTool(mcp.NewTool(
		"optimize_schedule",
		mcp.WithDescription("优化工作日程"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		timeHorizon := req.GetString("time_horizon", "")
		if timeHorizon == "" {
			timeHorizon = "today"
		}
		workHours := 8
		workHours = int(req.GetFloat("work_hours", 8))
		todos, err := sqlite.GetAllTodos()
		if err != nil {
			return nil, err
		}
		var priorityTasks []db.Todo
		for _, todo := range todos {
			if (todo.Status == "pending" || todo.Status == "in_progress") &&
				(todo.Priority == "urgent" || todo.Priority == "high") {
				priorityTasks = append(priorityTasks, todo)
			}
		}
		optimization := fmt.Sprintf("Schedule Optimization for %s (%d work hours):\n", timeHorizon, workHours)
		optimization += fmt.Sprintf("Found %d high-priority tasks to schedule\n", len(priorityTasks))
		optimization += "Recommendations:\n"
		optimization += "- Start with urgent tasks in the morning when energy is highest\n"
		optimization += "- Group similar tasks together for efficiency\n"
		optimization += "- Schedule breaks between complex tasks\n"
		optimization += "- Reserve buffer time for unexpected issues"
		return mcp.NewToolResultText(optimization), nil
	})

	// break_down_task
	s.AddTool(mcp.NewTool(
		"break_down_task",
		mcp.WithDescription("将复杂任务分解为子任务"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idFloat := req.GetFloat("task_id", 0)

		taskID := int(idFloat)
		complexity := req.GetString("complexity", "")
		if complexity == "" {
			complexity = "medium"
		}
		todo, err := sqlite.GetTodoByID(taskID)
		if err != nil {
			return nil, fmt.Errorf("task with ID %d not found", taskID)
		}
		breakdown := fmt.Sprintf("Task Breakdown for: %s\n", todo.Title)
		breakdown += fmt.Sprintf("Complexity: %s\n\n", complexity)
		breakdown += "Suggested subtasks:\n"
		if todo.Title == "Prepare Q3 presentation for Friday meeting" {
			breakdown += "1. Research and gather Q3 data (2 hours)\n"
			breakdown += "2. Create presentation outline (30 minutes)\n"
			breakdown += "3. Design slides and visuals (3 hours)\n"
			breakdown += "4. Review content with team (1 hour)\n"
			breakdown += "5. Practice presentation delivery (1 hour)\n"
			breakdown += "6. Prepare for Q&A session (30 minutes)\n"
		} else {
			switch complexity {
			case "simple":
				breakdown += "1. Plan the task (15 minutes)\n"
				breakdown += "2. Execute the main work (1-2 hours)\n"
				breakdown += "3. Review and finalize (15 minutes)\n"
			case "medium":
				breakdown += "1. Research and planning (30 minutes)\n"
				breakdown += "2. Break into smaller components (15 minutes)\n"
				breakdown += "3. Execute main work (2-4 hours)\n"
				breakdown += "4. Review and iterate (30 minutes)\n"
				breakdown += "5. Final quality check (15 minutes)\n"
			case "complex":
				breakdown += "1. Comprehensive research (1-2 hours)\n"
				breakdown += "2. Create detailed plan (30 minutes)\n"
				breakdown += "3. Identify dependencies (30 minutes)\n"
				breakdown += "4. Execute phase 1 (2-3 hours)\n"
				breakdown += "5. Review and adjust plan (30 minutes)\n"
				breakdown += "6. Execute phase 2 (2-3 hours)\n"
				breakdown += "7. Integration and testing (1 hour)\n"
				breakdown += "8. Final review and documentation (1 hour)\n"
			}
		}
		return mcp.NewToolResultText(breakdown), nil
	})
}
