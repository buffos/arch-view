const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "numeric_input.js")).href).then(({ readNumericInput }) => {
  for (const value of ["1.5", "2oops", "Infinity", "9007199254740992"]) {
    assert.throws(() => readNumericInput({ value }, 2, true));
  }
  assert.equal(readNumericInput({ value: "7" }, 2, true), 7);
  assert.equal(readNumericInput({ value: "1e3" }, 2, true), 1000);
  assert.equal(readNumericInput({ value: "" }, 2, true), 2);
  assert.equal(readNumericInput({ value: "1.5" }, 0), 1.5);
  assert.throws(() => readNumericInput({ value: "0", min: "1" }, 2, true));
  assert.throws(() => readNumericInput({ value: "81", max: "80" }, 0, true));
  assert.throws(() => readNumericInput({ value: "", validity: { valid: false }, validationMessage: "Invalid numeric input" }, 2, true), /Invalid numeric input/);
}).catch((error) => { console.error(error); process.exitCode = 1; });
