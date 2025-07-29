package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/server"
	"log"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type UserProfile struct {
	Name         string       `json:"name"`
	Timezone     string       `json:"timezone"`
	WorkSchedule WorkSchedule `json:"work_schedule"`
}

type WorkSchedule struct {
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	WorkDays  []string `json:"work_days"`
}

type Todo struct {
	ID                int        `json:"id"`
	Title             string     `json:"title"`
	Description       string     `json:"description"`
	Priority          string     `json:"priority"`
	Status            string     `json:"status"`
	CreatedDate       time.Time  `json:"created_date"`
	DueDate           *time.Time `json:"due_date"`
	LastUpdated       time.Time  `json:"last_updated"`
	EstimatedDuration string     `json:"estimated_duration"`
	Category          string     `json:"category"`
}

type DataStructure struct {
	UserProfile UserProfile `json:"user_profile"`
	Todos       []Todo      `json:"todos"`
}

type AIRequest struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

type AIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// 全局数据库实例
var db *SQLiteDatabase

func getTodos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	todos, err := db.GetAllTodos()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(todos)
}

func createTodo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var todo Todo
	err := json.NewDecoder(r.Body).Decode(&todo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 设置默认值
	if todo.Status == "" {
		todo.Status = "pending"
	}
	if todo.Priority == "" {
		todo.Priority = "medium"
	}
	if todo.Category == "" {
		todo.Category = "personal"
	}
	todo.CreatedDate = time.Now()
	todo.LastUpdated = time.Now()

	if err := db.CreateTodo(&todo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(todo)
}

func updateTodo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var updatedTodo Todo
	err = json.NewDecoder(r.Body).Decode(&updatedTodo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 获取现有todo
	todo, err := db.GetTodoByID(id)
	if err != nil {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	// 更新字段
	updatedTodo.ID = id
	updatedTodo.CreatedDate = todo.CreatedDate
	updatedTodo.LastUpdated = time.Now()

	if err := db.UpdateTodo(&updatedTodo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updatedTodo)
}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := db.DeleteTodo(id); err != nil {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// MCP AI Functions
func aiAnalyzeTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	todos, err := db.GetAllTodos()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// AI Analysis Logic
	now := time.Now()
	var urgentTasks []Todo
	var overdueTasks []Todo
	var staleTasks []Todo
	var todayTasks []Todo

	for _, todo := range todos {
		// Check for urgent tasks
		if todo.Priority == "urgent" || todo.Priority == "high" {
			if todo.DueDate != nil && todo.DueDate.Before(now.AddDate(0, 0, 2)) {
				urgentTasks = append(urgentTasks, todo)
			}
		}

		// Check for overdue tasks
		if todo.DueDate != nil && todo.DueDate.Before(now) && todo.Status != "completed" {
			overdueTasks = append(overdueTasks, todo)
		}

		// Check for stale tasks (not updated in 30 days)
		if now.Sub(todo.LastUpdated).Hours() > 24*30 {
			staleTasks = append(staleTasks, todo)
		}

		// Check for today's tasks
		if todo.DueDate != nil && todo.DueDate.Format("2006-01-02") == now.Format("2006-01-02") {
			todayTasks = append(todayTasks, todo)
		}
	}

	analysis := map[string]interface{}{
		"total_tasks":   len(todos),
		"urgent_tasks":  urgentTasks,
		"overdue_tasks": overdueTasks,
		"stale_tasks":   staleTasks,
		"today_tasks":   todayTasks,
		"recommendations": []string{
			"优先处理紧急任务",
			"检查并更新过期任务",
			"考虑将大任务分解为小任务",
			"定期回顾和清理任务列表",
		},
	}

	json.NewEncoder(w).Encode(analysis)
}

func aiOptimizeSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	todos, err := db.GetAllTodos()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get high priority pending tasks
	var priorityTasks []Todo
	for _, todo := range todos {
		if (todo.Status == "pending" || todo.Status == "in_progress") &&
			(todo.Priority == "urgent" || todo.Priority == "high") {
			priorityTasks = append(priorityTasks, todo)
		}
	}

	// Sort by priority and due date
	priorityOrder := map[string]int{
		"urgent": 1,
		"high":   2,
		"medium": 3,
		"low":    4,
	}

	for i := 0; i < len(priorityTasks)-1; i++ {
		for j := i + 1; j < len(priorityTasks); j++ {
			pi := priorityOrder[priorityTasks[i].Priority]
			pj := priorityOrder[priorityTasks[j].Priority]

			if pi > pj || (pi == pj && priorityTasks[i].DueDate != nil && priorityTasks[j].DueDate != nil &&
				priorityTasks[i].DueDate.After(*priorityTasks[j].DueDate)) {
				priorityTasks[i], priorityTasks[j] = priorityTasks[j], priorityTasks[i]
			}
		}
	}

	// Limit to top 10
	if len(priorityTasks) > 10 {
		priorityTasks = priorityTasks[:10]
	}

	schedule := map[string]interface{}{
		"optimized_tasks": priorityTasks,
		"schedule_advice": []string{
			"上午处理紧急任务，精力最充沛",
			"将相似任务归类处理，提高效率",
			"复杂任务之间安排休息时间",
			"预留缓冲时间应对突发情况",
		},
	}

	json.NewEncoder(w).Encode(schedule)
}

func getUserProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	profile, err := db.GetUserProfile()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(profile)
}

// HTTP请求日志中间件
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func main() {
	// 初始化数据库
	var err error
	db, err = NewSQLiteDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize SQLite database: %v", err)
	}
	defer db.Close()

	// 导入初始数据
	//if err := db.ImportFromJSON("data.json"); err != nil {
	//	log.Printf("Warning: Failed to import data from data.json: %v", err)
	//}

	r := mux.NewRouter()

	// API routes
	r.HandleFunc("/api/todos", getTodos).Methods("GET")
	r.HandleFunc("/api/todos", createTodo).Methods("POST")
	r.HandleFunc("/api/todos/{id}", updateTodo).Methods("PUT")
	r.HandleFunc("/api/todos/{id}", deleteTodo).Methods("DELETE")

	// AI/MCP routes
	r.HandleFunc("/api/ai/analyze", aiAnalyzeTasks).Methods("GET")
	r.HandleFunc("/api/ai/optimize", aiOptimizeSchedule).Methods("GET")

	// User profile route
	r.HandleFunc("/api/profile", getUserProfile).Methods("GET")

	// Serve static files
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(r)

	// 添加日志中间件
	handler = loggingMiddleware(handler)

	// 新建mcp server
	s := server.NewMCPServer(
		"go-mcp-todo-list",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	RegisterTodoTools(s, db)

	if err := serveSSE(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

	fmt.Println("🚀 AI智能待办助手服务器启动成功!")
	fmt.Println("📍 访问地址: http://localhost:8081")
	fmt.Println("🤖 AI分析功能已启用")
	fmt.Println("📊 数据已从data.json导入到内存数据库")
	fmt.Println("🔌 MCP服务器HTTP API端点:")
	fmt.Println("   - POST /mcp/initialize")
	fmt.Println("   - GET  /mcp/tools/list")
	fmt.Println("   - POST /mcp/tools/call")
	log.Fatal(http.ListenAndServe(":8081", handler))
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
