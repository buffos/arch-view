import { escapeHTML } from "./utils.js";
import { PADDING_SIDES } from "./layout_value.js";

function attributes(option, suffix = "") {
  return 'class="layout-option-input" data-layout-option-id="' + escapeHTML(option.id) + '" aria-label="' + escapeHTML(option.name + suffix) + '"';
}
function select(option, value, choices) {
  return "<select " + attributes(option) + '><option value="">Engine default</option>'
    + choices.map((choice) => '<option value="' + escapeHTML(choice) + '"' + (String(value) === String(choice) ? " selected" : "") + ">" + escapeHTML(choice) + "</option>").join("") + "</select>";
}
function numeric(option, value) {
  return '<input type="number" step="' + (option.type === "INT" ? "1" : "any") + '" value="' + escapeHTML(value ?? "")
    + '" placeholder="Engine default"' + (option.minimum == null ? "" : ' min="' + option.minimum + '"')
    + (option.maximum == null ? "" : ' max="' + option.maximum + '"') + " " + attributes(option) + ">";
}
function padding(option, value) {
  return '<div class="layout-settings-controls">' + PADDING_SIDES.map((side) => '<label class="settings-field">' + side + '<input type="number" min="0" max="10000" step="any" data-layout-padding="' + side
    + '" value="' + escapeHTML(value?.[side] ?? "") + '" placeholder="Default" ' + attributes(option, " " + side) + "></label>").join("") + "</div>";
}
const controls = {
  padding,
  ENUM: (option, value) => select(option, value, option.allowed_values || []),
  BOOLEAN: (option, value) => select(option, value, ["true", "false"]),
  INT: numeric, DOUBLE: numeric,
  STRING: (option, value) => '<input type="text" value="' + escapeHTML(value ?? "") + '" placeholder="Engine default" ' + attributes(option) + ">"
};
const parsers = { BOOLEAN: (raw) => raw === "true", INT: Number, DOUBLE: Number };

export function renderOptionControl(option, draft) {
  const control = controls[option.control || option.type];
  return control ? control(option, draft.options?.[option.id]) : '<span class="layout-option-state">Not implemented</span>';
}

export function updateOptionValue(option, input, draft) {
  const raw = input.value;
  if (option.control === "padding") {
    if (raw === "") { delete draft.options[option.id]; return; }
    const value = { top: 0, right: 0, bottom: 0, left: 0, ...draft.options[option.id] };
    value[input.dataset.layoutPadding] = Number(raw);
    draft.options[option.id] = value;
    input.closest?.(".layout-option-control")?.querySelectorAll("[data-layout-padding]").forEach((field) => {
      if (field !== input) field.value = String(value[field.dataset.layoutPadding]);
    });
    return;
  }
  if (raw === "") delete draft.options[option.id];
  else draft.options[option.id] = (parsers[option.type] || String)(raw);
}
