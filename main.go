package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"net/http"

	mcp_golang "github.com/metoro-io/mcp-golang"
	mcphttp "github.com/metoro-io/mcp-golang/transport/http"
)

const (
	defaultAppPort = 8080
	ServerName     = "Jira MCP Server"
	ServerVersion  = "1.0.0"
)

// JiraConfig holds Jira connection details
type JiraConfig struct {
	URL    string `json:"url" jsonschema:"required,description=The Jira instance URL (Cloud or Data Center)"`
	APIKey string `json:"api_key" jsonschema:"required,description=The Jira API key or Personal Access Token"`
	Email  string `json:"email" jsonschema:"required,description=The email address for Jira Cloud authentication"`
}

// CreateIssueArgs defines arguments for creating a Jira issue
type CreateIssueArgs struct {
	JiraConfig  JiraConfig `json:"jira_config" jsonschema:"required,description=Jira connection configuration"`
	ProjectKey  string     `json:"project_key" jsonschema:"required,description=The key of the project to create the issue in"`
	Summary     string     `json:"summary" jsonschema:"required,description=The summary or title of the issue"`
	Description string     `json:"description" jsonschema:"description=Optional description of the issue"`
	IssueType   string     `json:"issue_type" jsonschema:"required,description=The type of issue (e.g., Bug, Story, Task)"`
}

// UpdateIssueArgs defines arguments for updating a Jira issue
type UpdateIssueArgs struct {
	JiraConfig  JiraConfig `json:"jira_config" jsonschema:"required,description=Jira connection configuration"`
	IssueKey    string     `json:"issue_key" jsonschema:"required,description=The key of the issue to update (e.g., PROJ-123)"`
	Summary     string     `json:"summary" jsonschema:"description=The new summary or title of the issue"`
	Description string     `json:"description" jsonschema:"description=The new description of the issue"`
}

// SearchIssuesArgs defines arguments for searching Jira issues
type SearchIssuesArgs struct {
	JiraConfig JiraConfig `json:"jira_config" jsonschema:"required,description=Jira connection configuration"`
	JQL        string     `json:"jql" jsonschema:"required,description=The JQL query to search issues (e.g., 'project = PROJ AND status = Open')"`
}

// JiraClient encapsulates Jira API interactions
type JiraClient struct {
	config     JiraConfig
	httpClient *http.Client
	isCloud    bool
}

// NewJiraClient initializes a Jira client
func NewJiraClient(config JiraConfig) *JiraClient {
	isCloud := strings.Contains(strings.ToLower(config.URL), ".atlassian.net")
	return &JiraClient{
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		isCloud:    isCloud,
	}
}

// getBaseAPIPath returns the appropriate API base path based on Jira type
func (c *JiraClient) getBaseAPIPath() string {
	if c.isCloud {
		return "/rest/api/3"
	}
	return "/rest/api/2" // Data Center typically uses /api/2, but some endpoints may vary
}

