package ticktick

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	BaseURL          = "https://api.ticktick.com/open/v1"
	AuthURL          = "https://ticktick.com/oauth/authorize"
	TokenURL         = "https://ticktick.com/oauth/token"
	DefaultScopes    = "tasks:read tasks:write"
	DefaultRedirect  = "http://localhost:8765/callback"
)

type Client struct {
	httpClient   *http.Client
	clientID     string
	clientSecret string
	redirectURI  string
	token        *Token
}

type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type Project struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color,omitempty"`
	SortOrder  int64  `json:"sortOrder,omitempty"`
	Closed     bool   `json:"closed,omitempty"`
	GroupID    string `json:"groupId,omitempty"`
	ViewMode   string `json:"viewMode,omitempty"`
	Permission string `json:"permission,omitempty"`
	Kind       string `json:"kind,omitempty"`
}

type Task struct {
	ID          string     `json:"id,omitempty"`
	ProjectID   string     `json:"projectId"`
	Title       string     `json:"title"`
	Content     string     `json:"content,omitempty"`
	Desc        string     `json:"desc,omitempty"`
	AllDay      bool       `json:"allDay,omitempty"`
	StartDate   string     `json:"startDate,omitempty"`
	DueDate     string     `json:"dueDate,omitempty"`
	TimeZone    string     `json:"timeZone,omitempty"`
	Reminders   []string   `json:"reminders,omitempty"`
	Priority    int        `json:"priority,omitempty"`
	Status      int        `json:"status,omitempty"`
	SortOrder   int64      `json:"sortOrder,omitempty"`
	Items       []CheckItem `json:"items,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
}

// FlexibleDate handles TickTick's date fields which can be strings (input) or numbers (output)
type FlexibleDate string

func (f *FlexibleDate) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexibleDate(s)
		return nil
	}
	// Try number (Unix timestamp in milliseconds)
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		if n > 0 {
			t := time.UnixMilli(n)
			*f = FlexibleDate(t.Format(time.RFC3339))
		}
		return nil
	}
	return nil
}

func (f FlexibleDate) MarshalJSON() ([]byte, error) {
	if f == "" {
		return []byte("null"), nil
	}
	return json.Marshal(string(f))
}

type CheckItem struct {
	ID            string       `json:"id,omitempty"`
	Title         string       `json:"title"`
	Status        int          `json:"status,omitempty"`
	SortOrder     int64        `json:"sortOrder,omitempty"`
	StartDate     FlexibleDate `json:"startDate,omitempty"`
	DueDate       FlexibleDate `json:"dueDate,omitempty"`
	IsAllDay      bool         `json:"isAllDay,omitempty"`
	TimeZone      string       `json:"timeZone,omitempty"`
	CompletedTime FlexibleDate `json:"completedTime,omitempty"`
}

func NewClient(clientID, clientSecret, redirectURI string) *Client {
	if redirectURI == "" {
		redirectURI = DefaultRedirect
	}
	return &Client{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
	}
}

func (c *Client) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("redirect_uri", c.redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", DefaultScopes)
	if state != "" {
		params.Set("state", state)
	}
	return fmt.Sprintf("%s?%s", AuthURL, params.Encode())
}

func (c *Client) ExchangeCode(code string) (*Token, error) {
	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", c.redirectURI)

	req, err := http.NewRequest("POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s - %s", resp.Status, string(body))
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	c.token = &token
	return &token, nil
}

func (c *Client) RefreshToken() (*Token, error) {
	if c.token == nil || c.token.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("refresh_token", c.token.RefreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequest("POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: %s - %s", resp.Status, string(body))
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	c.token = &token
	return &token, nil
}

func (c *Client) SetToken(token *Token) {
	c.token = token
}

func (c *Client) GetToken() *Token {
	return c.token
}

func (c *Client) IsAuthenticated() bool {
	return c.token != nil && c.token.AccessToken != ""
}

func (c *Client) IsTokenExpired() bool {
	if c.token == nil {
		return true
	}
	return time.Now().After(c.token.ExpiresAt.Add(-5 * time.Minute))
}

func (c *Client) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	url := fmt.Sprintf("%s%s", BaseURL, endpoint)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed: %s - %s", resp.Status, string(respBody))
	}

	return respBody, nil
}

func (c *Client) GetProjects() ([]Project, error) {
	body, err := c.doRequest("GET", "/project", nil)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, fmt.Errorf("parsing projects: %w", err)
	}

	return projects, nil
}

func (c *Client) CreateTask(task *Task) (*Task, error) {
	body, err := c.doRequest("POST", "/task", task)
	if err != nil {
		return nil, err
	}

	var createdTask Task
	if err := json.Unmarshal(body, &createdTask); err != nil {
		return nil, fmt.Errorf("parsing created task: %w", err)
	}

	return &createdTask, nil
}

func (c *Client) CreateProject(name string) (*Project, error) {
	project := &Project{
		Name: name,
	}

	body, err := c.doRequest("POST", "/project", project)
	if err != nil {
		return nil, err
	}

	var createdProject Project
	if err := json.Unmarshal(body, &createdProject); err != nil {
		return nil, fmt.Errorf("parsing created project: %w", err)
	}

	return &createdProject, nil
}
