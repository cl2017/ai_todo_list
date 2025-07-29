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
            // MCP Integration
            mcpClient: null,
            mcpConnected: false,
            aiRecommendations: [],
            todaysSuggestions: [],
            showAIInsights: false,
            aiInsights: null,
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
        // MCP Integration Methods
        async initializeMCP() {
            try {
                const wsUrl = `ws://${window.location.host}/mcp`;
                this.mcpClient = new MCPClient(wsUrl);
                
                this.mcpClient.on('connected', () => {
                    this.mcpConnected = true;
                    this.showNotification('AI助手已连接', 'success');
                    this.loadAIRecommendations();
                    this.loadTodaysSuggestions();
                });
                
                this.mcpClient.on('disconnected', () => {
                    this.mcpConnected = false;
                    this.showNotification('AI助手连接断开', 'error');
                });
                
                this.mcpClient.on('notification', (notification) => {
                    this.handleMCPNotification(notification);
                });
                
                await this.mcpClient.connect();
            } catch (error) {
                console.error('MCP连接失败:', error);
                this.showNotification('AI助手连接失败，使用基础功能', 'error');
                // Fallback to HTTP API
                this.fetchTodos();
            }
        },

        handleMCPNotification(notification) {
            // Handle real-time updates from MCP server
            switch (notification.method) {
                case 'todos/created':
                    this.todos.push(notification.params);
                    this.showNotification('任务已创建', 'success');
                    break;
                case 'todos/updated':
                    const updatedIndex = this.todos.findIndex(t => t.id === notification.params.id);
                    if (updatedIndex !== -1) {
                        this.todos[updatedIndex] = notification.params;
                        this.showNotification('任务已更新', 'success');
                    }
                    break;
                case 'todos/deleted':
                    this.todos = this.todos.filter(t => t.id !== notification.params.id);
                    this.showNotification('任务已删除', 'success');
                    break;
            }
        },

        async fetchTodos() {
            try {
                if (this.mcpConnected && this.mcpClient) {
                    // Use MCP to fetch todos
                    const result = await this.mcpClient.listTodos();
                    // Note: MCP returns analysis text, we still need HTTP API for actual data
                    const response = await axios.get('/api/todos');
                    this.todos = response.data || [];
                } else {
                    // Fallback to HTTP API
                    const response = await axios.get('/api/todos');
                    this.todos = response.data || [];
                }
            } catch (error) {
                console.error('获取任务失败:', error);
                this.showNotification('获取任务失败', 'error');
            }
        },

        // AI Intelligence Methods
        async loadAIRecommendations() {
            if (!this.mcpConnected || !this.mcpClient) return;
            
            try {
                this.isLoadingAI = true;
                this.aiRecommendations = await this.mcpClient.getAIRecommendations();
            } catch (error) {
                console.error('获取AI推荐失败:', error);
                this.aiRecommendations = ['AI推荐暂时不可用'];
            } finally {
                this.isLoadingAI = false;
            }
        },

        async loadTodaysSuggestions() {
            if (!this.mcpConnected || !this.mcpClient) return;
            
            try {
                this.todaysSuggestions = await this.mcpClient.getTodaysSuggestions();
            } catch (error) {
                console.error('获取今日建议失败:', error);
                this.todaysSuggestions = ['今日建议暂时不可用'];
            }
        },

        async addTodo() {
            try {
                const todoData = { ...this.newTodo };
                if (todoData.due_date) {
                    todoData.due_date = new Date(todoData.due_date).toISOString();
                }
                
                let response;
                if (this.mcpConnected && this.mcpClient) {
                    // Use smart todo creation with AI assistance
                    const smartResult = await this.mcpClient.createSmartTodo(todoData.title, todoData.description);
                    this.aiInsights = smartResult.aiInsights;
                    this.showAIInsights = true;
                    
                    // Still use HTTP API for actual creation to maintain consistency
                    response = await axios.post('/api/todos', {
                        ...todoData,
                        priority: smartResult.aiInsights.detectedPriority,
                        category: smartResult.aiInsights.detectedCategory,
                        estimated_duration: smartResult.aiInsights.estimatedDuration
                    });
                } else {
                    response = await axios.post('/api/todos', todoData);
                }
                
                this.todos.push(response.data);
                this.resetNewTodo();
                this.showAddForm = false;
                this.showNotification('任务添加成功', 'success');
                this.getAIAnalysis(); // 刷新AI分析
                this.loadAIRecommendations(); // 刷新AI推荐
            } catch (error) {
                console.error('添加任务失败:', error);
                this.showNotification('添加任务失败', 'error');
            }
        },

        async breakDownTask(todo) {
            if (!this.mcpConnected || !this.mcpClient) {
                this.showNotification('AI任务分解功能需要连接AI助手', 'error');
                return;
            }

            try {
                this.isLoadingAI = true;
                const breakdown = await this.mcpClient.breakDownTask(todo.id, 'medium');
                
                if (breakdown.content && breakdown.content[0]) {
                    // Show breakdown in a modal or notification
                    const breakdownText = breakdown.content[0].text;
                    this.showTaskBreakdown(todo.title, breakdownText);
                }
            } catch (error) {
                console.error('任务分解失败:', error);
                this.showNotification('任务分解失败', 'error');
            } finally {
                this.isLoadingAI = false;
            }
        },

        showTaskBreakdown(taskTitle, breakdown) {
            // Create a modal to show task breakdown
            const modal = document.createElement('div');
            modal.className = 'modal-overlay';
            modal.innerHTML = `
                <div class="modal" style="max-width: 600px;">
                    <h3>任务分解: ${taskTitle}</h3>
                    <pre style="white-space: pre-wrap; font-family: inherit; margin: 20px 0;">${breakdown}</pre>
                    <div class="form-buttons">
                        <button onclick="this.closest('.modal-overlay').remove()">关闭</button>
                    </div>
                </div>
            `;
            document.body.appendChild(modal);
        },

        async optimizeMySchedule() {
            if (!this.mcpConnected || !this.mcpClient) {
                this.showNotification('AI日程优化功能需要连接AI助手', 'error');
                return;
            }

            try {
                this.isLoadingAI = true;
                const optimization = await this.mcpClient.optimizeSchedule('today', 8);
                
                if (optimization.content && optimization.content[0]) {
                    const optimizationText = optimization.content[0].text;
                    this.showScheduleOptimization(optimizationText);
                }
            } catch (error) {
                console.error('日程优化失败:', error);
                this.showNotification('日程优化失败', 'error');
            } finally {
                this.isLoadingAI = false;
            }
        },

        showScheduleOptimization(optimization) {
            const modal = document.createElement('div');
            modal.className = 'modal-overlay';
            modal.innerHTML = `
                <div class="modal" style="max-width: 600px;">
                    <h3>AI日程优化建议</h3>
                    <pre style="white-space: pre-wrap; font-family: inherit; margin: 20px 0;">${optimization}</pre>
                    <div class="form-buttons">
                        <button onclick="this.closest('.modal-overlay').remove()">关闭</button>
                    </div>
                </div>
            `;
            document.body.appendChild(modal);
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
        // Initialize MCP connection first
        await this.initializeMCP();
        
        // Fallback to HTTP API if MCP fails
        if (!this.mcpConnected) {
            await this.fetchTodos();
            await this.getAIAnalysis();
        } else {
            await this.fetchTodos();
        }
        
        // 定期刷新AI分析和推荐
        setInterval(() => {
            if (this.mcpConnected) {
                this.loadAIRecommendations();
                this.loadTodaysSuggestions();
            }
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