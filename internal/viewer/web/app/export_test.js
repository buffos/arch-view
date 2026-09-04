const assert = require("node:assert/strict");
const { pathToFileURL } = require("node:url");
const path = require("node:path");

// Minimal SVG tree for testing serialization without a browser dependency.
class Element {
  constructor(attributes = {}, children = [], computed = {}) {
    this.attributes = { ...attributes };
    this.children = children;
    children.forEach((child) => { child.parent = this; });
    this.computed = computed;
    this.styles = {};
    this.style = { setProperty: (name, value) => { this.styles[name] = value; } };
    this.ownerDocument = { defaultView: { getComputedStyle: (node) => ({
      getPropertyValue: (name) => node.computed[name] || "",
      background: "rgb(9, 14, 29)"
    }) } };
  }
  getAttribute(name) { return this.attributes[name] ?? null; }
  setAttribute(name, value) { this.attributes[name] = value; }
  removeAttribute(name) { delete this.attributes[name]; }
  querySelectorAll(selector) {
    const descendants = this.children.flatMap((child) => [child, ...child.querySelectorAll("*")]);
    if (selector === "*") return descendants;
    return descendants.filter((child) => selector.split(",").some((part) =>
      String(child.attributes.class || "").split(" ").includes(part.trim().slice(1))));
  }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
  cloneNode() { return new Element(this.attributes, this.children.map((child) => child.cloneNode()), this.computed); }
  remove() { this.parent.children = this.parent.children.filter((child) => child !== this); }
}

import(pathToFileURL(path.join(__dirname, "export.js")).href).then(({ serializeSVG, exportBounds, downloadCurrentSVG }) => {
  let serialized;
  global.XMLSerializer = class { serializeToString(copy) { serialized = copy; return "<svg/>"; } };
  try {
    for (const viewportClass of ["viewport-content", "okf-viewport-content"]) {
      const edge = new Element({ class: "containment", "marker-end": "url(#arrow)" }, [], { stroke: "rgb(116, 131, 169)" });
      const semantic = new Element({ class: "semantic-link" }, [], { "stroke-dasharray": "7px, 5px" });
      const node = new Element({ transform: "translate(10 20)" }, [], { fill: "rgb(17, 34, 51)", "font-size": "14px" });
      const label = new Element({ class: "edge-label geometry-edge-label", "data-edge-label-id": "label::e1::count" }, [], { fill: "rgb(216, 226, 255)" });
      const junction = new Element({ class: "edge-junction", "data-junction-id": "junction::1:2" }, [], { stroke: "rgb(110, 231, 249)" });
      const content = new Element({ class: viewportClass, transform: "translate(90 70) scale(2)" }, [edge, semantic, node, label, junction, new Element({ class: "edge-hit" })]);
      content.getBBox = () => ({ x: 10, y: 20, width: 200, height: 100 });
      const source = new Element({ viewBox: "0 0 500 500" }, [content]);
      assert.deepEqual(exportBounds(source), { x: -22, y: -12, width: 264, height: 164 });
      assert.match(serializeSVG(source, exportBounds(source)), /<svg\/>/);
      const exportedContent = serialized.children[0];
      assert.equal(exportedContent.getAttribute("transform"), null);
      assert.equal(content.getAttribute("transform"), "translate(90 70) scale(2)", "live viewport must not change");
      assert.equal(exportedContent.children.length, 5);
      assert.equal(exportedContent.children[0].getAttribute("marker-end"), "url(#arrow)");
      assert.equal(exportedContent.children[1].getAttribute("marker-end"), null);
      assert.equal(exportedContent.children[1].styles["stroke-dasharray"], "7px, 5px");
      assert.equal(exportedContent.children[2].styles.fill, "rgb(17, 34, 51)");
      assert.equal(exportedContent.children[2].getAttribute("transform"), "translate(10 20)");
      assert.equal(exportedContent.children[3].getAttribute("data-edge-label-id"), "label::e1::count");
      assert.equal(exportedContent.children[3].styles.fill, "rgb(216, 226, 255)");
      assert.equal(exportedContent.children[4].getAttribute("data-junction-id"), "junction::1:2");
      assert.equal(exportedContent.children[4].styles.stroke, "rgb(110, 231, 249)");
      assert.equal(serialized.getAttribute("viewBox"), "-22 -12 264 164");
      assert.equal(serialized.styles["background-color"], "#090e1d", "exports remain readable outside the dark viewer page");
      content.getBBox = () => { throw new Error("unmeasurable"); };
      assert.deepEqual(exportBounds(source), { x: 0, y: 0, width: 500, height: 500 });
    }
    assert.equal(downloadCurrentSVG({ elements: { graph: { querySelector: () => null } } }), false);
  } finally { delete global.XMLSerializer; }
}).catch((error) => { console.error(error); process.exitCode = 1; });
