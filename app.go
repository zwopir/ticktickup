package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/zwopir/ticktickup/ticktick"
)

type App struct {
	ctx           context.Context
	client        *ticktick.Client
	oauthServer   *http.Server
	oauthListener net.Listener
	authCodeChan  chan string
	mu            sync.Mutex
}

type ProjectInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ImportResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Imported  int    `json:"imported"`
	Failed    int    `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

type TaskInput struct {
	Title     string         `json:"title"`
	Content   string         `json:"content,omitempty"`
	DueDate   string         `json:"dueDate,omitempty"`
	Priority  int            `json:"priority,omitempty"`
	Tags      []string       `json:"tags,omitempty"`
	Subtasks  []SubtaskInput `json:"subtasks,omitempty"`
}

type SubtaskInput struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	StartDate   string `json:"startDate,omitempty"`
	DueDate     string `json:"dueDate,omitempty"`
}

func NewApp() *App {
	return &App{
		authCodeChan: make(chan string, 1),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.initClient()
}

func (a *App) initClient() error {
	config, err := ticktick.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if config == nil {
		return nil
	}

	a.client = ticktick.NewClient(config.ClientID, config.ClientSecret, config.RedirectURI)

	token, err := ticktick.LoadToken()
	if err != nil {
		return fmt.Errorf("loading token: %w", err)
	}
	if token != nil {
		a.client.SetToken(token)
	}

	return nil
}

func (a *App) IsConfigured() bool {
	config, _ := ticktick.LoadConfig()
	return config != nil && config.ClientID != "" && config.ClientSecret != ""
}

func (a *App) SaveConfig(clientID, clientSecret string) error {
	config := &ticktick.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  ticktick.DefaultRedirect,
	}
	if err := ticktick.SaveConfig(config); err != nil {
		return err
	}
	a.client = ticktick.NewClient(clientID, clientSecret, ticktick.DefaultRedirect)
	return nil
}

func (a *App) IsAuthenticated() bool {
	if a.client == nil {
		return false
	}
	return a.client.IsAuthenticated() && !a.client.IsTokenExpired()
}

func (a *App) NeedsAuth() bool {
	if a.client == nil {
		return true
	}
	if !a.client.IsAuthenticated() {
		return true
	}
	if a.client.IsTokenExpired() {
		// Try to refresh token
		_, err := a.client.RefreshToken()
		if err != nil {
			return true
		}
		// Save refreshed token
		if err := ticktick.SaveToken(a.client.GetToken()); err != nil {
			fmt.Printf("Warning: failed to save refreshed token: %v\n", err)
		}
		return false
	}
	return false
}

func (a *App) StartAuth() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client == nil {
		return "", fmt.Errorf("client not configured")
	}

	// Start local OAuth callback server
	listener, err := net.Listen("tcp", ":8765")
	if err != nil {
		return "", fmt.Errorf("failed to start callback server: %w", err)
	}
	a.oauthListener = listener

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", a.handleOAuthCallback)

	a.oauthServer = &http.Server{Handler: mux}
	go func() {
		if err := a.oauthServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("OAuth server error: %v\n", err)
		}
	}()

	// Clear any stale auth codes
	select {
	case <-a.authCodeChan:
	default:
	}

	authURL := a.client.GetAuthURL("")
	runtime.BrowserOpenURL(a.ctx, authURL)

	return authURL, nil
}

func (a *App) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	errorMsg := r.URL.Query().Get("error")

	if errorMsg != "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<!DOCTYPE html><html><body>
			<h1>Authentication Failed</h1>
			<p>Error: %s</p>
			<p>You can close this window.</p>
		</body></html>`, errorMsg)
		return
	}

	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<!DOCTYPE html><html><body>
			<h1>Authentication Failed</h1>
			<p>No authorization code received.</p>
			<p>You can close this window.</p>
		</body></html>`)
		return
	}

	// Send code to channel
	select {
	case a.authCodeChan <- code:
	default:
	}

	fmt.Fprintf(w, `<!DOCTYPE html><html><body>
		<h1>Authentication Successful!</h1>
		<p>You can close this window and return to the app.</p>
		<script>setTimeout(function() { window.close(); }, 2000);</script>
	</body></html>`)
}

func (a *App) WaitForAuthCode() error {
	code := <-a.authCodeChan

	// Shutdown the OAuth server
	if a.oauthServer != nil {
		a.oauthServer.Close()
	}
	if a.oauthListener != nil {
		a.oauthListener.Close()
	}

	// Exchange code for token
	token, err := a.client.ExchangeCode(code)
	if err != nil {
		return fmt.Errorf("exchanging code: %w", err)
	}

	// Save token
	if err := ticktick.SaveToken(token); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}

	return nil
}

func (a *App) Logout() error {
	if err := ticktick.DeleteToken(); err != nil {
		return err
	}
	if a.client != nil {
		a.client.SetToken(nil)
	}
	return nil
}

func (a *App) GetProjects() ([]ProjectInfo, error) {
	if a.client == nil {
		return nil, fmt.Errorf("client not configured")
	}

	projects, err := a.client.GetProjects()
	if err != nil {
		return nil, err
	}

	result := make([]ProjectInfo, 0, len(projects))
	for _, p := range projects {
		if !p.Closed {
			result = append(result, ProjectInfo{
				ID:   p.ID,
				Name: p.Name,
			})
		}
	}

	return result, nil
}

func (a *App) CreateProject(name string) (*ProjectInfo, error) {
	if a.client == nil {
		return nil, fmt.Errorf("client not configured")
	}

	if name == "" {
		return nil, fmt.Errorf("project name is required")
	}

	project, err := a.client.CreateProject(name)
	if err != nil {
		return nil, err
	}

	return &ProjectInfo{
		ID:   project.ID,
		Name: project.Name,
	}, nil
}

func (a *App) ImportTasks(projectID string, fileContent string, fileType string) ImportResult {
	if a.client == nil {
		return ImportResult{Success: false, Message: "Client not configured"}
	}

	var tasks []TaskInput
	var err error

	switch strings.ToLower(fileType) {
	case "csv":
		tasks, err = parseCSV(fileContent)
	case "json":
		tasks, err = parseJSON(fileContent)
	default:
		return ImportResult{Success: false, Message: fmt.Sprintf("Unsupported file type: %s", fileType)}
	}

	if err != nil {
		return ImportResult{Success: false, Message: fmt.Sprintf("Failed to parse file: %v", err)}
	}

	if len(tasks) == 0 {
		return ImportResult{Success: false, Message: "No tasks found in file"}
	}

	var imported, failed int
	var errors []string

	for _, task := range tasks {
		// Convert subtasks to CheckItems
		var items []ticktick.CheckItem
		for _, subtask := range task.Subtasks {
			item := ticktick.CheckItem{
				Title:     subtask.Title,
				StartDate: ticktick.FlexibleDate(subtask.StartDate),
				DueDate:   ticktick.FlexibleDate(subtask.DueDate),
			}
			items = append(items, item)
		}

		tickTask := &ticktick.Task{
			ProjectID: projectID,
			Title:     task.Title,
			Content:   task.Content,
			DueDate:   task.DueDate,
			Priority:  task.Priority,
			Tags:      task.Tags,
			Items:     items,
		}

		_, err := a.client.CreateTask(tickTask)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("Failed to create '%s': %v", task.Title, err))
		} else {
			imported++
		}
	}

	result := ImportResult{
		Success:  failed == 0,
		Imported: imported,
		Failed:   failed,
		Errors:   errors,
	}

	if failed == 0 {
		result.Message = fmt.Sprintf("Successfully imported %d tasks", imported)
	} else if imported > 0 {
		result.Message = fmt.Sprintf("Imported %d tasks, %d failed", imported, failed)
	} else {
		result.Message = "All tasks failed to import"
	}

	return result
}

func parseCSV(content string) ([]TaskInput, error) {
	reader := csv.NewReader(strings.NewReader(content))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty CSV file")
	}

	// Find column indices from header
	header := records[0]
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Check for required title column
	titleIdx, hasTitleIdx := colIndex["title"]
	if !hasTitleIdx {
		titleIdx, hasTitleIdx = colIndex["name"]
	}
	if !hasTitleIdx {
		titleIdx, hasTitleIdx = colIndex["task"]
	}
	if !hasTitleIdx {
		// Assume first column is title if no header match
		titleIdx = 0
	}

	tasks := make([]TaskInput, 0, len(records)-1)
	for _, record := range records[1:] {
		if len(record) <= titleIdx || strings.TrimSpace(record[titleIdx]) == "" {
			continue
		}

		task := TaskInput{
			Title: strings.TrimSpace(record[titleIdx]),
		}

		// Try to get optional fields
		if idx, ok := colIndex["content"]; ok && idx < len(record) {
			task.Content = strings.TrimSpace(record[idx])
		}
		if idx, ok := colIndex["description"]; ok && idx < len(record) && task.Content == "" {
			task.Content = strings.TrimSpace(record[idx])
		}
		if idx, ok := colIndex["duedate"]; ok && idx < len(record) {
			task.DueDate = strings.TrimSpace(record[idx])
		}
		if idx, ok := colIndex["due_date"]; ok && idx < len(record) && task.DueDate == "" {
			task.DueDate = strings.TrimSpace(record[idx])
		}
		if idx, ok := colIndex["due"]; ok && idx < len(record) && task.DueDate == "" {
			task.DueDate = strings.TrimSpace(record[idx])
		}
		if idx, ok := colIndex["tags"]; ok && idx < len(record) {
			tagStr := strings.TrimSpace(record[idx])
			if tagStr != "" {
				task.Tags = strings.Split(tagStr, ",")
				for i, tag := range task.Tags {
					task.Tags[i] = strings.TrimSpace(tag)
				}
			}
		}

		// Parse subtasks (semicolon-separated, each subtask can have pipe-separated fields: title|startDate|dueDate)
		if idx, ok := colIndex["subtasks"]; ok && idx < len(record) {
			subtasksStr := strings.TrimSpace(record[idx])
			if subtasksStr != "" {
				subtaskEntries := strings.Split(subtasksStr, ";")
				for _, entry := range subtaskEntries {
					entry = strings.TrimSpace(entry)
					if entry == "" {
						continue
					}
					parts := strings.Split(entry, "|")
					subtask := SubtaskInput{
						Title: strings.TrimSpace(parts[0]),
					}
					if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
						subtask.StartDate = strings.TrimSpace(parts[1])
					}
					if len(parts) > 2 && strings.TrimSpace(parts[2]) != "" {
						subtask.DueDate = strings.TrimSpace(parts[2])
					}
					if subtask.Title != "" {
						task.Subtasks = append(task.Subtasks, subtask)
					}
				}
			}
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func parseJSON(content string) ([]TaskInput, error) {
	var tasks []TaskInput

	// Try parsing as array first
	if err := json.Unmarshal([]byte(content), &tasks); err == nil {
		return tasks, nil
	}

	// Try parsing as object with tasks array
	var wrapper struct {
		Tasks []TaskInput `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(content), &wrapper); err == nil && len(wrapper.Tasks) > 0 {
		return wrapper.Tasks, nil
	}

	// Try parsing as single task
	var single TaskInput
	if err := json.Unmarshal([]byte(content), &single); err == nil && single.Title != "" {
		return []TaskInput{single}, nil
	}

	return nil, fmt.Errorf("could not parse JSON as task array or object")
}
