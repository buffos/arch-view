# Shared CSS and SVG download follow-up

Scope: user-requested dialog/form CSS consolidation and OKF Download SVG.
This does not close issues or replace their human visual-review gates.

- Shared dialog frames, shells, headers, footers, action rows, form surfaces,
  OKF label/control bases, and common narrow-screen dialog rules now live in
  `04-form-dialog.css`. Existing class names remain. Feature modules retain
  different sizes, spacing, and layouts. Both stylesheet entrypoints use the
  same ordered module list, including OKF.
- OKF uses the existing Download SVG button and `downloadCurrentSVG` function.
  The shared serializer handles both viewport groups, removes only the export
  copy's pan/zoom transform, retains node positions and local marker references,
  and captures computed presentation styles instead of maintaining copied graph
  CSS. The standalone canvas uses the theme background token.
- Added tests for both scene types, padded geometry and fallback bounds,
  unchanged live transforms, styling, marker preservation, semantic links
  without arrows, shared stylesheet ownership, and bundle ordering.

Live evidence: the opt-in fixture produced actual Chrome downloads for Neutral
OKF, custom OKF, and architecture. XML parsing confirmed valid SVG; the custom
OKF download retained two ellipses with `rgb(17, 34, 51)` fills, and both viewer
downloads omitted the viewport transform. Browser download-event observation
timed out, so success was established from the files actually downloaded, not
the event helper. OKF profile and architecture layout dialogs were inspected
visually. Computed header, footer, and action styles matched between architecture
and quality dialogs. No mobile visual review was claimed.

Browser regression suite: 42 test files passed. All files changed in this
follow-up are below 600 lines. Temporary browser tabs and fixture server were
closed. No source bundle or saved project configuration was changed by the
verification interactions.

Final verification: `go test ./... -count=1`, `go test -race ./...`,
`go vet ./...`, `go build ./...`, JavaScript syntax checks, and
`git diff --check` passed.
