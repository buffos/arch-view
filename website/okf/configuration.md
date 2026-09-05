# OKF configuration

OKF source files are read-only, but a project-backed viewer can save presentation configuration in `.archview.json`.

## Separate layout settings

Architecture and OKF settings share the same layout controls but have separate storage:

| Viewer | Stored in |
| --- | --- |
| Architecture | The top-level `layout` section. |
| OKF | The selected profile's `layout` inside the `okf` section. |

Changing an OKF layout never changes the architecture layout. The OKF layout includes the selected algorithm, options, and enabled advanced renderer features.

The defaults are:

- Architecture: ELK Layered.
- OKF Neutral and Fog of War: ELK Layered, with Junctions and Presentation ports enabled by default.
- Advanced renderer features: disabled until explicitly enabled and saved.

Advanced features are currently verified for Layered. Selecting Mr. Tree or another algorithm remains valid for ordinary layout, but incompatible advanced features must be resolved before they can be applied.

## Drafts and saves

**Apply** changes the current session draft and reruns the layout. It does not write the project file.

**Save** validates the profile and writes the project configuration. **Save As** creates a project-local copy, which is required for a built-in profile. Saving preserves existing layout, analysis, unknown, and extension fields.

Writes use revision checks, idempotency keys, validation, and atomic replacement. A failed write leaves the previous configuration intact.

Profiles can also be bound to a selected bundle. Renaming, deleting, or replacing a profile updates bindings only after validation.

## Session limitations

Model-only sessions do not have a project configuration to update. The current self-contained architecture HTML export also does not expose project-backed OKF persistence. Use the local project viewer when you need to discover bundles, edit profiles, or save settings.
