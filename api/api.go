package api

import (
	"encoding/json"
	"fydeos/db"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"time"
)

func GetTodos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	todos, err := db.DB.GetAllTodos()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(todos)
}

func CreateTodo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var todo db.Todo
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

	if err := db.DB.CreateTodo(&todo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(todo)
}

func UpdateTodo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var updatedTodo db.Todo
	err = json.NewDecoder(r.Body).Decode(&updatedTodo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 获取现有todo
	todo, err := db.DB.GetTodoByID(id)
	if err != nil {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	// 更新字段
	updatedTodo.ID = id
	updatedTodo.CreatedDate = todo.CreatedDate
	updatedTodo.LastUpdated = time.Now()

	if err := db.DB.UpdateTodo(&updatedTodo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updatedTodo)
}

func DeleteTodo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := db.DB.DeleteTodo(id); err != nil {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// MCP AI Functions
func AiAnalyzeTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	todos, err := db.DB.GetAllTodos()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// AI Analysis Logic
	now := time.Now()
	var urgentTasks []db.Todo
	var overdueTasks []db.Todo
	var staleTasks []db.Todo
	var todayTasks []db.Todo

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

func AiOptimizeSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	todos, err := db.DB.GetAllTodos()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get high priority pending tasks
	var priorityTasks []db.Todo
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

func GetUserProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	profile, err := db.DB.GetUserProfile()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(profile)
}
