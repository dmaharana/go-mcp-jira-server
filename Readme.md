# Explanation of the Implementation

## MCP Server Setup

The server uses the `metoro-io/mcp-golang` package with an HTTP transport layer (port 8080) to comply with the requirement for an HTTP-based REST API.
A resource endpoint (`info://server`) is registered to provide metadata about the server, including its name, version, and available actions (`create_issue`, `update_issue`, `search_issues`). This allows MCP clients to discover the server's capabilities.

## Jira Client

The `JiraClient` struct encapsulates Jira API interactions and determines whether the instance is Jira Cloud (based on `.atlassian.net` in the URL) or Data Center.
Authentication is handled differently:

- Jira Cloud: Uses Basic Auth with the user's email and API key.
- Data Center: Uses a Bearer token with the API key (Personal Access Token).
  The `getBaseAPIPath` method selects the appropriate API path (`/rest/api/3` for Cloud, `/rest/api/2` for Data Center) to handle endpoint differences.

## Supported Actions

- Create Issue:

  - Accepts `jira_config`, `project_key`, `summary`, `description`, and `issue_type`.
  - Sends a POST request to the Jira API's issue endpoint.
  - Returns the created issue key (e.g., `PROJ-123`).

- Update Issue:

  - Accepts `jira_config`, `issue_key`, `summary`, and `description`.
  - Sends a PUT request to update the specified issue.
  - Returns the updated issue key.

- Search Issues:

  - Accepts `jira_config` and `jql` query (e.g., `project = PROJ AND status = Open`).
  - Sends a GET request to the search endpoint with the JQL query.
  - Returns a list of issues with their keys and summaries.

## MCP Tools

Each action is registered as an MCP tool with JSON schema annotations to define required and optional fields, following the MCP specification.
Tools return responses in the MCP format using `mcp_golang.NewToolResponse`.

## Security and Configuration

The `JiraConfig` struct holds the Jira URL, API key, and email (for Cloud authentication).
For this example, the configuration is hardcoded in the tool handlers. In a production environment, you should load these from environment variables or a configuration file for security.

## Error Handling

The code includes comprehensive error handling for HTTP requests, JSON marshaling/unmarshaling, and API responses.
Non-successful HTTP status codes result in descriptive error messages.

# How to Run

## Install Dependencies

    bash

`go get github.com/metoro-io/mcp-golang`

Run the Server:
bash

`go run main.go`  
The server will start on [http://localhost:8080](http://localhost:8080).

### Test with an MCP Client

Configure your MCP client to send requests with the `jira_config` fields as shown in the example payloads.
Verify that the server correctly processes the Jira URL, email, and API key for each request.

## Additional Considerations

### Security

- Transmitting API keys in every request requires secure communication (e.g., HTTPS). Ensure the server is configured to use TLS in production.
- Consider validating the provided Jira URL (e.g., checking for valid domains) to prevent misuse.

### Error Handling

The existing error handling in `JiraClient` will catch issues like invalid URLs or credentials. You may want to add specific validation for the `JiraConfig` fields (e.g., ensuring the URL is well-formed or the email is valid for Cloud instances).

### Performance

Creating a new `JiraClient` for each request is simple but may introduce overhead. If performance becomes an issue, consider caching clients for frequently used configurations (e.g., using a map keyed by URL and email).

### Jira Data Center

For Data Center, the email field may not be required, as authentication typically uses only the API token. You could make the email field optional in the `JiraConfig` struct and skip Basic Auth for Data Center instances, but the current code works as is since the `JiraClient` handles authentication appropriately.

## Testing

To test the updated server:

    Use a tool like curl or an MCP client to send requests with the `jira_config` fields.
    Verify that the server correctly processes different Jira URLs (Cloud and Data Center) and authentication credentials.
    Check that the actions (create_issue, update_issue, search_issues) return the expected results.

For example, to test `create_issue` with curl (assuming the MCP server supports raw HTTP POST for testing):

bash

````curl -X POST http://localhost:8080/tool/create_issue \
    -H "Content-Type: application/json" \
    -d '{
    "jira_config": {
        "url": "https://your-jira-instance.atlassian.net",
        "api_key": "your-api-key",
        "email": "your.email@example.com"
    },
    "project_key": "PROJ",
    "summary": "Test Issue",
    "description": "This is a test",
    "issue_type": "Task"
}'```

### Client examples

### Example Client Requests

Below are example JSON payloads for each action:

**Create Issue**

```json
{
  "function": "create_issue",
  "parameters": {
    "jira_config": {
      "url": "https://your-jira-instance.atlassian.net",
      "api_key": "your-api-key",
      "email": "your.email@example.com"
    },
    "project_key": "PROJ",
    "summary": "Test Issue",
    "description": "This is a test issue created via MCP",
    "issue_type": "Task"
  }
}
````

**Update Issue**

```json
{
  "function": "update_issue",
  "parameters": {
    "jira_config": {
      "url": "https://your-jira-instance.atlassian.net",
      "api_key": "your-api-key",
      "email": "your.email@example.com"
    },
    "issue_key": "PROJ-123",
    "summary": "Updated Test Issue",
    "description": "Updated description"
  }
}
```

**Search Issues**

```json
{
  "function": "search_issues",
  "parameters": {
    "jira_config": {
      "url": "https://your-jira-instance.atlassian.net",
      "api_key": "your-api-key",
      "email": "your.email@example.com"
    },
    "jql": "project = PROJ AND status = Open"
  }
}
```
