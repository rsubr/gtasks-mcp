package mcp

func ToolSchemas() []map[string]interface{} {
	return []map[string]interface{}{
		{"name":"list","description":"List all tasks","inputSchema":map[string]interface{}{"type":"object","properties":map[string]interface{}{}}},
		{"name":"search","description":"Search tasks","inputSchema":map[string]interface{}{"type":"object","properties":map[string]interface{}{"query":map[string]string{"type":"string"}},"required":[]string{"query"}}},
		{"name":"create","description":"Create task","inputSchema":map[string]interface{}{"type":"object","properties":map[string]interface{}{"title":map[string]string{"type":"string"},"notes":map[string]string{"type":"string"},"due":map[string]string{"type":"string"}},"required":[]string{"title"}}},
	}
}
