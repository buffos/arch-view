# MCP installation

Arch View can expose the same live analysis data to a coding assistant through
an MCP-compatible newline-delimited JSON-RPC server.

MCP uses the same analyzer packaging choices as the command line. A
single-binary installation runs the built-in analyzers in-process. A packaged
release has an `analyzers/` catalog and starts verified analyzer child
processes. The packaged form provides process isolation and independently
shippable analyzer packages; the single binary is easier to install. See the
[installation guide](/guide/installation) for the full distinction and file
layouts.

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

Some MCP clients use the following common JSON shape:

~~~json
{
  "mcpServers": {
    "arch-view-app": {
      "command": "arch-view",
      "args": ["mcp", "--project", "/home/alice/code/app"]
    },
    "arch-view-library": {
      "command": "arch-view",
      "args": ["mcp", "--project", "/home/alice/code/library"]
    }
  }
}
~~~

This JSON shape is common, but MCP does not require every editor to use the
same configuration file or field names. Follow the editor's MCP setup guide
when choosing where to put it. Codex uses TOML; its setup appears below. On
Windows, use the escaped Windows paths shown above.

Each entry starts one Arch View process and one live session for one repository.
One or two entries are a normal, practical setup. Five repositories can mean
up to five Arch View processes, watchers, analyzer states, and initial scans.
There is no special hard limit of five; resource use grows with the number of
enabled entries. Disable entries for repositories you are not using when the
client supports that option.

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

For a Codex project that contains several repositories, use the user-level
Codex configuration with one named entry per repository. Do not assume that a
single global entry with `--project .` will switch roots automatically.

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

Codex then exposes two named Arch View servers. Add one entry for each
repository you want available. A project-scoped `.codex/config.toml` is useful
when a Codex project contains only one repository; for a multi-repository
Codex project, the central named-entry setup is the reliable choice.
Project-scoped configuration is used only for trusted repositories.

## Claude Desktop setup

Claude Desktop uses a JSON configuration file for local MCP servers. Its
configuration is similar to the generic `mcpServers` example above. Add one
named server entry for each repository:

~~~json
{
  "mcpServers": {
    "arch-view-app": {
      "command": "arch-view.exe",
      "args": ["mcp", "--project", "C:\\work\\app"]
    },
    "arch-view-library": {
      "command": "arch-view.exe",
      "args": ["mcp", "--project", "C:\\work\\library"]
    }
  }
}
~~~

On macOS or Linux, use the Unix executable and repository paths instead:

~~~json
{
  "mcpServers": {
    "arch-view-app": {
      "command": "arch-view",
      "args": ["mcp", "--project", "/home/alice/code/app"]
    },
    "arch-view-library": {
      "command": "arch-view",
      "args": ["mcp", "--project", "/home/alice/code/library"]
    }
  }
}
~~~

Claude Desktop normally keeps this configuration in
`%APPDATA%\\Claude\\claude_desktop_config.json` on Windows and
`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS.
Use Claude's current local MCP setup instructions if the application provides
a different configuration location or a managed extension flow.

After changing the file, restart Claude Desktop so it starts the configured
servers. Claude can then use the named repository servers, but one server
entry still has one fixed project root. A single `--project .` entry does not
automatically switch between repositories in a multi-repository workspace.

## Required options

`--project` is required at server startup. It names the single repository that
Arch View may inspect during that server session. You normally set it once in
the MCP client configuration; the MCP request cannot replace it with another
path.

## Quality baselines

The server uses the profile's baseline reference by default. It reads only the
matching `baseline_id` and `revision` from `quality-baselines/`. A missing
baseline is reported as a warning and does not suppress findings. An invalid
or ambiguous match is an error.

An agent can use `get_quality_baselines` to list saved baseline documents or
read a bounded page of one safe filename. `evaluate_quality` accepts
`baseline_mode` values `profile` (default), `none`, and `selected`. The
`selected` mode reads the listed baseline files for that request only; it does
not change the profile or server session.

Baseline writes are disabled by default. Start the server with
`--allow-policy-writes` and the configured policy token before using the
authorized `append_baseline` operation. It merges reviewed active findings
with observed coverage into the canonical profile baseline, updates the
profile reference, and tells the client to evaluate again. It never edits
source files. See [MCP tools](/mcp/tools) and [Baselines](/quality/baselines)
for request details.

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

For a ready-to-paste MCP quality-review prompt, see the [MCP quality-review
prompt](/mcp/tools#copy-paste-prompt-for-an-mcp-quality-review). For the
dual-mode workflow that uses MCP when available and the CLI otherwise, see the
[agent skill](/mcp/agent-skill).
