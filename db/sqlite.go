package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
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
var DB *SQLiteDatabase

// SQLiteDatabase 使用SQLite3存储的数据库实现
type SQLiteDatabase struct {
	db     *sql.DB
	nextID int
}

func NewSQLiteDatabase() (*SQLiteDatabase, error) {
	// 打开位于当前目录的SQLite3数据库文件
	db, err := sql.Open("sqlite3", "./todos.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %v", err)
	}

	// 验证数据库连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite database: %v", err)
	}

	// 创建SQLite数据库实例
	sqliteDB := &SQLiteDatabase{
		db:     db,
		nextID: 1,
	}

	// 导入初始数据
	//if err := sqliteDB.ImportFromJSON("data.json"); err != nil {
	//	log.Printf("Warning: Failed to import data from data.json: %v", err)
	//}

	// 初始化数据库表
	if err := sqliteDB.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	// 获取当前最大ID
	sqliteDB.updateNextID()

	DB = sqliteDB

	return sqliteDB, nil
}

func (d *SQLiteDatabase) initDatabase() error {
	// 创建todos表
	todosTable := `CREATE TABLE IF NOT EXISTS todos (
		id INTEGER PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT,
		priority TEXT DEFAULT 'medium',
		status TEXT DEFAULT 'pending',
		created_date TIMESTAMP,
		due_date TIMESTAMP NULL,
		last_updated TIMESTAMP,
		estimated_duration TEXT,
		category TEXT DEFAULT 'personal'
	);`

	// 创建user_profile表
	userProfileTable := `CREATE TABLE IF NOT EXISTS user_profile (
		id INTEGER PRIMARY KEY,
		name TEXT,
		timezone TEXT,
		work_schedule_start TEXT,
		work_schedule_end TEXT,
		work_schedule_days TEXT
	);`

	// 执行SQL创建表
	_, err := d.db.Exec(todosTable)
	if err != nil {
		return fmt.Errorf("failed to create todos table: %v", err)
	}

	_, err = d.db.Exec(userProfileTable)
	if err != nil {
		return fmt.Errorf("failed to create user_profile table: %v", err)
	}

	return nil
}

func (d *SQLiteDatabase) updateNextID() {
	// 查询最大ID并更新nextID
	var maxID int
	row := d.db.QueryRow("SELECT COALESCE(MAX(id), 0) FROM todos")
	if err := row.Scan(&maxID); err != nil {
		log.Printf("Warning: Failed to get max ID: %v, using default 1", err)
		maxID = 0
	}

	d.nextID = maxID + 1
}

func (d *SQLiteDatabase) ImportFromJSON(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read data.json: %v", err)
	}

	var dataStruct DataStructure
	if err := json.Unmarshal(data, &dataStruct); err != nil {
		return fmt.Errorf("failed to parse data.json: %v", err)
	}

	// 开始事务
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	// 导入用户配置
	if dataStruct.UserProfile.Name != "" {
		// 首先删除现有的用户配置
		_, err = tx.Exec("DELETE FROM user_profile")
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to clear user profile: %v", err)
		}

		// 将工作日数组转换为JSON字符串
		workDaysJSON, err := json.Marshal(dataStruct.UserProfile.WorkSchedule.WorkDays)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to marshal work days: %v", err)
		}

		// 插入用户配置
		_, err = tx.Exec(
			"INSERT INTO user_profile (id, name, timezone, work_schedule_start, work_schedule_end, work_schedule_days) VALUES (1, ?, ?, ?, ?, ?)",
			dataStruct.UserProfile.Name,
			dataStruct.UserProfile.Timezone,
			dataStruct.UserProfile.WorkSchedule.StartTime,
			dataStruct.UserProfile.WorkSchedule.EndTime,
			string(workDaysJSON),
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert user profile: %v", err)
		}
	}

	// 导入待办事项
	if len(dataStruct.Todos) > 0 {
		// 插入待办事项数据
		for _, todo := range dataStruct.Todos {
			var dueDate interface{}
			if todo.DueDate != nil {
				dueDate = todo.DueDate
			} else {
				dueDate = nil
			}

			_, err = tx.Exec(
				"INSERT OR REPLACE INTO todos (id, title, description, priority, status, created_date, due_date, last_updated, estimated_duration, category) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
				todo.ID,
				todo.Title,
				todo.Description,
				todo.Priority,
				todo.Status,
				todo.CreatedDate,
				dueDate,
				todo.LastUpdated,
				todo.EstimatedDuration,
				todo.Category,
			)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to insert todo: %v", err)
			}
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	// 更新nextID
	d.updateNextID()

	log.Println("Data imported successfully from data.json to SQLite database")
	return nil
}

