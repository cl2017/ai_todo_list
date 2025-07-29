package main

import (
	"fmt"
	"fydeos/api"
	"fydeos/db"
	"fydeos/mcp"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"log"
	"net/http"
)

func main() {
	// 初始化数据库
	var err error
	_, err = db.NewSQLiteDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize SQLite database: %v", err)
	}
	defer db.DB.Close()

	// init MCP Server
	mcp.InitMCP()

	r := mux.NewRouter()

	// API routes
	r.HandleFunc("/api/todos", api.GetTodos).Methods("GET")
	r.HandleFunc("/api/todos", api.CreateTodo).Methods("POST")
	r.HandleFunc("/api/todos/{id}", api.UpdateTodo).Methods("PUT")
	r.HandleFunc("/api/todos/{id}", api.DeleteTodo).Methods("DELETE")

	r.HandleFunc("/api/ai/analyze", api.AiAnalyzeTasks).Methods("GET")
	r.HandleFunc("/api/ai/optimize", api.AiOptimizeSchedule).Methods("GET")

	// User profile route
	r.HandleFunc("/api/profile", api.GetUserProfile).Methods("GET")

	// Serve static files
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(r)
	handler = loggingMiddleware(handler)

	fmt.Println("🚀 AI智能待办助手服务器启动成功!")
	fmt.Println("📍 访问地址: http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", handler))
}

// HTTP请求日志中间件
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
