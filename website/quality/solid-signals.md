# Technical SOLID signal rules

This page describes the rules that Arch View currently runs. It is more
technical than the [plain-language rule reference](/quality/rules), but the
main idea is simple:

> A SOLID signal is an observed code shape that deserves a design review. It
> is not proof that the code breaks a SOLID principle.

## What the evaluator does

For each source snapshot and each symbol that has the required facts, the
quality engine:

1. reads structural metrics reported by the analyzer;
2. checks the metrics against the rule's thresholds;
3. emits a finding when the formula is true; and
4. includes the metric facts, source span, rule version, and limitation.

The five rules require the capability source:solid.structure. The current
built-in Go analyzer reports this capability and the metrics described below.
Another analyzer must report the same capability and compatible metrics before
these rules can be evaluated for that language. Otherwise the report says
unsupported or not evaluable; that is not a clean result.

All five comparisons use an inclusive lower bound: observed value >= threshold.
The thresholds are not percentages and they are not scores.

Every result has:

~~~text
assessment_kind = signal
~~~

That label is intentional. Static syntax facts cannot show responsibility,
extension intent, runtime behavior, client usage, or the design decision behind
an abstraction.

## Structural metrics

The quality rules consume namespaced metric IDs. These are the facts that the
current structural provider can publish:

| Metric | Meaning in the current implementation | Used by |
| --- | --- | --- |
| source:solid.member_count | Reported members of a symbol. For a Go struct this is fields plus methods; for a Go interface it is interface entries. | SRP |
| source:solid.method_count | Methods associated with a Go type. It is published, but is not currently a trigger for one of the five signals. | None directly |
| source:solid.dependency_count | Number of distinct dependency names recognized in the syntax examined by the analyzer. Built-in types are ignored. | SRP |
| source:solid.concrete_dependency_count | Recognized dependencies that are not identified as interface types by the analyzer. | DIP |
| source:solid.interface_method_count | Method entries reported for a Go interface. | ISP |
| source:solid.type_switch_count | Number of Go type-switch syntax nodes in a function body. | OCP |
| source:solid.hierarchy_depth | Longest Go type-embedding chain: 0 for no embedding, 1 for direct embedding, 2 for two levels, and so on. | LSP |
| source:solid.derived_type_count | Number of type definitions in the parsed file that directly embed the subject type. | LSP |
| source:solid.abstraction_count | Extractor-reported abstraction count. It is available as a fact, but does not currently trigger one of these five rules. | None directly |

The Go implementation is syntax-based. It does not run the target program and
does not use full type-checker or runtime information to decide whether a
design is good or bad. Some dependencies, embeddings, or relationships may
therefore be outside the metric's scope.

## SRP — Single Responsibility Principle

Rule ID: signal:solid.srp

### Exact trigger

~~~text
member_count >= member_threshold
AND
dependency_count >= dependency_threshold
~~~

Default thresholds:

~~~text
member_threshold     = 10
dependency_threshold = 3
~~~

Both conditions must be true for the same symbol. If a symbol has 12 members
but only 2 dependencies, this rule does not report it. If it has 8 members and
4 dependencies, it also does not report it.

### Example

~~~text
OrderService
  members:      12
  dependencies:  4

12 >= 10 and 4 >= 3  -> signal emitted
~~~

The result says that the shape may contain several responsibilities. It does
not say which responsibilities they are, or that they are unrelated. A data
structure with many fields and a small, coherent API can be a valid design.

### Configuration

~~~json
{
  "rule_id": "signal:solid.srp",
  "parameters": {
    "namespace": "rule-config:solid-signal",
    "schema_version": "1.0.0",
    "payload": {
      "member_threshold": 15,
      "dependency_threshold": 5
    }
  }
}
~~~

## OCP — Open/Closed Principle

Rule ID: signal:solid.ocp

### Exact trigger

~~~text
type_switch_count >= type_switch_threshold
~~~

Default threshold:

~~~text
type_switch_threshold = 1
~~~

This means that one observed Go type switch is enough to produce a signal with
the default profile. The signal does not require several switches, even
though the human description refers to repeated type-switch structure.

### Example

~~~go
switch value := input.(type) {
case Request:
    handleRequest(value)
case Response:
    handleResponse(value)
}
~~~

The analyzer counts one type-switch syntax node. With the default threshold,
the signal is emitted. The review question is whether adding another case
requires editing a central decision point, or whether the switch is the
clearest design for this code.

### Configuration

~~~json
{
  "rule_id": "signal:solid.ocp",
  "parameters": {
    "namespace": "rule-config:solid-signal",
    "schema_version": "1.0.0",
    "payload": {
      "type_switch_threshold": 3
    }
  }
}
~~~

The current metric counts syntax nodes. It does not determine whether a type
switch is an intentional, stable boundary or an extension point.

## LSP — Liskov Substitution Principle

Rule ID: signal:solid.lsp

### Exact trigger

~~~text
hierarchy_depth >= hierarchy_depth_threshold
AND
derived_type_count >= derived_type_threshold
~~~