// createIssue creates a new issue in Jira
func (c *JiraClient) createIssue(args CreateIssueArgs) (string, error) {
	url := fmt.Sprintf("%s%s/issue", c.config.URL, c.getBaseAPIPath())
	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project": map[string]string{
				"key": args.ProjectKey,
			},
			"summary":     args.Summary,
			"description": args.Description,
			"issuetype": map[string]string{
				"name": args.IssueType,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	if c.isCloud {
		req.SetBasicAuth(c.config.Email, c.config.APIKey)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to create issue, status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	issueKey, ok := result["key"].(string)
	if !ok {
		return "", fmt.Errorf("issue key not found in response")
	}

	return issueKey, nil
}

// updateIssue updates an existing issue in Jira
func (c *JiraClient) updateIssue(args UpdateIssueArgs) (string, error) {
	url := fmt.Sprintf("%s%s/issue/%s", c.config.URL, c.getBaseAPIPath(), args.IssueKey)
	payload := map[string]interface{}{
		"fields": map[string]interface{}{},
	}

	if args.Summary != "" {
		payload["fields"].(map[string]interface{})["summary"] = args.Summary
	}
	if args.Description != "" {
		payload["fields"].(map[string]interface{})["description"] = args.Description
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("PUT", url, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	if c.isCloud {
		req.SetBasicAuth(c.config.Email, c.config.APIKey)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to update issue, status: %d", resp.StatusCode)
	}

	return args.IssueKey, nil
}

// searchIssues searches issues using JQL
func (c *JiraClient) searchIssues(args SearchIssuesArgs) ([]map[string]string, error) {
	url := fmt.Sprintf("%s%s/search?jql=%s&fields=summary", c.config.URL, c.getBaseAPIPath(), args.JQL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	if c.isCloud {
		req.SetBasicAuth(c.config.Email, c.config.APIKey)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to search issues, status: %d", resp.StatusCode)
	}

	var result struct {
		Issues []struct {
			Key    string `json:"key"`
			Fields struct {
				Summary string `json:"summary"`
			} `json:"fields"`
		} `json:"issues"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	issues := make([]map[string]string, 0, len(result.Issues))
	for _, issue := range result.Issues {
		issues = append(issues, map[string]string{
			"key":     issue.Key,
			"summary": issue.Fields.Summary,
		})
	}

	return issues, nil
}

func main() {
	log.Println("Starting Jira MCP Server...")

	// Parse command line arguments
	var appPort int
	flag.IntVar(&appPort, "port", defaultAppPort, "The port to listen on")
	flag.Parse()

	log.Printf("Listening on port: %d", appPort)

	// Initialize MCP server with HTTP transport
	transport := mcphttp.NewHTTPTransport("/mcp")
	transport.WithAddr(fmt.Sprintf(":%d", appPort))
	server := mcp_golang.NewServer(transport)

	// Register server info endpoint
	err := server.RegisterResource("info://server", "Server Information", "Provides details about the server and available actions", "application/json",
		func() (*mcp_golang.ResourceResponse, error) {
			info := map[string]interface{}{
				"name":    ServerName,
				"version": ServerVersion,
				"actions": []string{
					"create_issue",
					"update_issue",
					"search_issues",
				},
			}
			return mcp_golang.NewResourceResponse(
				mcp_golang.NewTextEmbeddedResource(
					"info://server",
					string(mustMarshal(info)),
					"application/json",
				),
			), nil
		})
	if err != nil {
		log.Fatalf("Error registering server info: %v", err)
	}

	// Register create issue tool
	err = server.RegisterTool("create_issue", "Create a new Jira issue",
		func(args CreateIssueArgs) (*mcp_golang.ToolResponse, error) {
			client := NewJiraClient(args.JiraConfig)
			issueKey, err := client.createIssue(args)
			if err != nil {
				return nil, err
			}
			return mcp_golang.NewToolResponse(
				mcp_golang.NewTextContent(fmt.Sprintf("Created issue: %s", issueKey)),
			), nil
		})
	if err != nil {
		log.Fatalf("Error registering create_issue tool: %v", err)
	}

	// Register update issue tool
	err = server.RegisterTool("update_issue", "Update an existing Jira issue",
		func(args UpdateIssueArgs) (*mcp_golang.ToolResponse, error) {
			client := NewJiraClient(args.JiraConfig)
			issueKey, err := client.updateIssue(args)
			if err != nil {
				return nil, err
			}
			return mcp_golang.NewToolResponse(
				mcp_golang.NewTextContent(fmt.Sprintf("Updated issue: %s", issueKey)),
			), nil
		})
	if err != nil {
		log.Fatalf("Error registering update_issue tool: %v", err)
	}

	// Register search issues tool
	err = server.RegisterTool("search_issues", "Search Jira issues using JQL",
		func(args SearchIssuesArgs) (*mcp_golang.ToolResponse, error) {
			client := NewJiraClient(args.JiraConfig)
			issues, err := client.searchIssues(args)
			if err != nil {
				return nil, err
			}
			return mcp_golang.NewToolResponse(
				mcp_golang.NewTextContent(string(mustMarshal(issues))),
			), nil
		})
	if err != nil {
		log.Fatalf("Error registering search_issues tool: %v", err)
	}

	// Start the server
	if err := server.Serve(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// mustMarshal is a helper to marshal JSON or panic
func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
