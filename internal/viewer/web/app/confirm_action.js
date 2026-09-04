// Use a DOM dialog so embedded browsers do not depend on native confirm().
export function confirmAction({ title, message, confirmLabel }) {
  const dialog = document.createElement("dialog");
  dialog.className = "layout-settings-dialog";
  dialog.setAttribute("aria-label", title);
  dialog.innerHTML = '<div class="layout-settings-shell"><header class="layout-settings-header"><h3></h3></header>'
    + '<p></p><footer class="layout-settings-footer"><button type="button" class="button secondary">Cancel</button>'
    + '<button type="button" class="button"></button></footer></div>';
  dialog.querySelector("h3").textContent = title;
  dialog.querySelector("p").textContent = message;
  const [cancel, confirm] = dialog.querySelectorAll("button");
  confirm.textContent = confirmLabel;
  cancel.autofocus = true;
  return new Promise((resolve, reject) => {
    cancel.addEventListener("click", () => dialog.close("cancelled"));
    confirm.addEventListener("click", () => dialog.close("confirmed"));
    dialog.addEventListener("cancel", (event) => {
      event.preventDefault();
      dialog.close("cancelled");
    });
    dialog.addEventListener("close", () => {
      dialog.remove();
      resolve(dialog.returnValue === "confirmed");
    }, { once: true });
    document.body.append(dialog);
    try { dialog.showModal(); }
    catch (error) { dialog.remove(); reject(error); }
  });
}
