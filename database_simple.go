package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// SimpleDatabase 使用内存存储的简单数据库实现
type SimpleDatabase struct {
	todos       []Todo
	userProfile *UserProfile
	mutex       sync.RWMutex
	nextID      int
}

func NewSimpleDatabase() (*SimpleDatabase, error) {
	db := &SimpleDatabase{
		todos:  make([]Todo, 0),
		nextID: 1,
	}
	return db, nil
}

func (d *SimpleDatabase) ImportFromJSON(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read data.json: %v", err)
	}

	var dataStruct DataStructure
	if err := json.Unmarshal(data, &dataStruct); err != nil {
		return fmt.Errorf("failed to parse data.json: %v", err)
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.userProfile = &dataStruct.UserProfile
	d.todos = dataStruct.Todos

	// 找到最大ID，设置nextID
	maxID := 0
	for _, todo := range d.todos {
		if todo.ID > maxID {
			maxID = todo.ID
		}
	}
	d.nextID = maxID + 1

	log.Println("Data imported successfully from data.json")
	return nil
}

// CRUD 操作
func (d *SimpleDatabase) GetAllTodos() ([]Todo, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	// 复制todos切片以避免并发问题
	todosCopy := make([]Todo, len(d.todos))
	copy(todosCopy, d.todos)

	// 按优先级和截止日期排序
	priorityOrder := map[string]int{
		"urgent": 1,
		"high":   2,
		"medium": 3,
		"low":    4,
	}

	for i := 0; i < len(todosCopy)-1; i++ {
		for j := i + 1; j < len(todosCopy); j++ {
			pi := priorityOrder[todosCopy[i].Priority]
			pj := priorityOrder[todosCopy[j].Priority]

			if pi > pj || (pi == pj && todosCopy[i].DueDate != nil && todosCopy[j].DueDate != nil &&
				todosCopy[i].DueDate.After(*todosCopy[j].DueDate)) {
				todosCopy[i], todosCopy[j] = todosCopy[j], todosCopy[i]
			}
		}
	}

	return todosCopy, nil
}

func (d *SimpleDatabase) GetTodoByID(id int) (*Todo, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for _, todo := range d.todos {
		if todo.ID == id {
			return &todo, nil
		}
	}
	return nil, fmt.Errorf("todo with ID %d not found", id)
}

func (d *SimpleDatabase) CreateTodo(todo *Todo) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	todo.ID = d.nextID
	d.nextID++
	todo.CreatedDate = time.Now()
	todo.LastUpdated = time.Now()

	d.todos = append(d.todos, *todo)
	return nil
}

func (d *SimpleDatabase) UpdateTodo(todo *Todo) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for i, existingTodo := range d.todos {
		if existingTodo.ID == todo.ID {
			todo.CreatedDate = existingTodo.CreatedDate
			todo.LastUpdated = time.Now()
			d.todos[i] = *todo
			return nil
		}
	}
	return fmt.Errorf("todo with ID %d not found", todo.ID)
}

func (d *SimpleDatabase) DeleteTodo(id int) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for i, todo := range d.todos {
		if todo.ID == id {
			d.todos = append(d.todos[:i], d.todos[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("todo with ID %d not found", id)
}

func (d *SimpleDatabase) GetUserProfile() (*UserProfile, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.userProfile == nil {
		return nil, fmt.Errorf("user profile not found")
	}
	return d.userProfile, nil
}

func (d *SimpleDatabase) Close() error {
	// 内存数据库不需要关闭
	return nil
}
