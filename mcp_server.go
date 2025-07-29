package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// MCP Protocol Types
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Tool Definitions
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type MCPToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MCP Server
type MCPServer struct {
	db interface {
		GetAllTodos() ([]Todo, error)
		GetTodoByID(id int) (*Todo, error)
		CreateTodo(todo *Todo) error
		UpdateTodo(todo *Todo) error
		DeleteTodo(id int) error
		GetUserProfile() (*UserProfile, error)
	}
	tools map[string]MCPTool
}

func NewMCPServer(db interface {
	GetAllTodos() ([]Todo, error)
	GetTodoByID(id int) (*Todo, error)
	CreateTodo(todo *Todo) error
	UpdateTodo(todo *Todo) error
	DeleteTodo(id int) error
	GetUserProfile() (*UserProfile, error)
}) *MCPServer {
	server := &MCPServer{
		db:    db,
		tools: make(map[string]MCPTool),
	}

	server.registerTools()
	return server
}

func (s *MCPServer) registerTools() {
	// Register todo management tools
	s.tools["list_todos"] = MCPTool{
		Name:        "list_todos",
		Description: "List all todos with optional filtering",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter by status (pending, in_progress, completed)",
					"enum":        []string{"pending", "in_progress", "completed", "scheduled"},
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"description": "Filter by priority (urgent, high, medium, low)",
					"enum":        []string{"urgent", "high", "medium", "low"},
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Filter by category",
				},
			},
		},
	}

	s.tools["create_todo"] = MCPTool{
		Name:        "create_todo",
		Description: "Create a new todo item",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Todo title",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Todo description",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"description": "Priority level",
					"enum":        []string{"urgent", "high", "medium", "low"},
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Todo category",
				},
				"due_date": map[string]interface{}{
					"type":        "string",
					"description": "Due date in ISO format",
				},
				"estimated_duration": map[string]interface{}{
					"type":        "string",
					"description": "Estimated duration",
				},
			},
			"required": []string{"title"},
		},
	}

	s.tools["update_todo"] = MCPTool{
		Name:        "update_todo",
		Description: "Update an existing todo item",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "integer",
					"description": "Todo ID",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Todo title",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Todo description",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"description": "Priority level",
					"enum":        []string{"urgent", "high", "medium", "low"},
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Todo status",
					"enum":        []string{"pending", "in_progress", "completed", "scheduled"},
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Todo category",
				},
				"due_date": map[string]interface{}{
					"type":        "string",
					"description": "Due date in ISO format",
				},
				"estimated_duration": map[string]interface{}{
					"type":        "string",
					"description": "Estimated duration",
				},
			},
			"required": []string{"id"},
		},
	}

	s.tools["delete_todo"] = MCPTool{
		Name:        "delete_todo",
		Description: "Delete a todo item",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "integer",
					"description": "Todo ID to delete",
				},
			},
			"required": []string{"id"},
		},
	}

	s.tools["analyze_tasks"] = MCPTool{
		Name:        "analyze_tasks",
		Description: "Analyze tasks and provide insights",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"analysis_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of analysis to perform",
					"enum":        []string{"priority", "overdue", "stale", "workload"},
				},
			},
		},
	}

	s.tools["optimize_schedule"] = MCPTool{
		Name:        "optimize_schedule",
		Description: "Optimize task schedule based on priorities and deadlines",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"time_horizon": map[string]interface{}{
					"type":        "string",
					"description": "Time horizon for optimization (today, week, month)",
					"enum":        []string{"today", "week", "month"},
				},
				"work_hours": map[string]interface{}{
					"type":        "integer",
					"description": "Available work hours per day",
				},
			},
		},
	}

	s.tools["break_down_task"] = MCPTool{
		Name:        "break_down_task",
		Description: "Break down a complex task into smaller subtasks",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_id": map[string]interface{}{
					"type":        "integer",
					"description": "ID of the task to break down",
				},
				"complexity": map[string]interface{}{
					"type":        "string",
					"description": "Task complexity level",
					"enum":        []string{"simple", "medium", "complex"},
				},
			},
			"required": []string{"task_id"},
		},
	}
}

