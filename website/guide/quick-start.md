# Quick start

This page gets you from a project folder to a browser view.

If you are installing Arch View for normal use, start with [Installation](/guide/installation)
to choose between the single-binary and packaged distributions. The commands
below use the source-checkout form so that every step is visible.

## 1. Check that Go is available

~~~powershell
go version
~~~

Arch View is a Go program. The exact minimum Go version is defined by the repository's go.mod file.

## 2. Open a project directly

From the Arch View checkout:

~~~powershell
go run ./cmd/arch-view open --project C:\path\to\your\project --port 0
~~~

What each part means:

| Part | Meaning |
| --- | --- |
| go run | Compile and run the program for this command. |
| ./cmd/arch-view | The Arch View command-line program. |
| open | Start the browser viewer. |
| --project ... | Tell Arch View which project to read. |
| --port 0 | Ask the operating system to choose a free local port. |

The terminal prints a local address. Open that address in your browser.

## 3. Use the current folder

If the terminal is already at the project root:

~~~powershell
go run ./cmd/arch-view open --project . --port 0
~~~

The dot means “this folder.”

## 4. Open an existing model

If you already have a canonical model JSON file:

~~~powershell
go run ./cmd/arch-view open --model architecture.json --port 0
~~~

This skips analysis. The viewer opens the information already stored in the model.

## 5. Create a report without opening a browser

~~~powershell
go run ./cmd/arch-view analyze --project . --format analysis-json --output analysis.json
~~~

The JSON file is useful in CI, scripts, and later export steps.

## 6. Export a browser file

First create or obtain a canonical model, then export it:

~~~powershell
go run ./cmd/arch-view export --input model.json --format html --output architecture.html
~~~

The HTML file is self-contained. It can be opened without starting the Arch View server.

## What should happen?

You should see a graph, a summary area, and controls for search, scope, layout, and quality checks when a report contains them.

If the viewer does not load, start with [Common questions](/troubleshooting/common-questions). The most useful first check is to copy the complete terminal output, including the command and any error text.

## Safety note

Arch View is designed for local, read-only analysis. The default value of --safe-mode disables target-code execution and tool-assisted execution. Keep that default unless you understand why a different behavior is needed.
