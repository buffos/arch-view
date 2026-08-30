# Findings and coverage

A finding is one result attached to one rule and one subject.

## How to read a finding

Read it in this order:

1. Rule name.
2. Exact check or advisory signal.
3. Subject: file, symbol, module, or relationship.
4. Observed value.
5. Configured limit or trigger.
6. Evidence and location.
7. Limitation, if it is a signal.

Example:

~~~text
Module incoming coupling
module orders
observed: 14
configured maximum: 10
~~~

The rule counted 14 different modules that depend on orders. Two relationships from the same module still count as one source module.

## Is a finding good or bad?

The finding itself is not a moral judgment about the code.

An exact result can be useful even when you decide not to change anything. An advisory signal can be useful even when it is a false positive for your design.

Use this decision:

| Question | Action |
| --- | --- |
| Is the result based on the right source and scope? | If not, fix the analysis input first. |
| Is the result expected for this project? | If yes, document the reason and consider a baseline. |
| Is the result an actual design problem? | Change the code or configuration. |
| Is the result only a suggestion? | Review it with a human; do not treat it as proof. |

## Coverage is not a score

Coverage answers “how much of the selected profile could the report evaluate?”

If the viewer says **13 of 25 checks observed**, 13 checks had the data they needed. Ten may be unsupported by the analyzer. Two may need input that this model does not contain.

An unsupported rule is not an implementation claim by itself. It is a statement about the current analyzer/report boundary. The [rule reference](/quality/rules) explains which facts each rule needs.

## Bounded results

Large reports are returned in pages. A message such as “showing the first 25 findings” means more results exist.

Use **Load more**, pagination, the bounded quality query, or the JSON export. The limit protects the browser from rendering thousands of cards at once.

## Evidence states

| State | Meaning |
| --- | --- |
| Precise location | A file and line range support the result. |
| File provenance only | A file was involved, but no trustworthy line range was reported. |
| Partial | Some supporting evidence exists, but it is incomplete. |
| Unavailable | The report does not contain the requested evidence. |

File provenance is useful navigation information. It is not proof of one exact line.