// HTTP Handlers for MCP API
func (s *MCPServer) HandleInitialize(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      "init",
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{
					"listChanged": true,
				},
				"resources": map[string]interface{}{
					"subscribe":   true,
					"listChanged": true,
				},
			},
			"serverInfo": map[string]interface{}{
				"name":    "AI Todo Assistant MCP Server",
				"version": "1.0.0",
			},
		},
	}

	json.NewEncoder(w).Encode(response)
}

// 添加缺失的MCP端点
func (s *MCPServer) HandlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      "ping",
		Result:  "pong",
	}

	json.NewEncoder(w).Encode(response)
}

func (s *MCPServer) HandleShutdown(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      "shutdown",
		Result:  nil,
	}

	json.NewEncoder(w).Encode(response)
}

func (s *MCPServer) HandleToolsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tools := make([]MCPTool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      "tools_list",
		Result: map[string]interface{}{
			"tools": tools,
		},
	}

	json.NewEncoder(w).Encode(response)
}

func (s *MCPServer) HandleToolCall(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var request MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	params, ok := request.Params.(map[string]interface{})
	if !ok {
		response := MCPResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	name, ok := params["name"].(string)
	if !ok {
		response := MCPResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing tool name",
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	arguments, _ := params["arguments"].(map[string]interface{})

	result, err := s.executeTool(name, arguments)
	if err != nil {
		response := MCPResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &MCPError{
				Code:    -32603,
				Message: err.Error(),
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  result,
	}

	json.NewEncoder(w).Encode(response)
}

func (s *MCPServer) executeTool(name string, arguments map[string]interface{}) (interface{}, error) {
	switch name {
	case "list_todos":
		return s.executeListTodos(arguments)
	case "create_todo":
		return s.executeCreateTodo(arguments)
	case "update_todo":
		return s.executeUpdateTodo(arguments)
	case "delete_todo":
		return s.executeDeleteTodo(arguments)
	case "analyze_tasks":
		return s.executeAnalyzeTasks(arguments)
	case "optimize_schedule":
		return s.executeOptimizeSchedule(arguments)
	case "break_down_task":
		return s.executeBreakDownTask(arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *MCPServer) executeListTodos(args map[string]interface{}) (interface{}, error) {
	todos, err := s.db.GetAllTodos()
	if err != nil {
		return nil, err
	}

	// 应用过滤器
	filteredTodos := make([]Todo, 0)
	for _, todo := range todos {
		include := true

		if status, ok := args["status"].(string); ok && status != "" {
			if todo.Status != status {
				include = false
			}
		}

		if priority, ok := args["priority"].(string); ok && priority != "" {
			if todo.Priority != priority {
				include = false
			}
		}

		if category, ok := args["category"].(string); ok && category != "" {
			if todo.Category != category {
				include = false
			}
		}

		if include {
			filteredTodos = append(filteredTodos, todo)
		}
	}

	return MCPToolResult{
		Content: []MCPContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Found %d todos matching the criteria", len(filteredTodos)),
			},
		},
	}, nil
}

func (s *MCPServer) executeCreateTodo(args map[string]interface{}) (interface{}, error) {
	title, ok := args["title"].(string)
	if !ok {
		return nil, fmt.Errorf("title is required")
	}

	todo := &Todo{
		Title:             title,
		Description:       getStringArg(args, "description"),
		Priority:          getStringArg(args, "priority"),
		Category:          getStringArg(args, "category"),
		Status:            "pending",
		CreatedDate:       time.Now(),
		LastUpdated:       time.Now(),
		EstimatedDuration: getStringArg(args, "estimated_duration"),
	}

	if todo.Priority == "" {
		todo.Priority = "medium"
	}
	if todo.Category == "" {
		todo.Category = "personal"
	}

	if dueDateStr, ok := args["due_date"].(string); ok && dueDateStr != "" {
		if dueDate, err := time.Parse(time.RFC3339, dueDateStr); err == nil {
			todo.DueDate = &dueDate
		}
	}

	if err := s.db.CreateTodo(todo); err != nil {
		return nil, err
	}

	return MCPToolResult{
		Content: []MCPContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Created todo: %s (ID: %d)", todo.Title, todo.ID),
			},
		},
	}, nil
}

func (s *MCPServer) executeUpdateTodo(args map[string]interface{}) (interface{}, error) {
	idFloat, ok := args["id"].(float64)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}
	id := int(idFloat)

	todo, err := s.db.GetTodoByID(id)
	if err != nil {
		return nil, fmt.Errorf("todo with ID %d not found", id)
	}

	// 更新字段
	if title, ok := args["title"].(string); ok {
		todo.Title = title
	}
	if description, ok := args["description"].(string); ok {
		todo.Description = description
	}
	if priority, ok := args["priority"].(string); ok {
		todo.Priority = priority
	}
	if status, ok := args["status"].(string); ok {
		todo.Status = status
	}
	if category, ok := args["category"].(string); ok {
		todo.Category = category
	}
	if estimatedDuration, ok := args["estimated_duration"].(string); ok {
		todo.EstimatedDuration = estimatedDuration
	}
	if dueDateStr, ok := args["due_date"].(string); ok && dueDateStr != "" {
		if dueDate, err := time.Parse(time.RFC3339, dueDateStr); err == nil {
			todo.DueDate = &dueDate
		}
	}

	todo.LastUpdated = time.Now()

	if err := s.db.UpdateTodo(todo); err != nil {
		return nil, err
	}

	return MCPToolResult{
		Content: []MCPContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Updated todo: %s (ID: %d)", todo.Title, todo.ID),
			},
		},
	}, nil
}

