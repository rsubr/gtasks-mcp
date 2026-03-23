package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"gtasks-mcp/internal/logging"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func MustGetClient(tokenFile string) *http.Client {
	logging.Info("loading oauth credentials", "credentials_file", "gcp-oauth.keys.json", "token_file", tokenFile)
	b, err := os.ReadFile("gcp-oauth.keys.json")
	if err != nil { log.Fatal(err) }

	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/tasks")
	if err != nil { log.Fatal(err) }

	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		logging.Warn("oauth token file unavailable, starting interactive flow", "token_file", tokenFile, "error", err)
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	} else {
		logging.Info("loaded oauth token from file", "token_file", tokenFile)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	logging.Info("awaiting oauth authorization code from stdin")
	fmt.Println("Open URL:", authURL)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	code := scanner.Text()

	tok, err := config.Exchange(context.Background(), code)
	if err != nil { log.Fatal(err) }
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil { return nil, err }
	defer f.Close()

	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	logging.Info("saving oauth token", "token_file", path)
	f, _ := os.Create(path)
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
