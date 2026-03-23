package main

import (
	"flag"
	"log"
	"os"

	"gtasks-mcp/internal/auth"
	"gtasks-mcp/internal/logging"
	"gtasks-mcp/internal/mcp"
	"gtasks-mcp/internal/tasks"
)

func main() {
	tokenFile := flag.String("token-file", "token.json", "OAuth token file")
	tasklist := flag.String("tasklist", "My Tasks", "Task list name")
	logLevel := flag.String("log-level", "info", "debug|info|warn|error")
	flag.Parse()

	logging.Init(*logLevel)

	client := auth.MustGetClient(*tokenFile)
	svc := tasks.New(client, *tasklist)
	server := mcp.NewServer(svc)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Starting MCP server on :" + port)
	log.Fatal(server.Start(":" + port))
}
