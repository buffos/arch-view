import { bootstrap } from "./app/bootstrap.js";
import { selectedView } from "./app/mode_navigation.js";
import { bootstrapOKF } from "./app/okf_bootstrap.js";

document.addEventListener("DOMContentLoaded", function () {
  const meta = document.querySelector('meta[name="okf-enabled"]');
  if (selectedView(window.location.search) === "okf" && meta && meta.content === "true") bootstrapOKF();
  else bootstrap();
});
