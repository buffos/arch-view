# What is Arch View?

Arch View is a read-only tool for making a map of a software project.

It answers questions such as:

- Which parts of the project exist?
- Which part uses another part?
- Which files belong to a module?
- Which public symbols have documentation?
- Which checks passed, failed, or could not run?

It does not rewrite your source code. It reports what it can observe.

## A small example

Imagine this project:

~~~text
shop/
  web/
  orders/
  payments/
~~~

If web calls orders, and orders calls payments, Arch View can show:

~~~text
web  ->  orders  ->  payments
~~~

The arrow means “the first module has a reported dependency on the second module.” It does not mean that every line of code calls the other module.

## What Arch View does not claim

Static analysis has limits. A result can be incomplete when a language feature is dynamic or when the analyzer does not have enough information.

For that reason, the viewer distinguishes between:

- **Observed**: the analyzer reported the fact.
- **Partial**: some of the fact was available, but some was missing.
- **Unsupported**: this analyzer does not provide the fact.
- **Not evaluable**: the fact exists in principle, but the current report cannot calculate it.

These labels are important. “Unsupported” is not the same as “the code is wrong.”

## Two audiences

The terminal commands are useful for automation and coding agents. The viewer is for people who want to understand the result without reading a large JSON file.

Use the [CLI reference](/cli/overview) when you want a repeatable command. Use the [viewer guide](/viewer/overview) when you want to explore.
