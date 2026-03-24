package mcp

func ToolSchemas() []map[string]interface{} {
	return []map[string]interface{}{
		{"name":"list","description":"List all tasks","inputSchema":map[string]interface{}{"type":"object","properties":map[string]interface{}{}}},
		{"name":"read","description":"Read a task by ID or resource URI","inputSchema":map[string]interface{}{"type":"object","properties":map[string]interface{}{"id":map[string]string{"type":"string"},"uri":map[string]string{"type":"string"}}}},
		{"name":"search","description":"Search tasks","inputSchema":map[string]interface{}{"type":"object","properties":map[string]interface{}{"query":map[string]string{"type":"string"}},"required":[]string{"query"}}},
		{"name":"create","description":"Create task","inputSchema":map[string]interface{}{"type":"object","properties":map[string]interface{}{"title":map[string]string{"type":"string"},"notes":map[string]string{"type":"string"},"due":map[string]string{"type":"string"},"recurrence":map[string]string{"type":"string"}},"required":[]string{"title"}}},
		{"name":"update","description":"Update a task in the configured task list","inputSchema":map[string]interface{}{"type":"object","properties":map[string]interface{}{"id":map[string]string{"type":"string"},"uri":map[string]string{"type":"string"},"title":map[string]string{"type":"string"},"notes":map[string]string{"type":"string"},"status":map[string]string{"type":"string"},"due":map[string]string{"type":"string"},"recurrence":map[string]string{"type":"string"}}}},
		{"name":"delete","description":"Delete a task by ID or resource URI","inputSchema":map[string]interface{}{"type":"object","properties":map[string]interface{}{"id":map[string]string{"type":"string"},"uri":map[string]string{"type":"string"}}}},
		{"name":"clear","description":"Clear all completed tasks from the configured task list","inputSchema":map[string]interface{}{"type":"object","properties":map[string]interface{}{}}},
	}
}
