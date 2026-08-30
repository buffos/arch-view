# Inspect a node

Select a node or group and choose **Open inspection**.

The inspection page is for questions that need more detail:

- Which files belong to this item?
- Which symbols were found?
- Which modules does it depend on?
- What evidence supports the result?
- Which quality findings affect it?

## Overview

Overview gives the short human summary:

- name and kind;
- language;
- hierarchy;
- active scope;
- useful counts;
- non-neutral statuses;
- quality summary.

It does not show long IDs, hashes, or a giant file list.

## Structure

Structure shows how the item is organized and how it relates to its children. Use it when the question is “what is inside this module or group?”

## Files

Files are loaded in bounded pages. The page does not request thousands of files just because you opened one node.

Each file can show:

- repository-relative path;
- language;
- line count and byte count;
- analysis status;
- source location when one is known.

## Symbols

Symbols are declarations such as functions, types, constants, methods, and classes.

Use the search field to find a symbol. Use the kind filter to show only a kind such as a Go function or Go struct.

The filter stays visible when there are no matches. Clear the filter to see the full list again.

## Dependencies

Dependencies show reported outgoing and incoming relationships. Read the direction carefully:

~~~text
A -> B
~~~

means “A depends on B.”

## Evidence

Evidence is shown only when the report links a result to a source fact, file, symbol, relationship, or metric.

There are two different cases:

- **Line-level source evidence**: the report names a file and a readable line range, such as Lines 13–48.
- **File provenance only**: the report names a file that was involved, but it did not provide a trustworthy exact line range.

File provenance is useful for finding the right area. It is not proof that one specific line caused the result.

The viewer must never present “line unavailable” as if it were precise evidence.

## Technical details

IDs, hashes, provider versions, snapshot data, and copy actions belong in Technical details. They are useful for debugging and automation, but they should not interrupt the human explanation.

## Source excerpts

Source excerpts are opt-in. Selecting **View source** is an explicit request to show a bounded, read-only excerpt. If the report has no source content or no safe location, the viewer says so.

An unavailable excerpt is not a failed analysis. It means the report does not contain enough source data for that action.
