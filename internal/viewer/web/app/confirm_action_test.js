const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "confirm_action.js")).href).then(async ({ confirmAction }) => {
  class Dialog extends EventTarget {
    constructor() {
      super();
      this.buttons = [new EventTarget(), new EventTarget()];
      this.heading = {};
      this.message = {};
    }
    setAttribute() {}
    querySelector(selector) { return selector === "h3" ? this.heading : this.message; }
    querySelectorAll() { return this.buttons; }
    showModal() { this.open = true; }
    close(value) { this.returnValue = value; this.open = false; this.dispatchEvent(new Event("close")); }
    remove() { this.removed = true; }
  }
  let dialog;
  global.document = {
    createElement: () => (dialog = new Dialog()),
    body: { append: () => {} }
  };
  global.confirm = () => { throw new Error("Native dialogs unavailable"); };
  const question = { title: "Delete?", message: '<unsafe id="value">', confirmLabel: "Delete profile" };
  for (const action of ["confirm", "cancel", "escape"]) {
    const answer = confirmAction(question);
    assert.equal(dialog.message.textContent, question.message, "untrusted values remain text");
    assert.equal(dialog.buttons[0].autofocus, true, "Cancel is the default action");
    if (action === "escape") dialog.dispatchEvent(new Event("cancel", { cancelable: true }));
    else dialog.buttons[action === "confirm" ? 1 : 0].dispatchEvent(new Event("click"));
    assert.equal(await answer, action === "confirm");
    assert.equal(dialog.removed, true);
  }
  Dialog.prototype.showModal = () => { throw new Error("Dialog unavailable"); };
  await assert.rejects(confirmAction(question), /Dialog unavailable/);
  assert.equal(dialog.removed, true);
  delete global.document;
  delete global.confirm;
}).catch((error) => { console.error(error); process.exitCode = 1; });
