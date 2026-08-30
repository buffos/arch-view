# How Arch View thinks about code

The words in the viewer are easier once the basic ideas are clear.

## Project

The project is the folder Arch View reads. It is the boundary for paths and source files in the report.

Example:

~~~text
Project: payments-service
Root:    C:\work\payments-service
~~~

## Module

A module is a named part of a project. Depending on the language, it may be a package, namespace, crate, directory, or another unit the analyzer can identify.

Example:

~~~text
web -> orders
~~~

Here web and orders are modules.

## Dependency

A dependency says that one module uses another module or an external reference.

~~~text
orders -> database
~~~

This is a reported relationship. It is not a promise that the dependency is always used at runtime.

## Scope

A scope says which part of the project a result belongs to.

For a multi-language project, you may see separate scopes for Go and TypeScript. **All scopes** combines information only when the report provides an explicit combined view.

Changing the scope changes the question you are asking. A finding in one scope should not silently appear in another scope.

## Source fact

A source fact is a small observation extracted from a file. Examples include:

- a file has 420 lines;
- a function body covers 61 lines;
- a symbol is public;
- a symbol has documentation;
- a type has three concrete dependencies.

The viewer loads large source-fact lists on demand so the first screen stays readable.

## Finding

A finding is the result of a quality rule.

Example:

> orders has 14 incoming module targets. The configured maximum is 10.

The number comes from the report. The rule's meaning comes from the profile.

## Baseline

A baseline records findings that you have reviewed and accepted for now.

It does not fix the code. It tells a later run: “do not show these exact, unchanged findings as new work.” Read [Baselines](/quality/baselines) for the safe workflow.

## Evidence

Evidence is the link between a result and the files, symbols, relationships, or measurements that support it.

Precise evidence has a line range. File-only provenance tells you which file was involved, but it does not prove one exact line.

## Technical details

The model contains stable IDs, hashes, provider versions, and snapshot identities. Those values help tools compare reports. They are available in JSON and in the viewer's technical section, but they are not the main explanation for a person.
