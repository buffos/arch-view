# HTTP transport

HTTP is an optional adapter over the same live session used by the viewer,
CLI, and stdio MCP server. It does not add new analysis semantics.

## Start it

~~~powershell
go run ./cmd/arch-view mcp --project . --transport local_http --port 0
~~~

This binds to `127.0.0.1` only. Use `authenticated_http` when another process
or browser client needs access:

~~~powershell
go run ./cmd/arch-view mcp --project . --transport authenticated_http --port 0 --auth-token a-long-random-token --origin http://localhost:3000
~~~

Authenticated requests must send either:

~~~text
Authorization: Bearer a-long-random-token
~~~

or the equivalent `X-Arch-View-Token` header. Browser requests are same-origin
by default. A cross-origin browser client must be listed with `--origin`; the
server rejects other origins and accepts an unauthenticated `OPTIONS`
preflight only for an allowed origin. The server does not accept a repository
root, filesystem path, shell command, or permission grant from a client
request.

## Routes

The endpoint printed by the command is the session prefix. The available
routes are:

| Method and suffix | Use |
| --- | --- |
| `GET /status` | Read state and freshness. |
| `POST /ensure-current` | Wait for a verified current revision. |
| `GET /scopes` | List scopes. |
| `POST /search/files`, `/search/symbols`, `/search/documentation` | Run bounded structural queries. |
| `POST /search/text` | Run bounded exact text search. |
| `GET /modules/{id}` | Read module facts. |
| `GET /callers-callees/{id}` | Read callers and callees. |
| `POST /source-context` | Read explicit bounded source context. |
| `GET /quality` | Read quality profiles and rules. |
| `POST /quality/evaluate` | Run temporary quality evaluation. |
| `POST /quality/findings` | Read bounded findings. |
| `POST /quality/evidence` | Read finding evidence. |
| `POST /quality/compare` | Compare reports. |
| `POST /policy` | Run a separately permissioned policy operation. |

All successful responses carry the same revision, freshness, coverage,
ordering, cursor, and budget fields as stdio. Authentication, origin, and
HTTP error envelopes are transport-specific.

## Safety

Local and authenticated HTTP remain read-only by default. Profile and baseline
writes require all of these: an allowlisted operation, explicit authorization,
a validated document or exact current report, safe project-relative destination,
and a reason for a baseline. Failed checks do not change source files,
analysis snapshots, or existing policy documents.
