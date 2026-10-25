# Client Setup & Integration Guide

Connect your favorite AI assistant, coding agent, or development environment to the Geomilli MCP server. This guide provides copy-paste ready configurations for Claude Desktop, Claude Code, Cursor, VS Code, Antigravity, Docker, and standalone binaries.

---

## 1. Claude Desktop

Claude Desktop communicates with Geomilli MCP via standard input/output (`stdio`) or Streamable HTTP.

### Configuration File Locations

* **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
* **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
* **Linux**: `~/.config/Claude/claude_desktop_config.json`

### Option A: Local Binary (`stdio`, Recommended)

Compile the binary or point to a prebuilt release:

```json
{
  "mcpServers": {
    "geomilli": {
      "command": "/absolute/path/to/geomilli/geomilli-mcp/bin/geomilli-mcp",
      "args": [
        "--transport", "stdio",
        "--api-url", "http://localhost:8080"
      ]
    }
  }
}
```

> [!TIP]
> You can also run directly from source with Go without manual precompilation:
> ```json
> {
>   "mcpServers": {
>     "geomilli": {
>       "command": "go",
>       "args": [
>         "run",
>         "./cmd/geomilli-mcp",
>         "--api-url", "http://localhost:8080"
>       ],
>       "cwd": "/absolute/path/to/geomilli/geomilli-mcp"
>     }
>   }
> }
> ```

### Option B: Streamable HTTP Transport

If you have the Geomilli cluster running via Docker Compose (`make up-mcp`), connect over HTTP:

```json
{
  "mcpServers": {
    "geomilli": {
      "url": "http://localhost:8090/mcp",
      "headers": {
        "Authorization": "Bearer dev-mcp-token"
      }
    }
  }
}
```

Restart Claude Desktop after updating the configuration. You should see a hammer/tool icon indicating that Geomilli's 13+ tools are available.

---

## 2. Claude Code (CLI)

Add Geomilli MCP to [Claude Code](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/overview) with a single command:

### Local Subprocess (`stdio`)

```bash
claude mcp add geomilli -- /absolute/path/to/geomilli/geomilli-mcp/bin/geomilli-mcp --api-url http://localhost:8080
```

### Streamable HTTP Endpoint

```bash
claude mcp add geomilli http://localhost:8090/mcp --header "Authorization: Bearer dev-mcp-token"
```

Verify available tools:
```bash
claude mcp list
```

---

## 3. Cursor

Cursor supports MCP servers natively. You can configure it either at the project level or globally.

### Project-Level Configuration (`.cursor/mcp.json`)

Create a `.cursor/mcp.json` file in the root of your project:

```json
{
  "mcpServers": {
    "geomilli": {
      "command": "/absolute/path/to/geomilli/geomilli-mcp/bin/geomilli-mcp",
      "args": ["--api-url", "http://localhost:8080"]
    }
  }
}
```

### Global Cursor Configuration

1. Open **Cursor Settings** (`Cmd + ,` or `Ctrl + ,`).
2. Navigate to **Features** > **MCP**.
3. Click **Add New MCP Server**:
   - **Name**: `geomilli`
   - **Type**: `command` (or `sse`/`http`)
   - **Command**: `/absolute/path/to/geomilli/geomilli-mcp/bin/geomilli-mcp --api-url http://localhost:8080`

---

## 4. VS Code (Cline / Continue / GitHub Copilot)

For VS Code extensions such as [Cline](https://github.com/cline/cline) or [Continue](https://continue.dev):

### Cline Configuration (`cline_mcp_settings.json`)

```json
{
  "mcpServers": {
    "geomilli": {
      "command": "/absolute/path/to/geomilli/geomilli-mcp/bin/geomilli-mcp",
      "args": [
        "--transport", "stdio",
        "--api-url", "http://localhost:8080"
      ],
      "env": {
        "MCP_ENABLE_OPS_TOOLS": "false"
      }
    }
  }
}
```

### Continue Configuration (`config.json`)

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "transport": {
          "type": "stdio",
          "command": "/absolute/path/to/geomilli/geomilli-mcp/bin/geomilli-mcp",
          "args": ["--api-url", "http://localhost:8080"]
        }
      }
    ]
  }
}
```

---

## 5. Google Antigravity & AGY Agents

Antigravity provides first-class support for MCP tools. Configure Geomilli MCP in your workspace or global customization directory:

Create or edit `~/.gemini/antigravity/mcp/geomilli.json`:

```json
{
  "command": "/absolute/path/to/geomilli/geomilli-mcp/bin/geomilli-mcp",
  "args": [
    "--transport", "stdio",
    "--api-url", "http://localhost:8080"
  ],
  "env": {
    "MCP_ENABLE_OPS_TOOLS": "false"
  }
}
```

Agents in the conversation will automatically discover and invoke Geomilli tools (e.g. `geocode`, `find_nearby`, `measure_distance`).

---

## 6. Docker & Docker Compose Deployment

Geomilli provides a pre-configured Docker service with Docker Compose profile support.

### Running with Docker Compose (`make up-mcp`)

Start the complete Geomilli cluster (Meilisearch, Harvester daemon, API service, Studio dashboard, and MCP server) with one command:

```bash
make up-mcp
```

Or using Docker Compose directly:

```bash
docker compose --profile mcp up -d --build
```

The MCP server will listen at `http://localhost:8090/mcp` and automatically connect to `geomilli-api:8080` on the internal bridge network.