// CRUD 操作
func (d *SQLiteDatabase) GetAllTodos() ([]Todo, error) {
	rows, err := d.db.Query(
		"SELECT id, title, description, priority, status, created_date, due_date, last_updated, estimated_duration, category FROM todos",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query todos: %v", err)
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		var dueDate sql.NullTime

		err := rows.Scan(
			&todo.ID,
			&todo.Title,
			&todo.Description,
			&todo.Priority,
			&todo.Status,
			&todo.CreatedDate,
			&dueDate,
			&todo.LastUpdated,
			&todo.EstimatedDuration,
			&todo.Category,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan todo: %v", err)
		}

		if dueDate.Valid {
			todo.DueDate = &dueDate.Time
		} else {
			todo.DueDate = nil
		}

		todos = append(todos, todo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating todos rows: %v", err)
	}

	// 按优先级和截止日期排序
	priorityOrder := map[string]int{
		"urgent": 1,
		"high":   2,
		"medium": 3,
		"low":    4,
	}

	for i := 0; i < len(todos)-1; i++ {
		for j := i + 1; j < len(todos); j++ {
			pi := priorityOrder[todos[i].Priority]
			pj := priorityOrder[todos[j].Priority]

			if pi > pj || (pi == pj && todos[i].DueDate != nil && todos[j].DueDate != nil &&
				todos[i].DueDate.After(*todos[j].DueDate)) {
				todos[i], todos[j] = todos[j], todos[i]
			}
		}
	}

	return todos, nil
}

func (d *SQLiteDatabase) GetTodoByID(id int) (*Todo, error) {
	var todo Todo
	var dueDate sql.NullTime

	row := d.db.QueryRow(
		"SELECT id, title, description, priority, status, created_date, due_date, last_updated, estimated_duration, category FROM todos WHERE id = ?",
		id,
	)

	err := row.Scan(
		&todo.ID,
		&todo.Title,
		&todo.Description,
		&todo.Priority,
		&todo.Status,
		&todo.CreatedDate,
		&dueDate,
		&todo.LastUpdated,
		&todo.EstimatedDuration,
		&todo.Category,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("todo with ID %d not found", id)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get todo: %v", err)
	}

	if dueDate.Valid {
		todo.DueDate = &dueDate.Time
	} else {
		todo.DueDate = nil
	}

	return &todo, nil
}

func (d *SQLiteDatabase) CreateTodo(todo *Todo) error {
	todo.ID = d.nextID
	todo.CreatedDate = time.Now()
	todo.LastUpdated = time.Now()

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

	var dueDate interface{}
	if todo.DueDate != nil {
		dueDate = todo.DueDate
	} else {
		dueDate = nil
	}

	_, err := d.db.Exec(
		"INSERT INTO todos (id, title, description, priority, status, created_date, due_date, last_updated, estimated_duration, category) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		todo.ID,
		todo.Title,
		todo.Description,
		todo.Priority,
		todo.Status,
		todo.CreatedDate,
		dueDate,
		todo.LastUpdated,
		todo.EstimatedDuration,
		todo.Category,
	)

	if err != nil {
		return fmt.Errorf("failed to create todo: %v", err)
	}

	d.nextID++
	return nil
}

func (d *SQLiteDatabase) UpdateTodo(todo *Todo) error {
	// 检查待办事项是否存在
	existingTodo, err := d.GetTodoByID(todo.ID)
	if err != nil {
		return err
	}

	// 保留创建日期，更新最后修改日期
	todo.CreatedDate = existingTodo.CreatedDate
	todo.LastUpdated = time.Now()

	var dueDate interface{}
	if todo.DueDate != nil {
		dueDate = todo.DueDate
	} else {
		dueDate = nil
	}

	_, err = d.db.Exec(
		"UPDATE todos SET title = ?, description = ?, priority = ?, status = ?, due_date = ?, last_updated = ?, estimated_duration = ?, category = ? WHERE id = ?",
		todo.Title,
		todo.Description,
		todo.Priority,
		todo.Status,
		dueDate,
		todo.LastUpdated,
		todo.EstimatedDuration,
		todo.Category,
		todo.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update todo: %v", err)
	}

	return nil
}

func (d *SQLiteDatabase) DeleteTodo(id int) error {
	result, err := d.db.Exec("DELETE FROM todos WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete todo: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking affected rows: %v", err)
	}

	if affected == 0 {
		return fmt.Errorf("todo with ID %d not found", id)
	}

	return nil
}

func (d *SQLiteDatabase) GetUserProfile() (*UserProfile, error) {
	row := d.db.QueryRow(
		"SELECT name, timezone, work_schedule_start, work_schedule_end, work_schedule_days FROM user_profile LIMIT 1",
	)

	var profile UserProfile
	var workSchedule WorkSchedule
	var workDaysJSON string

	err := row.Scan(
		&profile.Name,
		&profile.Timezone,
		&workSchedule.StartTime,
		&workSchedule.EndTime,
		&workDaysJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user profile not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %v", err)
	}

	// 解析工作日JSON
	var workDays []string
	if err := json.Unmarshal([]byte(workDaysJSON), &workDays); err != nil {
		return nil, fmt.Errorf("failed to unmarshal work days: %v", err)
	}
	workSchedule.WorkDays = workDays

	profile.WorkSchedule = workSchedule
	return &profile, nil
}

func (d *SQLiteDatabase) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}
