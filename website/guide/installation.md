# Installation

Arch View can be used in two different ways. The analysis logic is the same,
but the analyzers are packaged differently.

## Choose an installation

| You want to... | Use... |
| --- | --- |
| Develop Arch View from its source checkout | Source checkout |
| Install one command for personal use | Single-binary installation |
| Ship an application with isolated analyzer processes | Packaged installation |

The single-binary installation is the simplest choice for most users. The
packaged installation is useful when process isolation, independent analyzer
packages, or third-party analyzers matter.

## Option 1: run from a source checkout

This is useful when you are developing Arch View itself.

Requirements:

- Go 1.23 or newer.
- A C compiler available to cgo. The built-in parsers use tree-sitter through
  cgo, so builds need `CGO_ENABLED=1`.

From the Arch View checkout, run:

~~~powershell
go run ./cmd/arch-view open --project C:\work\my-project --port 0
~~~

`go run` builds the command and runs it. See [Quick start](/guide/quick-start)
for the first viewer session.

## Option 2: single-binary installation

Use this when you want to use Arch View without keeping its source checkout.

~~~text
go install github.com/buffo/arch-view/cmd/arch-view@latest
~~~

Go downloads the Arch View source and its dependencies, compiles the host and
the built-in analyzers, and installs one command:

~~~text
arch-view
~~~

On Windows the file is `arch-view.exe`. On macOS and Linux it is `arch-view`.
The executable is placed in `GOBIN`, or in `GOPATH/bin` when `GOBIN` is empty:

~~~text
go env GOBIN
go env GOPATH
~~~

The downloaded source is kept in Go's module cache. It is not copied into the
project that Arch View analyzes. The analyzer source is not compiled later at
runtime: it is compiled and linked into the executable during `go install`.

### Build the un-packaged binary yourself

If you have the Arch View checkout and want to create the single executable
without installing it into Go's bin directory, use `go build`:

~~~powershell
go build -o .\arch-view.exe .\cmd\arch-view
~~~

On macOS or Linux, use:

~~~text
go build -o ./arch-view ./cmd/arch-view
~~~

This creates only the host executable. The built-in analyzers are linked into
that executable, so no `analyzers/` directory is created. Run the result
from the checkout with:

~~~powershell
.\arch-view.exe open --project C:\work\my-project --port 0
~~~

On macOS or Linux:

~~~text
./arch-view open --project /home/alice/code/my-project --port 0
~~~

In other words, these are all un-packaged, single-binary workflows:

~~~text
go run ./cmd/arch-view       # build temporarily and run
go build ./cmd/arch-view     # create a local executable
go install ./cmd/arch-view   # install the current checkout's executable
~~~

### What this installation contains

The one executable contains the built-in Go, Python, TypeScript, Rust, and
Clojure analyzers:

~~~text
arch-view
  ├─ Go analyzer
  ├─ Python analyzer
  ├─ TypeScript analyzer
  ├─ Rust analyzer
  └─ Clojure analyzer
~~~

There is no separate `analyzers/` directory in this installation. With the
normal `auto` setting, Arch View uses these analyzers in-process.

Run it from any project directory:

~~~text
arch-view open --project . --port 0
~~~

If the command is not on `PATH`, run the executable using its full path. For
example, a Windows installation may use:

~~~text
C:\Users\alice\go\bin\arch-view.exe open --project C:\work\my-project --port 0
~~~

On macOS or Linux, the equivalent may be:

~~~text
/home/alice/go/bin/arch-view open --project /home/alice/code/my-project --port 0
~~~

## Option 3: packaged installation

A packaged installation is a release archive containing the host and separate,
already-compiled analyzer packages. A simplified installation looks like this:

~~~text
arch-view.exe                  # the host on Windows
analyzers/
  index.json                   # the package catalog
  org.archview.go/
    windows-amd64/
      analyzer.exe
      descriptor.json
  org.archview.python/
    windows-amd64/
      analyzer.exe
      descriptor.json
  ...
~~~

On macOS and Linux the executable names and platform directories are
different. A release archive supplies the correct files for its platform.

When the host finds an `analyzers` directory beside itself, or an analyzer root
configured with `ARCH_VIEW_ANALYZER_ROOT` or `ARCH_VIEW_ANALYZERS_ROOT`, `auto`
selects packaged mode. It
reads `index.json`, verifies the selected executable and descriptor, and then
starts the required analyzer as a child process when analysis is requested.
The host and analyzer exchange bounded protocol messages over standard input
and standard output.

The host does not search the project being analyzed for analyzer executables.
The analyzer files belong to the Arch View installation.

### Create a packaged build

From the Arch View checkout, use this one command to create the complete
host-plus-analyzers distribution:

~~~text
make release
~~~

`make release` compiles the host, compiles all enabled analyzer entrypoints,
creates the analyzer packages and `index.json`, and writes the complete result
under `dist/release/`. You do not need to run another command first.

For a package-only build, without the host application, use:

~~~text
make analyzers
~~~

That writes the analyzer packages and `index.json` under `dist/analyzers/`.
Both targets compile the analyzer entrypoints before the release is used; they
do not compile analyzers when a user later opens a project.

### Why have separate analyzer processes?

Separate analyzers are not needed for the basic one-binary installation. They
provide useful boundaries for a packaged release:

- A parser crash or memory problem is isolated from the viewer and host.
- Each analyzer can have its own native dependencies and platform build.
- An analyzer package can be tested, replaced, or updated independently of the
  host process.
- Optional or third-party analyzers can be added without importing their Go
  source into the host.
- The host stays language-neutral and talks to every analyzer through the same
  contract.

The tradeoff is that a packaged installation has more files and needs a valid
catalog. It is more complex to distribute than one executable.

### Can the catalog contain an unknown analyzer?

Yes. Packaged mode does not require the analyzer ID to be hardcoded in the
host's Go source. The package must still be registered in `index.json` and
include a matching descriptor, executable, API version, platform information,
and integrity hashes. An arbitrary executable dropped into the directory is
not enough.

The catalog is the runtime registration for packaged analyzers. The host can
read it and create an analyzer entry without rebuilding the host. The standard
release builder currently assembles the official Arch View analyzer set; a
third-party package must produce a catalog entry that satisfies the same
contract.

## How Arch View chooses the mode

The host exposes the choice when you need to inspect it or override it:

| Mode | Meaning |
| --- | --- |
| `auto` | Use a valid packaged installation when one is present; otherwise use the built-in analyzers in-process. |
| `in-process` | Force the built-in analyzers linked into the host. |
| `packaged` | Require the application-managed analyzer catalog and child executables. |
| `explicit` | Use an external analyzer descriptor supplied with the command. |

Examples:

~~~text
arch-view analyzers
arch-view analyzers --analyzer-runtime in-process
arch-view analyzers --analyzer-runtime packaged
~~~

`packaged` reports an error when the required catalog is missing or invalid.
`auto` does not compile or download analyzers at runtime. It only chooses
between already-installed analyzer implementations.

Do not use `go install github.com/buffo/arch-view/cmd/analyzers/...` as a
normal end-user installation. That command creates separate child binaries,
but it does not by itself create the catalog and verified package layout that
packaged mode needs.

## MCP installation

MCP uses the same host and analyzer modes. The MCP client starts one Arch View
server process for a fixed project. See [MCP installation](/mcp/installation)
for stdio, HTTP, Codex, Claude Desktop, and multi-repository configuration.
