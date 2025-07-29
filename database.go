package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Database struct {
	db *sql.DB
}

func NewDatabase() (*Database, error) {
	// 尝试使用SQLite文件数据库
	db, err := sql.Open("sqlite3", "./todos.db")
	if err != nil {
		// 如果失败，尝试使用内存数据库
		log.Println("Warning: Failed to open file database, trying memory database...")
		db, err = sql.Open("sqlite3", ":memory:")
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %v", err)
		}
	}

	// 创建表结构
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return &Database{db: db}, nil
}

func createTables(db *sql.DB) error {
	// 创建用户配置表
	userProfileSQL := `
	CREATE TABLE IF NOT EXISTS user_profile (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		timezone TEXT NOT NULL,
		work_schedule_start TEXT NOT NULL,
		work_schedule_end TEXT NOT NULL,
		work_schedule_days TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	// 创建待办事项表
	todosSQL := `
	CREATE TABLE IF NOT EXISTS todos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		description TEXT,
		priority TEXT DEFAULT 'medium',
		status TEXT DEFAULT 'pending',
		created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
		due_date DATETIME,
		last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
		estimated_duration TEXT,
		category TEXT DEFAULT 'personal'
	)`

	if _, err := db.Exec(userProfileSQL); err != nil {
		return fmt.Errorf("failed to create user_profile table: %v", err)
	}

	if _, err := db.Exec(todosSQL); err != nil {
		return fmt.Errorf("failed to create todos table: %v", err)
	}

	return nil
}

func (d *Database) ImportFromJSON(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read data.json: %v", err)
	}

	var dataStruct DataStructure
	if err := json.Unmarshal(data, &dataStruct); err != nil {
		return fmt.Errorf("failed to parse data.json: %v", err)
	}

	// 导入用户配置
	if err := d.importUserProfile(dataStruct.UserProfile); err != nil {
		return fmt.Errorf("failed to import user profile: %v", err)
	}

	// 导入待办事项
	if err := d.importTodos(dataStruct.Todos); err != nil {
		return fmt.Errorf("failed to import todos: %v", err)
	}

	log.Println("Data imported successfully from data.json")
	return nil
}

func (d *Database) importUserProfile(profile UserProfile) error {
	// 清空现有数据
	if _, err := d.db.Exec("DELETE FROM user_profile"); err != nil {
		return err
	}

	// 序列化工作日程
	workDaysJSON, err := json.Marshal(profile.WorkSchedule.WorkDays)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
		INSERT INTO user_profile (name, timezone, work_schedule_start, work_schedule_end, work_schedule_days)
		VALUES (?, ?, ?, ?, ?)
	`, profile.Name, profile.Timezone, profile.WorkSchedule.StartTime, profile.WorkSchedule.EndTime, string(workDaysJSON))

	return err
}

func (d *Database) importTodos(todos []Todo) error {
	// 清空现有数据
	if _, err := d.db.Exec("DELETE FROM todos"); err != nil {
		return err
	}

	stmt, err := d.db.Prepare(`
		INSERT INTO todos (id, title, description, priority, status, created_date, due_date, last_updated, estimated_duration, category)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, todo := range todos {
		_, err := stmt.Exec(
			todo.ID,
			todo.Title,
			todo.Description,
			todo.Priority,
			todo.Status,
			todo.CreatedDate,
			todo.DueDate,
			todo.LastUpdated,
			todo.EstimatedDuration,
			todo.Category,
		)
		if err != nil {
			return fmt.Errorf("failed to insert todo %d: %v", todo.ID, err)
		}
	}

	return nil
}

// CRUD 操作
func (d *Database) GetAllTodos() ([]Todo, error) {
	rows, err := d.db.Query(`
		SELECT id, title, description, priority, status, created_date, due_date, last_updated, estimated_duration, category
		FROM todos
		ORDER BY 
			CASE priority 
				WHEN 'urgent' THEN 1 
				WHEN 'high' THEN 2 
				WHEN 'medium' THEN 3 
				WHEN 'low' THEN 4 
			END,
			due_date ASC
	`)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		if dueDate.Valid {
			todo.DueDate = &dueDate.Time
		}
		todos = append(todos, todo)
	}

	return todos, nil
}

func (d *Database) GetTodoByID(id int) (*Todo, error) {
	var todo Todo
	var dueDate sql.NullTime
	err := d.db.QueryRow(`
		SELECT id, title, description, priority, status, created_date, due_date, last_updated, estimated_duration, category
		FROM todos WHERE id = ?
	`, id).Scan(
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
		return nil, err
	}
	if dueDate.Valid {
		todo.DueDate = &dueDate.Time
	}
	return &todo, nil
}

func (d *Database) CreateTodo(todo *Todo) error {
	result, err := d.db.Exec(`
		INSERT INTO todos (title, description, priority, status, created_date, due_date, last_updated, estimated_duration, category)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, todo.Title, todo.Description, todo.Priority, todo.Status, todo.CreatedDate, todo.DueDate, todo.LastUpdated, todo.EstimatedDuration, todo.Category)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	todo.ID = int(id)
	return nil
}

func (d *Database) UpdateTodo(todo *Todo) error {
	_, err := d.db.Exec(`
		UPDATE todos 
		SET title = ?, description = ?, priority = ?, status = ?, due_date = ?, last_updated = ?, estimated_duration = ?, category = ?
		WHERE id = ?
	`, todo.Title, todo.Description, todo.Priority, todo.Status, todo.DueDate, todo.LastUpdated, todo.EstimatedDuration, todo.Category, todo.ID)
	return err
}

func (d *Database) DeleteTodo(id int) error {
	_, err := d.db.Exec("DELETE FROM todos WHERE id = ?", id)
	return err
}

func (d *Database) GetUserProfile() (*UserProfile, error) {
	var profile UserProfile
	var workDaysJSON string
	err := d.db.QueryRow(`
		SELECT name, timezone, work_schedule_start, work_schedule_end, work_schedule_days
		FROM user_profile LIMIT 1
	`).Scan(&profile.Name, &profile.Timezone, &profile.WorkSchedule.StartTime, &profile.WorkSchedule.EndTime, &workDaysJSON)
	if err != nil {
		return nil, err
	}

	var workDays []string
	if err := json.Unmarshal([]byte(workDaysJSON), &workDays); err != nil {
		return nil, err
	}
	profile.WorkSchedule.WorkDays = workDays

	return &profile, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}
