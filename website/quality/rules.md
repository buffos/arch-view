# Quality rule reference

This page explains the checks in plain language. The technical rule identity is included below each heading for people who need to configure JSON or automation.

## File size

**What it checks:** the number of physical lines in each reported file.

**Example:** a file has 620 lines and the limit is 500. The rule reports the file.

**Why it can matter:** very large files can be harder to understand and change. A generated file may be large for a good reason.

**How to respond:** split the file when it contains separate responsibilities. Otherwise, record why it is large or adjust the limit.

Technical key: source:file.max-lines

## Callable size

**What it checks:** the line span of a reported function, method, or other callable body.

**Example:** a function body covers 61 lines and the limit is 50.

**Why it can matter:** long callables often mix several steps.

**How to respond:** separate validation, transformation, and side effects when they are truly different jobs. Do not split code only to make a number smaller.

Technical key: source:callable.max-lines

## Callable complexity

**What it checks:** the complexity number supplied by the analyzer for a callable.

**Example:** a function has complexity 12 and the configured maximum is 10.

**Why it can matter:** more branches create more paths to understand and test.

**How to respond:** simplify conditions, name decisions, or split independent paths. First check how the analyzer defines complexity.

Technical key: source:callable.max-cyclomatic-complexity

## Callable nesting depth

**What it checks:** how deeply a callable nests conditions, loops, and similar blocks.

**Example:** maximum depth is 6 and the limit is 4.

**Why it can matter:** the reader must keep more context in mind.

**How to respond:** use early returns, extract a small helper, or simplify the control flow.

Technical key: source:callable.max-nesting-depth

## Public symbol documentation

**What it checks:** whether a public symbol has documentation reported by the analyzer.

**Example:** the analyzer sees a public function but no attached documentation record.

**Why it can matter:** public code is easier to use when its purpose and inputs are clear.

**How to respond:** add documentation in the language's normal form. If the analyzer says documentation is missing but the source clearly has it, check analyzer support and the symbol's source scope before changing the code.

Technical key: source:public-symbol.documentation

## Module outgoing coupling

**What it checks:** how many different modules this module depends on.

**Example:**

~~~text
orders -> users
orders -> payments
orders -> inventory
~~~

Outgoing coupling for orders is 3. Multiple edges to payments still count as one target module.

**Why it can matter:** changes in many targets can make this module harder to understand and test.

**How to respond:** review whether the module is coordinating too many responsibilities. A high number is not automatically wrong.

Technical key: architecture:module.max-efferent-coupling

## Module incoming coupling

**What it checks:** how many different modules depend on this module.

**Example:**

~~~text
web      -> orders
reports  -> orders
worker   -> orders
~~~

Incoming coupling for orders is 3.

**Why it can matter:** a change in orders may affect many consumers. That can mean the module is important, stable, or too widely shared.

**How to respond:** keep the public boundary stable, add tests around it, or split unrelated responsibilities. Do not remove consumers only to make the number smaller.

Technical key: architecture:module.max-afferent-coupling

## Architecture cycles

**What it checks:** whether the dependency graph contains a loop.

**Example:**

~~~text
web -> orders
orders -> payments
payments -> orders
~~~

Orders and payments form a cycle.

**Why it can matter:** cycles make ownership, startup order, and independent testing harder.

**How to respond:** choose a one-way boundary, move shared data into a smaller module, or invert the dependency through an interface.

Technical key: architecture:no-cycles

## Forbidden dependency

**What it checks:** only dependencies that match a policy you explicitly configured.

**Example:** the policy says web/* must not depend on database. If web/cart depends on database, the rule reports that edge.

**Important:** Arch View does not guess which dependencies are forbidden. No policy means no meaningful forbidden-dependency result.

**How to respond:** either remove the dependency, change the boundary, or update the policy if the dependency is allowed.

Technical key: architecture:forbidden-dependency

## Layer direction

**What it checks:** whether dependencies follow an explicit layer policy.

**Example:** you define that layer 0 may depend on layer 1, but not the other way around. A dependency from layer 1 to layer 0 is reported.

**Important:** the visual position of a node is not enough. The graph needs explicit layer assignments and an explicit direction policy.

**How to respond:** correct the dependency, correct the layer assignment, or revise the architecture policy.

Technical key: architecture:layer-direction

## SOLID signals

SOLID signals are advisory. They report a structural shape that may deserve review. They do not prove a SOLID violation.

For the exact metrics, formulas, default thresholds, and analyzer limitations,
see [Technical SOLID signal rules](/quality/solid-signals).

### Single responsibility signal

Reports a type with both many members and many dependencies. Review whether it has several unrelated jobs.

### Open/closed signal

Reports repeated type-switch structure. Review whether adding a new type requires editing a central switch.

### Liskov substitutability signal

Reports a deep hierarchy with derived types. Review behavior and contracts; the shape alone cannot prove a substitutability problem.

### Interface segregation signal

Reports an interface with many methods. Review whether clients need only small parts of it.

### Dependency inversion signal

Reports a symbol with several concrete dependencies. Review whether it should depend on smaller abstractions.

Example:

~~~text
BuildResult directly creates FileStore, Logger, and NetworkClient.
The configured concrete-dependency threshold is 3.
Result: review signal.
~~~

This is not automatically a false positive. The concrete dependencies are an observed fact. The conclusion “this violates DIP” would require design intent that static analysis cannot see.

Technical keys: signal:solid.srp, signal:solid.ocp, signal:solid.lsp, signal:solid.isp, and signal:solid.dip.

## Rule parameters

Threshold rules normally accept an operator, a limit, and a unit. SOLID signals accept threshold fields such as member threshold, dependency threshold, type-switch threshold, hierarchy-depth threshold, interface-method threshold, or concrete-dependency threshold.

Architecture rules use explicit selectors for modules, tags, layers, or stable-key patterns. They do not infer a policy from names alone.

### Coupling parameters

The coupling rules use these settings:

| Technical key | Plain meaning |
| --- | --- |
| `external_policy` | Decide whether non-local or external modules count. Use `exclude` to count only local modules. Use `include` to count both. |
| `operator` | Decide how to compare the measured number with the limit, such as greater than or less than. |
| `limit` | The number at which the rule changes from okay to reported. |
| `unit` | Confirms what is being counted. For coupling, it is a module. |

### SOLID signal parameters

These are optional thresholds. A signal uses only the fields that apply to its structural question. A larger threshold makes the signal less eager to report.

| Technical key | Plain meaning |
| --- | --- |
| `concrete_dependency_threshold` | How many concrete dependencies can appear before a review signal. |
| `dependency_threshold` | How many dependencies can appear before a review signal. |
| `derived_type_threshold` | How many derived types can appear before a hierarchy signal. |
| `hierarchy_depth_threshold` | How deep a type hierarchy can become before a hierarchy signal. |
| `interface_method_threshold` | How many methods an interface can have before an interface-size signal. |
| `member_threshold` | How many members a type can have before a responsibility signal. |
| `minimum` | A lower bound used by a signal when the profile supplies one. |
| `threshold` | A general cutoff used by a signal when the profile supplies one. |
| `type_switch_threshold` | How many type-switch cases can appear before an extension signal. |

The exact rule result still depends on the capabilities reported by the analyzer. A missing capability is unsupported; it is not a passing result.
