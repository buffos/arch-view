const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const read = (name) => fs.readFileSync(path.join(__dirname, "../styles", name), "utf8");
const shared = read("04-form-dialog.css");
const layout = read("10-settings.css");
const quality = read("13-quality-profile.css");
const okf = read("14-okf.css");

for (const css of [layout, quality]) {
  assert.doesNotMatch(css, /box-shadow: 0 28px 90px/);
  assert.doesNotMatch(css, /\.(?:layout-settings|quality-profile)-footer\s*\{[^}]*padding-top:/);
  assert.doesNotMatch(css, /backdrop-filter: blur\(3px\)/);
}
assert.match(shared, /\.layout-settings-footer,\s*\.quality-profile-footer\s*\{/);
assert.match(shared, /\.layout-settings-actions,\s*\.quality-profile-actions,\s*\.okf-editor-actions/);
assert.match(shared, /\.settings-field input,[\s\S]*\.okf-form-section textarea\s*\{/);
assert.match(shared, /@media \(max-width: 680px\)/);
assert.doesNotMatch(okf, /#download-svg/);
assert.doesNotMatch(okf, /marker(?:-start|-mid|-end)?\s*:/);

const entry = fs.readFileSync(path.join(__dirname, "../styles.css"), "utf8");
const assets = fs.readFileSync(path.join(__dirname, "../../assets.go"), "utf8");
const imports = [...entry.matchAll(/@import url\("styles\/([^"]+)"\)/g)].map((match) => match[1]);
const modules = [...assets.matchAll(/^\s*"([\w-]+\.css)",/gm)].map((match) => match[1]);
assert.deepEqual(imports, modules, "stylesheet entrypoints must have identical ordering");
assert.equal(new Set(modules).size, modules.length, "each stylesheet is bundled only once");
