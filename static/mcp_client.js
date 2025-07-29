// MCP Client for AI Todo Assistant
class MCPClient {
    constructor(url) {
        this.url = url;
        this.ws = null;
        this.requestId = 1;
        this.pendingRequests = new Map();
        this.tools = [];
        this.connected = false;
        this.eventHandlers = {
            'connected': [],
            'disconnected': [],
            'notification': [],
            'error': []
        };
    }

    // Event handling
    on(event, handler) {
        if (this.eventHandlers[event]) {
            this.eventHandlers[event].push(handler);
        }
    }

    emit(event, data) {
        if (this.eventHandlers[event]) {
            this.eventHandlers[event].forEach(handler => handler(data));
        }
    }

    // Connect to MCP server
    async connect() {
        return new Promise((resolve, reject) => {
            try {
                this.ws = new WebSocket(this.url);
                
                this.ws.onopen = async () => {
                    console.log('MCP WebSocket connected');
                    this.connected = true;
                    
                    // Initialize the connection
                    await this.initialize();
                    
                    // Get available tools
                    await this.getTools();
                    
                    this.emit('connected');
                    resolve();
                };

                this.ws.onmessage = (event) => {
                    this.handleMessage(JSON.parse(event.data));
                };

                this.ws.onclose = () => {
                    console.log('MCP WebSocket disconnected');
                    this.connected = false;
                    this.emit('disconnected');
                };

                this.ws.onerror = (error) => {
                    console.error('MCP WebSocket error:', error);
                    this.emit('error', error);
                    reject(error);
                };
            } catch (error) {
                reject(error);
            }
        });
    }

