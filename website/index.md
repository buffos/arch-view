---
layout: home
title: Understand your codebase at a glance
titleTemplate: false
editLink: false
hero:
  name: Arch View
  text: Understand your codebase at a glance
  tagline: A local-first map of modules, dependencies, source facts, and code-quality signals.
  actions:
    - theme: brand
      text: Start here
      link: /guide/quick-start
    - theme: alt
      text: Open the demo
      link: /demo
features:
  - title: See the shape of your code
    details: Find the modules in a project and see how they depend on one another.
    link: /viewer/read-the-graph
  - title: Ask simple questions
    details: Inspect a module, its files, its symbols, and the evidence behind a result.
    link: /viewer/inspection
  - title: Check what matters
    details: Run exact checks and useful review signals with a profile you control.
    link: /quality/overview
---

## The short version

Arch View reads a project without changing it. It turns the result into a map that you can open in a browser.

You can use it in two ways:

- Run a command in a terminal and create a report.
- Open the viewer and explore the report as a person.

The viewer is local-first. Your source does not need to leave your machine.

## Start with one command

From the root of a Go project:

~~~powershell
go run github.com/buffo/arch-view/cmd/arch-view@latest open --project . --port 0
~~~

For a checkout of this repository, use:

~~~powershell
go run ./cmd/arch-view open --project . --port 0
~~~

The command starts a local web server and prints the address to open. If you want to understand every part of that command, read [Quick start](/guide/quick-start).

## What this site explains

This is a human guide, not an internal code dump. It explains what a result means, why a check can appear, and what you can do next.

Every command and setting has examples. Technical IDs are kept in secondary details so they are available when needed without becoming the main language of the guide.