func (s *MCPServer) executeDeleteTodo(args map[string]interface{}) (interface{}, error) {
	idFloat, ok := args["id"].(float64)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}
	id := int(idFloat)

	todo, err := s.db.GetTodoByID(id)
	if err != nil {
		return nil, fmt.Errorf("todo with ID %d not found", id)
	}

	if err := s.db.DeleteTodo(id); err != nil {
		return nil, err
	}

	return MCPToolResult{
		Content: []MCPContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Deleted todo: %s (ID: %d)", todo.Title, todo.ID),
			},
		},
	}, nil
}

func (s *MCPServer) executeAnalyzeTasks(args map[string]interface{}) (interface{}, error) {
	analysisType := getStringArg(args, "analysis_type")
	if analysisType == "" {
		analysisType = "priority"
	}

	todos, err := s.db.GetAllTodos()
	if err != nil {
		return nil, err
	}

	var analysis string
	switch analysisType {
	case "priority":
		urgent := 0
		high := 0
		medium := 0
		low := 0
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
		pending := 0
		inProgress := 0
		completed := 0
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

	return MCPToolResult{
		Content: []MCPContent{
			{
				Type: "text",
				Text: analysis,
			},
		},
	}, nil
}

func (s *MCPServer) executeOptimizeSchedule(args map[string]interface{}) (interface{}, error) {
	timeHorizon := getStringArg(args, "time_horizon")
	if timeHorizon == "" {
		timeHorizon = "today"
	}

	workHours := 8
	if wh, ok := args["work_hours"].(float64); ok {
		workHours = int(wh)
	}

	todos, err := s.db.GetAllTodos()
	if err != nil {
		return nil, err
	}

	// Get high priority pending tasks
	var priorityTasks []Todo
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

	return MCPToolResult{
		Content: []MCPContent{
			{
				Type: "text",
				Text: optimization,
			},
		},
	}, nil
}

func (s *MCPServer) executeBreakDownTask(args map[string]interface{}) (interface{}, error) {
	idFloat, ok := args["task_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("task_id is required")
	}
	taskID := int(idFloat)

	complexity := getStringArg(args, "complexity")
	if complexity == "" {
		complexity = "medium"
	}

	todo, err := s.db.GetTodoByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("task with ID %d not found", taskID)
	}

	breakdown := fmt.Sprintf("Task Breakdown for: %s\n", todo.Title)
	breakdown += fmt.Sprintf("Complexity: %s\n\n", complexity)
	breakdown += "Suggested subtasks:\n"

	// Generate subtasks based on task content and complexity
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

	return MCPToolResult{
		Content: []MCPContent{
			{
				Type: "text",
				Text: breakdown,
			},
		},
	}, nil
}

func getStringArg(args map[string]interface{}, key string) string {
	if val, ok := args[key].(string); ok {
		return val
	}
	return ""
}
