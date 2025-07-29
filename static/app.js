const { createApp } = Vue;

createApp({
    data() {
        return {
            todos: [],
            aiAnalysis: null,
            userProfile: {
                name: 'Sarah Johnson',
                timezone: 'America/Los_Angeles'
            },
            showAddForm: false,
            editingTodo: null,
            filterStatus: '',
            filterPriority: '',
            filterCategory: '',
            newTodo: {
                title: '',
                description: '',
                priority: 'medium',
                category: 'personal',
                due_date: '',
                estimated_duration: ''
            },
            isLoadingAI: false
        }
    },
    computed: {
        filteredTodos() {
            return this.todos.filter(todo => {
                const statusMatch = !this.filterStatus || todo.status === this.filterStatus;
                const priorityMatch = !this.filterPriority || todo.priority === this.filterPriority;
                const categoryMatch = !this.filterCategory || todo.category === this.filterCategory;
                return statusMatch && priorityMatch && categoryMatch;
            });
        }
    },
    methods: {

        async fetchTodos() {
            try {
                const response = await axios.get('/api/todos');
                this.todos = response.data || [];
            } catch (error) {
                console.error('获取任务失败:', error);
                this.showNotification('获取任务失败', 'error');
            }
        },


        async addTodo() {
            try {
                const todoData = { ...this.newTodo };
                if (todoData.due_date) {
                    todoData.due_date = new Date(todoData.due_date).toISOString();
                }
                
                const response = await axios.post('/api/todos', todoData);

                this.todos.push(response.data);
                this.resetNewTodo();
                this.showAddForm = false;
                this.showNotification('任务添加成功', 'success');
                this.getAIAnalysis(); // 刷新AI分析
            } catch (error) {
                console.error('添加任务失败:', error);
                this.showNotification('添加任务失败', 'error');
            }
        },

        async breakDownTask(todo) {
            try {
                this.isLoadingAI = true;
                // 使用简单的本地分解逻辑
                const subtasks = this.generateSubtasks(todo);
                this.showTaskBreakdown(todo.title, subtasks);
            } catch (error) {
                console.error('任务分解失败:', error);
                this.showNotification('任务分解失败', 'error');
            } finally {
                this.isLoadingAI = false;
            }
        },

        showTaskBreakdown(taskTitle, subtasks) {
            // 创建子任务列表的HTML
            const subtasksHtml = subtasks.map(task => 
                `<div class="subtask">
                    <h4>${task.title}</h4>
                    <p>${task.description}</p>
                    <p>预计时间: ${task.estimated_duration}</p>
                </div>`
            ).join('');

            // Create a modal to show task breakdown
            const modal = document.createElement('div');
            modal.className = 'modal-overlay';
            modal.innerHTML = `
                <div class="modal" style="max-width: 600px;">
                    <h3>任务分解: ${taskTitle}</h3>
                    <div style="margin: 20px 0;">
                        ${subtasksHtml || '<p>无法为此任务生成子任务</p>'}
                    </div>
                    <div class="form-buttons">
                        <button onclick="this.closest('.modal-overlay').remove()">关闭</button>
                    </div>
                </div>
            `;
            document.body.appendChild(modal);
        },

        async optimizeMySchedule() {
            try {
                this.isLoadingAI = true;
                const response = await axios.get('/api/ai/optimize');
                const optimization = response.data;

                // 显示日程优化建议
                const modal = document.createElement('div');
                modal.className = 'modal-overlay';
                modal.innerHTML = `
                    <div class="modal" style="max-width: 600px;">
                        <h3>日程优化建议</h3>
                        <div style="margin: 20px 0;">
                            <h4>建议任务顺序:</h4>
                            <ul>
                                ${optimization.optimized_tasks.map(task => 
                                    `<li>${task.title} (${this.getPriorityText(task.priority)})</li>`
                                ).join('')}
                            </ul>
                            <h4>优化提示:</h4>
                            <ul>
                                ${optimization.schedule_advice.map(advice => 
                                    `<li>${advice}</li>`
                                ).join('')}
                            </ul>
                        </div>
                        <div class="form-buttons">
                            <button onclick="this.closest('.modal-overlay').remove()">关闭</button>
                        </div>
                    </div>
                `;
                document.body.appendChild(modal);

                this.showNotification('日程优化建议已生成', 'success');
            } catch (error) {
                console.error('日程优化失败:', error);
                this.showNotification('日程优化失败', 'error');
            } finally {
                this.isLoadingAI = false;
            }
        },

        async updateTodo() {
            try {
                const todoData = { ...this.editingTodo };
                if (todoData.due_date && typeof todoData.due_date === 'string') {
                    todoData.due_date = new Date(todoData.due_date).toISOString();
                }
                
                const response = await axios.put(`/api/todos/${this.editingTodo.id}`, todoData);
                const index = this.todos.findIndex(t => t.id === this.editingTodo.id);
                if (index !== -1) {
                    this.todos[index] = response.data;
                }
                this.closeEditModal();
                this.showNotification('任务更新成功', 'success');
                this.getAIAnalysis(); // 刷新AI分析
            } catch (error) {
                console.error('更新任务失败:', error);
                this.showNotification('更新任务失败', 'error');
            }
        },

        async updateTodoStatus(todo, newStatus) {
            try {
                const updatedTodo = { ...todo, status: newStatus };
                if (updatedTodo.due_date && typeof updatedTodo.due_date === 'string') {
                    updatedTodo.due_date = new Date(updatedTodo.due_date).toISOString();
                }
                
                const response = await axios.put(`/api/todos/${todo.id}`, updatedTodo);
                const index = this.todos.findIndex(t => t.id === todo.id);
                if (index !== -1) {
                    this.todos[index] = response.data;
                }
                this.showNotification('任务状态更新成功', 'success');
                this.getAIAnalysis(); // 刷新AI分析
            } catch (error) {
                console.error('更新任务状态失败:', error);
                this.showNotification('更新任务状态失败', 'error');
            }
        },

        async deleteTodo(id) {
            if (!confirm('确定要删除这个任务吗？')) {
                return;
            }
            
            try {
                await axios.delete(`/api/todos/${id}`);
                this.todos = this.todos.filter(t => t.id !== id);
                this.showNotification('任务删除成功', 'success');
                this.getAIAnalysis(); // 刷新AI分析
            } catch (error) {
                console.error('删除任务失败:', error);
                this.showNotification('删除任务失败', 'error');
            }
        },

        async getAIAnalysis() {
            try {
                const response = await axios.get('/api/ai/analyze');
                this.aiAnalysis = response.data;
            } catch (error) {
                console.error('获取AI分析失败:', error);
                this.showNotification('获取AI分析失败', 'error');
            }
        },

        async getAIOptimization() {
            try {
                const response = await axios.get('/api/ai/optimize');
                console.log('AI优化建议:', response.data);
                this.showNotification('AI优化建议已生成', 'success');
            } catch (error) {
                console.error('获取AI优化失败:', error);
                this.showNotification('获取AI优化失败', 'error');
            }
        },

        editTodo(todo) {
            this.editingTodo = { ...todo };
            // 格式化日期为datetime-local输入格式
            if (this.editingTodo.due_date) {
                const date = new Date(this.editingTodo.due_date);
                this.editingTodo.due_date = date.toISOString().slice(0, 16);
            }
        },

        closeEditModal() {
            this.editingTodo = null;
        },

        resetNewTodo() {
            this.newTodo = {
                title: '',
                description: '',
                priority: 'medium',
                category: 'personal',
                due_date: '',
                estimated_duration: ''
            };
        },

        getPriorityText(priority) {
            const priorityMap = {
                'urgent': '紧急',
                'high': '高',
                'medium': '中',
                'low': '低'
            };
            return priorityMap[priority] || priority;
        },

        getCategoryText(category) {
            const categoryMap = {
                'work': '工作',
                'personal': '个人',
                'health': '健康',
                'financial': '财务',
                'household': '家务',
                'family': '家庭',
                'career': '职业',
                'hobby': '爱好',
                'personal_development': '个人发展',
                'maintenance': '维护',
                'tech': '技术',
                'social': '社交',
                'volunteer': '志愿',
                'gardening': '园艺',
                'travel': '旅行',
                'wellness': '健康',
                'important': '重要',
                'safety': '安全',
                'security': '安全'
            };
            return categoryMap[category] || category;
        },

        getStatusText(status) {
            const statusMap = {
                'pending': '待处理',
                'in_progress': '进行中',
                'completed': '已完成',
                'scheduled': '已安排',
                'ongoing': '持续进行'
            };
            return statusMap[status] || status;
        },

        formatDate(dateString) {
            if (!dateString) return '';
            const date = new Date(dateString);
            return date.toLocaleString('zh-CN', {
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit'
            });
        },

        showNotification(message, type = 'info') {
            // 简单的通知实现
            const notification = document.createElement('div');
            notification.className = `notification ${type}`;
            notification.textContent = message;
            notification.style.cssText = `
                position: fixed;
                top: 20px;
                right: 20px;
                padding: 12px 20px;
                border-radius: 4px;
                color: white;
                font-weight: bold;
                z-index: 1000;
                animation: slideIn 0.3s ease-out;
            `;
            
            if (type === 'success') {
                notification.style.backgroundColor = '#4CAF50';
            } else if (type === 'error') {
                notification.style.backgroundColor = '#f44336';
            } else {
                notification.style.backgroundColor = '#2196F3';
            }
            
            document.body.appendChild(notification);
            
            setTimeout(() => {
                notification.style.animation = 'slideOut 0.3s ease-in';
                setTimeout(() => {
                    document.body.removeChild(notification);
                }, 300);
            }, 3000);
        },

        // 智能任务分解功能
        async breakDownTask(task) {
            // 这里可以集成真正的AI服务来分解任务
            const subtasks = this.generateSubtasks(task);
            for (const subtask of subtasks) {
                await this.addSubtask(subtask, task.id);
            }
        },

        generateSubtasks(task) {
            // 简单的任务分解逻辑
            const subtasks = [];
            if (task.title.includes('演示') || task.title.includes('presentation')) {
                subtasks.push({
                    title: `研究${task.title}的数据`,
                    description: '收集和分析相关数据',
                    priority: task.priority,
                    category: task.category,
                    estimated_duration: '2小时'
                });
                subtasks.push({
                    title: `创建${task.title}的幻灯片`,
                    description: '设计和制作演示幻灯片',
                    priority: task.priority,
                    category: task.category,
                    estimated_duration: '3小时'
                });
                subtasks.push({
                    title: `排练${task.title}`,
                    description: '练习演示内容和时间控制',
                    priority: task.priority,
                    category: task.category,
                    estimated_duration: '1小时'
                });
            }
            return subtasks;
        },

        // 智能优先级调整
        adjustPriorities() {
            const now = new Date();
            this.todos.forEach(todo => {
                if (todo.due_date) {
                    const dueDate = new Date(todo.due_date);
                    const daysUntilDue = (dueDate - now) / (1000 * 60 * 60 * 24);
                    
                    if (daysUntilDue < 1 && todo.priority !== 'urgent') {
                        todo.priority = 'urgent';
                        this.updateTodo(todo);
                    } else if (daysUntilDue < 3 && todo.priority === 'low') {
                        todo.priority = 'medium';
                        this.updateTodo(todo);
                    }
                }
            });
        }
    },

    async mounted() {
        // 获取待办事项和AI分析
        await this.fetchTodos();
        await this.getAIAnalysis();

        // 定期刷新AI分析
        setInterval(() => {
            this.getAIAnalysis();
        }, 300000); // 每5分钟刷新一次
        
        // 智能优先级调整
        setInterval(() => {
            this.adjustPriorities();
        }, 3600000); // 每小时检查一次
    }
}).mount('#app');

// 添加CSS动画
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    
    @keyframes slideOut {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(100%);
            opacity: 0;
        }
    }
`;
document.head.appendChild(style);