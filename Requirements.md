### Please use below information and create Jira MCP server in golang

- MCP (Model Context Protocol) reference document https://modelcontextprotocol.io/llms-full.txt to get the stanadards to build MCP server.
- Use golang package https://github.com/metoro-io/mcp-golang to build a MCP server in golang
- This will be a Jira Cloud & data center actions MCP server, that will accept Jira URL and determine the relevant REST API end points to use, as there will be difference between Cloud and DC versions
- MCP server will require Jira URL and API Key to be used as bearer token in the REST API calls
- Actions supported,
  - create issue in a specific project, return created issue key
  - update an issue based on the issue key, return updated issue key
  - search issues across different projects based on search criteria and return list of found issues with their Summary field
- It will be HTTP based REST API supporting server that will provide an endpoint to give details of the server and possible actions that can be used by any MCP Client to configure as tools to LLM call