### Inspect Container Logs

```bash
docker compose logs -f geomilli-mcp
```

### Standalone Docker Execution

To run `geomilli-mcp` as a standalone container pointing to an external or local API:

```bash
# Build the Docker image
docker build -t geomilli-mcp ./geomilli-mcp

# Run container exposed on port 8090
docker run -d \
  --name geomilli-mcp \
  -p 8090:8090 \
  -e MCP_TRANSPORT=http \
  -e MCP_HTTP_ADDR=0.0.0.0:8090 \
  -e MCP_AUTH_TOKEN=your-secret-token \
  -e GEOMILLI_API_URL=http://host.docker.internal:8080 \
  geomilli-mcp
```

---

## 7. Local Compilation & CLI Options

### Building the Binary

Compile a native static binary using the Go toolchain:

```bash
# From repo root:
make build

# Or directly in the geomilli-mcp directory:
cd geomilli-mcp
go build -ldflags="-s -w" -o bin/geomilli-mcp ./cmd/geomilli-mcp
```

### CLI Flags & Environment Variables

`geomilli-mcp` can be configured using command-line flags or environment variables. Flags take precedence over environment variables:

| CLI Flag | Environment Variable | Default | Description |
| :--- | :--- | :--- | :--- |
| `--api-url` | `GEOMILLI_API_URL` | `http://localhost:8080` | URL of the upstream `geomilli-api` REST service. |
| `--api-token` | `GEOMILLI_API_TOKEN` | _none_ | Optional upstream Bearer token for API authentication. |
| `--transport` | `MCP_TRANSPORT` | `stdio` | Transport protocol: `stdio` or `http`. |
| `--http-addr` | `MCP_HTTP_ADDR` | `127.0.0.1:8090` | Listen host:port when running in HTTP transport mode. |
| — | `MCP_AUTH_TOKEN` | _none_ | Required Bearer token for incoming HTTP requests (mandatory on non-loopback bindings). |
| — | `MCP_ALLOWED_ORIGINS` | _none_ | Comma-separated allowlist of valid `Origin` headers for DNS-rebinding protection. |
| `--enable-ops-tools` | `MCP_ENABLE_OPS_TOOLS` | `false` | Enable Tier-2 operations and telemetry tools (`get_service_status`, `list_import_queue`, `list_importable_countries`). |
| `--lang` | `MCP_DEFAULT_LANG` | _none_ | Default ISO language code (e.g. `en`, `fr`, `nl`) for localized place names. |
| — | `MCP_REQUEST_TIMEOUT` | `10s` | Per-call timeout for upstream REST API queries. |
| — | `MCP_TAXONOMY_TTL` | `10m` | TTL duration for caching taxonomy classes and supported languages. |

---

## 8. Interactive Testing with MCP Inspector

Anthropic's official `@modelcontextprotocol/inspector` is the best tool for debugging tools, schemas, resources, and prompt templates interactively:

```bash
# Run via root Makefile:
make mcp-inspect

# Or directly using npx:
npx @modelcontextprotocol/inspector go run ./geomilli-mcp/cmd/geomilli-mcp --api-url http://localhost:8080
```

The inspector launches a local web UI where you can:
1. View all 16 registered tool schemas and parameter validation constraints.
2. Send test parameters to `geocode`, `find_nearby`, or `measure_distance` and inspect raw JSON-RPC payloads.
3. Read `geomilli://` resources directly.
4. Test workflow prompts with custom prompt arguments.