Default thresholds:

~~~text
hierarchy_depth_threshold = 2
derived_type_threshold    = 1
~~~

In Go, the current implementation uses type embedding as the structural shape
for this signal. It does not claim that embedded types are class subclasses.

### Example

~~~text
Base
  <- Middle      (embeds Base)
       <- Final  (embeds Middle)

Final's hierarchy depth: 2
Base's direct derived-type count: 1
~~~

If the required metrics meet their thresholds, the signal asks for a review of
contracts and substitutability. It cannot inspect whether replacing one value
with another preserves behavior, error guarantees, or invariants.

The derived-type count is currently limited to type definitions found in the
same parsed source file. It is not a whole-repository inheritance graph.

### Configuration

~~~json
{
  "rule_id": "signal:solid.lsp",
  "parameters": {
    "namespace": "rule-config:solid-signal",
    "schema_version": "1.0.0",
    "payload": {
      "hierarchy_depth_threshold": 3,
      "derived_type_threshold": 2
    }
  }
}
~~~

## ISP — Interface Segregation Principle

Rule ID: signal:solid.isp

### Exact trigger

~~~text
interface_method_count >= interface_method_threshold
~~~

Default threshold:

~~~text
interface_method_threshold = 8
~~~

### Example

~~~go
type Repository interface {
    Read()
    Write()
    Delete()
    Search()
    Count()
    Begin()
    Commit()
    Rollback()
}
~~~

This interface has eight reported method entries, so the default signal is
emitted. The signal asks whether every client really needs the whole
interface. It does not know which methods individual clients call, and a wide
interface can be appropriate at a deliberate boundary.

### Configuration

~~~json
{
  "rule_id": "signal:solid.isp",
  "parameters": {
    "namespace": "rule-config:solid-signal",
    "schema_version": "1.0.0",
    "payload": {
      "interface_method_threshold": 12
    }
  }
}
~~~

## DIP — Dependency Inversion Principle

Rule ID: signal:solid.dip

### Exact trigger

~~~text
concrete_dependency_count >= concrete_dependency_threshold
~~~

Default threshold:

~~~text
concrete_dependency_threshold = 3
~~~

The current provider counts distinct recognized dependencies that it does not
identify as interfaces. It is a structural count, not a proof that a function
constructs those types or that the dependency should be inverted.

### Example

~~~text
BuildResult
  recognized concrete dependencies: 4
  recognized interface dependencies: 1

4 >= 3  -> signal emitted
~~~

The useful review question is whether the subject is tied to concrete details
that could be hidden behind a smaller abstraction. The finding can be a good,
real observation while the conclusion “this violates DIP” is still unknown.

### Configuration

~~~json
{
  "rule_id": "signal:solid.dip",
  "parameters": {
    "namespace": "rule-config:solid-signal",
    "schema_version": "1.0.0",
    "payload": {
      "concrete_dependency_threshold": 5
    }
  }
}
~~~

## Threshold fields and precedence

The profile schema accepts these non-negative integer fields:

| Field | Effective use |
| --- | --- |
| member_threshold | SRP member count |
| dependency_threshold | SRP dependency count |
| type_switch_threshold | OCP type-switch count |
| hierarchy_depth_threshold | LSP hierarchy depth |
| derived_type_threshold | LSP derived-type count |
| interface_method_threshold | ISP interface method count |
| concrete_dependency_threshold | DIP concrete dependency count |
| threshold | Convenience alias for the primary metric of the selected rule |

The generic threshold alias maps to member_threshold for SRP,
type_switch_threshold for OCP, hierarchy_depth_threshold for LSP,
interface_method_threshold for ISP, and concrete_dependency_threshold for
DIP. If both the generic alias and the corresponding rule-specific field are
present, the generic alias currently wins for that primary metric. Use the
rule-specific fields to make the configuration unambiguous.

minimum is accepted by the current parameter schema for compatibility, but the
evaluator does not currently use it in a trigger. Do not use minimum to
configure a SOLID threshold; use the rule-specific field or threshold.

## Coverage states

The rule result also reports whether the calculation had enough input:

| State | Meaning |
| --- | --- |
| observed | The required capability and all metrics needed for a subject were reported; the formula either triggered or stayed below its threshold. |
| partial | Structural subjects were present, but only some had the complete metric set. |
| not_evaluable | The capability was present, but no complete metric set was available for this rule. |
| unsupported | The source snapshot did not provide the required source:solid.structure capability, or explicitly marked it unavailable. |

These states are about analysis coverage. observed with no finding means the
observed values did not meet the formula. unsupported does not mean the code
passed the rule.

## How to review a signal

When a SOLID signal appears:

1. read the exact metric values and thresholds in the finding;
2. open the source span and inspect the actual code;
3. decide whether the shape represents a real design problem in this project;
4. fix it if it is a problem; or
5. baseline it with a specific reason if the shape is intentional.

Do not baseline a signal merely because the analyzer reported it. Also do not
call it a false positive just because it is advisory: the measured structural
fact can be correct even when the design is acceptable.
