# Layout option reference

Arch View uses a pinned layout catalog. The catalog contains many options known by the layout engine, but the viewer safely exposes only a smaller editable set.

## How to choose a setting

1. Pick a layout algorithm.
2. Change one setting.
3. Fit the graph.
4. Decide whether the result is easier to read.
5. Keep the setting only if it helps.

## Option fields

| Field | Meaning |
| --- | --- |
| Name | Human label for the option. |
| Type | Boolean, number, text, enum, or structured value. |
| Default | The value used when you do not set it. |
| Algorithms | Algorithms that can use it. |
| Target | The graph object it applies to. |
| Editable | Whether this viewer can change it safely. |
| Renderer support | Whether the current renderer supports the option. |

## Editable examples

| Setting | Example | Why you might change it |
| --- | --- | --- |
| Direction | DOWN | Read the graph from top to bottom. |
| Edge Routing | ORTHOGONAL | Make arrows follow horizontal and vertical segments. |
| Node Spacing | 60 | Give crowded nodes more room. |
| Layered Thoroughness | 10 | Spend more effort on a layered arrangement. |
| Random Seed | 7 | Make a randomized view repeatable. |

## Catalog-only options

A catalog-only option is documented because the pinned engine knows it. It is not a promise that the current viewer can edit it. The viewer marks it as catalog-only or unsupported and rejects unsafe changes.

The [complete generated catalog](/reference/layout-options.generated) contains every known option, grouped by algorithm and category, with type, default, targets, and support state.
