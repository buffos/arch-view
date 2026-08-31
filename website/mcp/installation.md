# MCP installation

Arch View can expose the same live analysis data to a coding assistant through
an MCP-compatible newline-delimited JSON-RPC server.

## Start the stdio server

~~~powershell
go run ./cmd/arch-view mcp --project .
~~~

The process reads protocol messages from standard input and writes protocol
messages to standard output. Keep diagnostic output on standard error. Do not
put a shell pipe or another program between the MCP client and this command.

For an MCP client, configure the command and arguments like this:

~~~json
{
  "command": "go",
  "args": ["run", "./cmd/arch-view", "mcp", "--project", "C:/work/my-project"]
}
~~~

When using a built binary, replace `go run ./cmd/arch-view` with the
executable name or path. Here are complete examples for a hypothetical
installation:

### Windows

If `arch-view.exe` is on `PATH`:

~~~json
{
  "command": "arch-view.exe",
  "args": ["mcp", "--project", "C:\\work\\my-project"]
}
~~~

If it is not on `PATH`:

~~~json
{
  "command": "C:\\Tools\\arch-view\\arch-view.exe",
  "args": ["mcp", "--project", "C:\\work\\my-project"]
}
~~~

### macOS

If `arch-view` is on `PATH`:

~~~json
{
  "command": "arch-view",
  "args": ["mcp", "--project", "/Users/alice/Code/my-project"]
}
~~~

If it is installed in a fixed directory:

~~~json
{
  "command": "/Users/alice/bin/arch-view",
  "args": ["mcp", "--project", "/Users/alice/Code/my-project"]
}
~~~

### Linux

If `arch-view` is on `PATH`:

~~~json
{
  "command": "arch-view",
  "args": ["mcp", "--project", "/home/alice/code/my-project"]
}
~~~

If it is installed in a fixed directory:

~~~json
{
  "command": "/home/alice/bin/arch-view",
  "args": ["mcp", "--project", "/home/alice/code/my-project"]
}
~~~

The executable path and the project path are configured once when the MCP
client starts the server. They are not supplied again for every MCP request.
The server keeps the project root fixed for its entire session. This is a
security boundary: a request cannot make the server inspect another directory,
leave the configured repository, or silently broaden its permissions.

If your MCP client starts commands with a chosen working directory, you can use
the repository-relative form instead:

~~~json
{
  "command": "arch-view",
  "args": ["mcp", "--project", "."]
}
~~~

This works only when the client starts the process in the intended repository.
Use an absolute project path when the client does not provide a working
directory setting.

## Using several repositories

The current MCP server is one repository per process. For several repositories,
create one named server entry per repository in the MCP client's configuration.
The assistant then selects the named server; you do not edit a path for each
tool call.

For example, a typical client configuration can contain:

~~~json
{
  "mcpServers": {
    "arch-view-application": {
      "command": "arch-view",
      "args": ["mcp", "--project", "/home/alice/code/application"]
    },
    "arch-view-library": {
      "command": "arch-view",
      "args": ["mcp", "--project", "/home/alice/code/library"]
    },
    "arch-view-tools": {
      "command": "arch-view",
      "args": ["mcp", "--project", "/home/alice/code/tools"]
    }
  }
}
~~~

Repeat the entry for each repository. On Windows, use the escaped Windows
paths shown above. If the client supports workspace-scoped MCP configuration,
put the same entry in each repository and use `--project .`.

Arch View does not currently expose a tool that accepts an arbitrary repository
path or switches the root of a running session. A future multi-repository
broker could support named, pre-approved roots, but it must keep an explicit
allowlist; accepting any path from an MCP request would remove the safety
boundary.

## Codex setup

The JSON examples above are a generic MCP-client format. Codex stores MCP
stdio servers in TOML configuration. Codex supports both a server working
directory (`cwd`) and project-scoped `.codex/config.toml` files. That means you
can use the same `--project .` arguments in every repository and set the
working directory once for that repository.

In each trusted repository, create `.codex/config.toml` with its own absolute
working directory:

~~~toml
[mcp_servers.arch_view]
command = "arch-view.exe"
args = ["mcp", "--project", "."]
cwd = "C:\\Users\\alice\\Code\\my-project"
~~~

On macOS or Linux, use the same configuration with a Unix path:

~~~toml
[mcp_servers.arch_view]
command = "arch-view"
args = ["mcp", "--project", "."]
cwd = "/home/alice/code/my-project"
~~~

Repeat the file for each repository, changing only `cwd`. For example, when
working in `application`, Codex loads that repository's configuration and
starts Arch View with `--project .` there. When working in `library`, it loads
the library configuration and starts a separate Arch View session for the
library. You do not change the path during a conversation or for individual
tool calls.

You can also put several named entries in the user-level Codex configuration
file if you prefer one central file:

~~~toml
[mcp_servers.arch_view_application]
command = "arch-view.exe"
args = ["mcp", "--project", "."]
cwd = "C:\\Users\\alice\\Code\\application"

[mcp_servers.arch_view_library]
command = "arch-view.exe"
args = ["mcp", "--project", "."]
cwd = "C:\\Users\\alice\\Code\\library"
~~~

Codex then exposes two named Arch View servers. The project-scoped approach is
usually easier because the active repository determines which one is loaded.
Project configuration is used only for trusted repositories.

## Required options

`--project` is required at server startup. It names the single repository that
Arch View may inspect during that server session. You normally set it once in
the MCP client configuration; the MCP request cannot replace it with another
path.

## Options

| Option | What it does |
| --- | --- |
| **--project** | Repository directory to analyze. Required. |
| **--transport** | `stdio` (default), `local_http`, or `authenticated_http`. |
| **--port** | Loopback HTTP port for an HTTP transport. `0` chooses an available port. |
| **--auth-token** | Required token for `authenticated_http`. |
| **--origin** | Allows one cross-origin browser client for HTTP. Repeatable. Without it, browser requests must be same-origin. |
| **--session-id** | Optional stable live session ID. |
| **--quality-profile** | Loads a quality profile JSON file for the session. |
| **--no-source-index** | Disables source files, symbols, and documentation queries. |
| **--no-watch** | Disables filesystem watching. |
| **--analyzer-runtime** | Selects `auto`, `packaged`, `in-process`, or `explicit` analyzer runtime. |
| **--plugin** | Supplies an external analyzer descriptor. Repeatable. |
| **--allow-untrusted-plugin** | Allows an explicitly supplied local descriptor. |
| **--allow-policy-writes** | Enables explicitly authorized profile and baseline writes. Off by default. |
| **--policy-token** | Opaque token required with `--allow-policy-writes`. |

The stdio server starts its own live session. It does not run the target
application, execute shell commands, edit source files, install packages, or
accept a repository root from a request.

## HTTP mode

HTTP is opt-in and binds to loopback only:

~~~powershell
go run ./cmd/arch-view mcp --project . --transport local_http --port 0
go run ./cmd/arch-view mcp --project . --transport authenticated_http --port 0 --auth-token change-me --origin http://localhost:3000
~~~

The command prints an endpoint and keeps running until `Ctrl+C`. The HTTP
routes are documented on [HTTP transport](/mcp/http). Use an authentication
token for any client other than a controlled local process.
