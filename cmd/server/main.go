package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"gtasks-mcp/internal/auth"
	"gtasks-mcp/internal/logging"
	"gtasks-mcp/internal/mcp"
	"gtasks-mcp/internal/tasks"
)

func main() {
	credentialsFile := flag.String("credentials-file", "", "OAuth client credentials JSON file")
	tokenFile := flag.String("token-file", "token.json", "OAuth token file")
	tasklist := flag.String("tasklist", "", "Task list name")
	logLevel := flag.String("log-level", "", "debug|info|warn|error")
	listenAddr := flag.String("listen-addr", "", "Server listen address, for example 0.0.0.0:8080")
	portFlag := flag.String("port", "", "Server port")
	flag.Parse()

	resolvedCredentialsFile := *credentialsFile
	if resolvedCredentialsFile == "" {
		resolvedCredentialsFile = os.Getenv("GOOGLE_OAUTH_CREDENTIALS_FILE")
	}
	if resolvedPath, err := auth.ResolveCredentialsFile(resolvedCredentialsFile); err != nil {
		logging.Init("info")
		logging.Warn("oauth credentials file unavailable at startup", "error", err)
		if resolvedCredentialsFile == "" {
			resolvedCredentialsFile = auth.DefaultCredentialsFilename
		}
	} else {
		resolvedCredentialsFile = resolvedPath
	}

	resolvedTaskList := *tasklist
	if resolvedTaskList == "" {
		resolvedTaskList = os.Getenv("TASKLIST_NAME")
	}
	if resolvedTaskList == "" {
		resolvedTaskList = "My Tasks"
	}

	resolvedLogLevel := *logLevel
	if resolvedLogLevel == "" {
		resolvedLogLevel = os.Getenv("LOG_LEVEL")
	}
	if resolvedLogLevel == "" {
		resolvedLogLevel = "info"
	}

	logging.Init(resolvedLogLevel)
	logging.Info("resolved startup configuration", "task_list", resolvedTaskList, "log_level", resolvedLogLevel, "credentials_file", resolvedCredentialsFile, "token_file", *tokenFile)

	var svc *tasks.Service
	client, err := auth.GetClient(resolvedCredentialsFile, *tokenFile)
	if err != nil {
		logging.Warn("starting with unavailable google tasks backend", "error", err)
		svc = tasks.NewUnavailable(resolvedTaskList, err)
	} else {
		svc, err = tasks.New(client, resolvedTaskList)
		if err != nil {
			logging.Warn("starting with unavailable google tasks backend", "error", err)
			svc = tasks.NewUnavailable(resolvedTaskList, err)
		}
	}
	server := mcp.NewServer(svc)

	resolvedListenAddr := *listenAddr
	if resolvedListenAddr == "" {
		resolvedListenAddr = os.Getenv("LISTEN_ADDR")
	}

	resolvedPort := *portFlag
	if resolvedPort == "" {
		resolvedPort = os.Getenv("PORT")
	}
	if resolvedPort == "" {
		resolvedPort = "8080"
	}
	if _, err := strconv.Atoi(resolvedPort); err != nil {
		log.Fatal(err)
	}

	if resolvedListenAddr == "" {
		resolvedListenAddr = ":" + resolvedPort
	}

	logging.Info("launching mcp server", "listen_addr", resolvedListenAddr)
	log.Println("Starting MCP server on " + resolvedListenAddr)
	log.Fatal(server.Start(resolvedListenAddr))
}
