# Command line overview

The command-line program is named Arch View.

Start with:

~~~powershell
go run ./cmd/arch-view --help
~~~

The main commands are:

| Command | Use it when you want to |
| --- | --- |
| analyzers | List the analyzers available on this machine. |
| analyze | Read a project and create a report. |
| open | Start the local browser viewer. |
| export | Turn a model into JSON, HTML, or SVG. |
| quality baseline | Record reviewed findings in a baseline file. |
| model normalize | Convert analysis JSON into canonical model JSON. |
| model validate | Check that a model JSON file is valid. |
| model projection | Create a smaller hierarchy view from a model. |

## A repeatable workflow

~~~text
analyze -> normalize if needed -> open or export -> review quality -> baseline accepted findings
~~~

Use the [Quick start](/guide/quick-start) for the shortest path.

## Exit codes

The command uses a non-zero exit code for invalid input, failed analysis, unsupported options, or a configured quality policy that matches a finding.

In automation, always keep the terminal output. It tells you whether the command failed before analysis, during analysis, or after a quality report was evaluated.

## Repeated options

Some options are repeatable. For example:

~~~powershell
go run ./cmd/arch-view analyze --project . --exclude vendor --exclude generated
~~~

The two exclusions are collected together. A comma-separated form is also accepted for quality exit statuses.
