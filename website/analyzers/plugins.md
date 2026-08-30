# External analyzer plugins

A plugin is an analyzer that Arch View loads from a descriptor. It can add language support without changing the built-in command.

## List plugins

~~~powershell
go run ./cmd/arch-view analyzers --plugin plugins/python-analyzer/external-plugin.json
~~~

## Use a plugin for analysis

~~~powershell
go run ./cmd/arch-view analyze --project . --plugin plugins/python-analyzer/external-plugin.json --output analysis.json
~~~

## Trust

An explicitly supplied local descriptor is not automatically trusted. Use **--allow-untrusted-plugin** only when you inspected and trust the descriptor.

~~~powershell
go run ./cmd/arch-view analyze --project . --plugin local-analyzer.json --allow-untrusted-plugin --output analysis.json
~~~

The plugin protocol, capabilities, and options are part of the report. If a plugin does not provide a capability, checks that need it may be unsupported.