    // Disconnect from MCP server
    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
            this.connected = false;
        }
    }

    // Handle incoming messages
    handleMessage(message) {
        if (message.id && this.pendingRequests.has(message.id)) {
            // This is a response to a request
            const { resolve, reject } = this.pendingRequests.get(message.id);
            this.pendingRequests.delete(message.id);

            if (message.error) {
                reject(new Error(message.error.message));
            } else {
                resolve(message.result);
            }
        } else if (message.method) {
            // This is a notification
            this.emit('notification', message);
        }
    }

    // Send a request and wait for response
    async sendRequest(method, params = null) {
        if (!this.connected) {
            throw new Error('MCP client not connected');
        }

        const id = this.requestId++;
        const request = {
            jsonrpc: '2.0',
            id: id,
            method: method,
            ...(params && { params })
        };

        return new Promise((resolve, reject) => {
            this.pendingRequests.set(id, { resolve, reject });
            this.ws.send(JSON.stringify(request));

            // Set timeout for request
            setTimeout(() => {
                if (this.pendingRequests.has(id)) {
                    this.pendingRequests.delete(id);
                    reject(new Error('Request timeout'));
                }
            }, 30000); // 30 second timeout
        });
    }

    // Initialize the MCP connection
    async initialize() {
        const result = await this.sendRequest('initialize', {
            protocolVersion: '2024-11-05',
            capabilities: {
                tools: {}
            },
            clientInfo: {
                name: 'AI Todo Assistant Web Client',
                version: '1.0.0'
            }
        });
        
        console.log('MCP initialized:', result);
        return result;
    }

    // Get available tools from server
    async getTools() {
        const result = await this.sendRequest('tools/list');
        this.tools = result.tools || [];
        console.log('Available MCP tools:', this.tools);
        return this.tools;
    }

    // Call a tool
    async callTool(name, args = {}) {
        const result = await this.sendRequest('tools/call', {
            name: name,
            arguments: args
        });
        return result;
    }

    // Ping the server
    async ping() {
        return await this.sendRequest('ping');
    }

    // Tool-specific methods for easier use

    // List todos with optional filtering
    async listTodos(filters = {}) {
        return await this.callTool('list_todos', filters);
    }

    // Create a new todo
    async createTodo(todoData) {
        return await this.callTool('create_todo', todoData);
    }

    // Update an existing todo
    async updateTodo(id, updates) {
        return await this.callTool('update_todo', { id, ...updates });
    }

    // Delete a todo
    async deleteTodo(id) {
        return await this.callTool('delete_todo', { id });
    }

    // Analyze tasks
    async analyzeTasks(analysisType = 'priority') {
        return await this.callTool('analyze_tasks', { analysis_type: analysisType });
    }

    // Optimize schedule
    async optimizeSchedule(timeHorizon = 'today', workHours = 8) {
        return await this.callTool('optimize_schedule', { 
            time_horizon: timeHorizon, 
            work_hours: workHours 
        });
    }

    // Break down a task
    async breakDownTask(taskId, complexity = 'medium') {
        return await this.callTool('break_down_task', { 
            task_id: taskId, 
            complexity: complexity 
        });
    }

    // AI Assistant methods for intelligent task management

    // Get AI recommendations for the user
    async getAIRecommendations() {
        try {
            const [priorityAnalysis, overdueAnalysis, workloadAnalysis] = await Promise.all([
                this.analyzeTasks('priority'),
                this.analyzeTasks('overdue'),
                this.analyzeTasks('workload')
            ]);

            const recommendations = [];
            
            // Parse analysis results and generate recommendations
            if (priorityAnalysis.content && priorityAnalysis.content[0]) {
                const priorityText = priorityAnalysis.content[0].text;
                if (priorityText.includes('Urgent: 0')) {
                    recommendations.push('太好了！没有紧急任务，可以专注于重要但不紧急的工作');
                } else {
                    recommendations.push('有紧急任务需要立即处理，建议优先完成');
                }
            }

            if (overdueAnalysis.content && overdueAnalysis.content[0]) {
                const overdueText = overdueAnalysis.content[0].text;
                const overdueMatch = overdueText.match(/(\d+) tasks are overdue/);
                if (overdueMatch && parseInt(overdueMatch[1]) > 0) {
                    recommendations.push(`有 ${overdueMatch[1]} 个任务已逾期，建议重新评估优先级`);
                }
            }

            if (workloadAnalysis.content && workloadAnalysis.content[0]) {
                const workloadText = workloadAnalysis.content[0].text;
                const pendingMatch = workloadText.match(/Pending: (\d+)/);
                if (pendingMatch && parseInt(pendingMatch[1]) > 10) {
                    recommendations.push('待处理任务较多，建议将大任务分解为小任务');
                }
            }

            return recommendations;
        } catch (error) {
            console.error('Failed to get AI recommendations:', error);
            return ['AI分析暂时不可用，请稍后重试'];
        }
    }

    // Smart task creation with AI assistance
    async createSmartTodo(title, description = '') {
        try {
            // Analyze the task title to determine priority and category
            let priority = 'medium';
            let category = 'personal';
            let estimatedDuration = '1 hour';

            // Simple AI logic for task classification
            const titleLower = title.toLowerCase();
            const descLower = description.toLowerCase();

            // Priority detection
            if (titleLower.includes('urgent') || titleLower.includes('asap') || 
                titleLower.includes('emergency') || titleLower.includes('deadline')) {
                priority = 'urgent';
            } else if (titleLower.includes('important') || titleLower.includes('critical') ||
                      titleLower.includes('meeting') || titleLower.includes('presentation')) {
                priority = 'high';
            } else if (titleLower.includes('someday') || titleLower.includes('maybe') ||
                      titleLower.includes('consider')) {
                priority = 'low';
            }

            // Category detection
            if (titleLower.includes('work') || titleLower.includes('office') || 
                titleLower.includes('meeting') || titleLower.includes('project') ||
                titleLower.includes('client') || titleLower.includes('presentation')) {
                category = 'work';
            } else if (titleLower.includes('health') || titleLower.includes('doctor') ||
                      titleLower.includes('exercise') || titleLower.includes('gym')) {
                category = 'health';
            } else if (titleLower.includes('family') || titleLower.includes('mom') ||
                      titleLower.includes('dad') || titleLower.includes('parent')) {
                category = 'family';
            } else if (titleLower.includes('money') || titleLower.includes('tax') ||
                      titleLower.includes('bank') || titleLower.includes('investment')) {
                category = 'financial';
            } else if (titleLower.includes('clean') || titleLower.includes('organize') ||
                      titleLower.includes('fix') || titleLower.includes('repair')) {
                category = 'household';
            }

            // Duration estimation
            if (titleLower.includes('quick') || titleLower.includes('call') ||
                titleLower.includes('email') || titleLower.includes('text')) {
                estimatedDuration = '15 minutes';
            } else if (titleLower.includes('research') || titleLower.includes('plan') ||
                      titleLower.includes('organize') || titleLower.includes('prepare')) {
                estimatedDuration = '2 hours';
            } else if (titleLower.includes('project') || titleLower.includes('presentation') ||
                      titleLower.includes('report') || titleLower.includes('analysis')) {
                estimatedDuration = '4 hours';
            }

            const todoData = {
                title,
                description,
                priority,
                category,
                estimated_duration: estimatedDuration
            };

            const result = await this.createTodo(todoData);
            
            // Return both the result and AI insights
            return {
                todo: result,
                aiInsights: {
                    detectedPriority: priority,
                    detectedCategory: category,
                    estimatedDuration: estimatedDuration,
                    reasoning: `AI分析：基于任务标题和描述，自动设置为${priority}优先级，${category}分类，预计用时${estimatedDuration}`
                }
            };
        } catch (error) {
            console.error('Failed to create smart todo:', error);
            throw error;
        }
    }

    // Get intelligent task suggestions for today
    async getTodaysSuggestions() {
        try {
            const optimization = await this.optimizeSchedule('today', 8);
            
            if (optimization.content && optimization.content[0]) {
                const suggestions = optimization.content[0].text.split('\n').filter(line => 
                    line.trim() && !line.includes('Schedule Optimization') && !line.includes('Found')
                );
                return suggestions;
            }
            
            return ['今天是新的一天，从最重要的任务开始吧！'];
        } catch (error) {
            console.error('Failed to get today\'s suggestions:', error);
            return ['AI建议暂时不可用，请稍后重试'];
        }
    }
}

// Export for use in other modules
window.MCPClient = MCPClient;