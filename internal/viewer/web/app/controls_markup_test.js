const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const markup = fs.readFileSync(path.join(__dirname, "..", "index.html"), "utf8");

assert.match(markup, /class="hero-controls"/);
assert.match(markup, /data-control-section="view"/);
assert.match(markup, /data-control-section="quality"/);
assert.match(markup, /data-control-section="actions"/);
assert.match(markup, /id="quality-profile-configure"/);
assert.match(markup, /id="quality-profile-configure" class="button secondary"/);
assert.match(markup, /id="quality-profile-save"/);
assert.match(markup, /id="quality-profile-save-as"/);
assert.doesNotMatch(markup, /class="hero-actions"/);
