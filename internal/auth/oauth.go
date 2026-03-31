package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gtasks-mcp/internal/logging"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const DefaultCredentialsFilename = "gcp-oauth.keys.json"

func MustGetClient(credentialsFile, tokenFile string) *http.Client {
	client, err := GetClient(credentialsFile, tokenFile)
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func GetClient(credentialsFile, tokenFile string) (*http.Client, error) {
	logging.Info("loading oauth credentials", "credentials_file", credentialsFile, "token_file", tokenFile)
	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/tasks")
	if err != nil {
		return nil, err
	}

	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		logging.Warn("oauth token file unavailable, starting interactive flow", "token_file", tokenFile, "error", err)
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	} else {
		logging.Info("loaded oauth token from file", "token_file", tokenFile)
	}

	ts := &persistingTokenSource{
		src:  config.TokenSource(context.Background(), tok),
		path: tokenFile,
	}
	return oauth2.NewClient(context.Background(), ts), nil
}

// persistingTokenSource wraps an oauth2.TokenSource and writes the token back
// to disk whenever it changes, so refreshed tokens (including rotated refresh
// tokens) survive server restarts.
type persistingTokenSource struct {
	mu   sync.Mutex
	src  oauth2.TokenSource
	path string
	last string // last access token seen — used to detect a refresh
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := p.src.Token()
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if tok.AccessToken != p.last {
		p.last = tok.AccessToken
		saveToken(p.path, tok)
		logging.Info("oauth token refreshed and persisted", "token_file", p.path)
	}
	return tok, nil
}

func ResolveCredentialsFile(configuredPath string) (string, error) {
	candidates := []string{}
	if configuredPath != "" {
		candidates = append(candidates, configuredPath)
	} else {
		candidates = append(candidates,
			filepath.Join("/auth", DefaultCredentialsFilename),
			DefaultCredentialsFilename,
		)
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}

	if configuredPath != "" {
		return "", fmt.Errorf("oauth credentials file not found: %s", configuredPath)
	}
	return "", fmt.Errorf("oauth credentials file not found; checked /auth/%s and ./%s", DefaultCredentialsFilename, DefaultCredentialsFilename)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	logging.Info("awaiting oauth authorization code from stdin")
	fmt.Println("Open URL:", authURL)
	fmt.Println("Paste the full redirect URL or just the authorization code:")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	code := extractAuthorizationCode(scanner.Text())

	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Fatal(err)
	}
	return tok
}

func extractAuthorizationCode(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	parsed, err := url.Parse(input)
	if err != nil {
		return input
	}

	code := strings.TrimSpace(parsed.Query().Get("code"))
	if code != "" {
		return code
	}
	return input
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	f, err := os.Create(path)
	if err != nil {
		logging.Warn("failed to save oauth token", "token_file", path, "error", err)
		return
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(token); err != nil {
		logging.Warn("failed to encode oauth token", "token_file", path, "error", err)
	}
}
