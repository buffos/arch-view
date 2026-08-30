# Viewer overview

The viewer is the browser part of Arch View. It shows a report as a map instead of a large JSON document.

## The main areas

| Area | What it is for |
| --- | --- |
| Summary cards | Quick counts for nodes, relationships, cycles, diagnostics, evidence, and quality results. |
| Graph | The visible architecture. Select a node or group to see its short summary. |
| Search | Find a module or group by name. |
| Analysis scope | Choose which part of a multi-analyzer report you are viewing. |
| References | Show or hide external and non-local references. |
| Quality checks | See the selected profile, coverage, findings, and baseline state. |
| Inspection | Read the selected node's files, symbols, dependencies, and evidence. |

## A good first visit

1. Read the summary counts.
2. Choose the scope you want.
3. Select a large or central node.
4. Read the short card on the right.
5. Open inspection only when you need source details.

The short card is intentionally small. It tells you what you selected and gives you the next useful action. It does not show every ID or every file.

## Scope is part of the question

If the viewer says **All scopes**, the graph may contain several analyzer results. If you need to understand one language, choose its individual scope.

Example:

- **Go scope**: “What does the Go analyzer know?”
- **TypeScript scope**: “What does the TypeScript analyzer know?”
- **All scopes**: “What can the combined report show?”

If an item is not visible in the selected scope, the viewer explains that instead of pretending the item was found.

## Local-first behavior

When you open a project, the server reads the project locally. When you open a self-contained export, the data is already inside the HTML file. No live server is needed for that export.
