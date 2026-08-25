(function () {
  "use strict";

  const modelID = document.querySelector('meta[name="model-id"]').content;
  const state = {
    scene: null,
    selected: null,
    query: "",
    referenceVisibility: "hidden",
    layout: null,
    layoutKey: "",
    layoutRequest: 0,
    layoutError: false
  };

  const elements = {
    projectLabel: document.getElementById("project-label"),
    modelStatus: document.getElementById("model-status"),
    viewDescription: document.getElementById("view-description"),
    summary: document.getElementById("summary"),
    errorBanner: document.getElementById("error-banner"),
    sceneState: document.getElementById("scene-state"),
    layerLegend: document.getElementById("layer-legend"),
    referenceSummary: document.getElementById("reference-summary"),
    graph: document.getElementById("graph-wrap"),
    detailsTitle: document.getElementById("details-title"),
    detailsKind: document.getElementById("details-kind"),
    detailsContent: document.getElementById("details-content"),
    accessibleList: document.getElementById("accessible-list"),
    listCount: document.getElementById("list-count"),
    listToggle: document.getElementById("list-toggle"),
    cycles: document.getElementById("cycles"),
    cycleCount: document.getElementById("cycle-count"),
    diagnostics: document.getElementById("diagnostics"),
    diagnosticCount: document.getElementById("diagnostic-count"),
    footerModelID: document.getElementById("footer-model-id"),
    search: document.getElementById("search"),
    referenceVisibility: document.getElementById("reference-visibility")
  };

  function escapeHTML(value) {
    return String(value == null ? "" : value)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  function truncate(value, length) {
    const text = String(value || "");
    return text.length > length ? text.slice(0, length - 1) + "…" : text;
  }

  function classForState(value) {
    return String(value || "none").replace(/[^a-z0-9_-]/gi, "-");
  }

  function formatList(values, empty) {
    if (!values || values.length === 0) return empty || "None";
    return values.join(", ");
  }

  function showError(message) {
    elements.errorBanner.textContent = message;
    elements.errorBanner.hidden = false;
  }

  function hideError() {
    elements.errorBanner.hidden = true;
    elements.errorBanner.textContent = "";
  }

  async function getJSON(path) {
    const response = await fetch(path, { headers: { Accept: "application/json" } });
    const body = await response.json();
    if (!response.ok) {
      throw new Error(body && body.error && body.error.message ? body.error.message : "The viewer request failed.");
    }
    return body;
  }

  function init() {
    elements.footerModelID.textContent = modelID;
    elements.listToggle.addEventListener("click", function () {
      const pressed = elements.listToggle.getAttribute("aria-pressed") === "true";
      elements.listToggle.setAttribute("aria-pressed", String(!pressed));
      document.body.classList.toggle("list-mode", !pressed);
    });
    elements.search.addEventListener("input", function (event) {
      state.query = event.target.value.trim().toLowerCase();
      renderGraph();
      renderAccessibleList();
    });
    elements.referenceVisibility.addEventListener("change", function (event) {
      state.referenceVisibility = event.target.value;
      state.selected = null;
      loadScene();
    });
    loadScene();
  }

  async function loadScene() {
    try {
      const scenePath = "/v1/models/" + encodeURIComponent(modelID) + "/projection?mode=overview&reference_visibility=" + encodeURIComponent(state.referenceVisibility);
      state.scene = await getJSON(scenePath);
      elements.referenceVisibility.value = state.scene.reference_visibility || state.referenceVisibility;
      state.layout = null;
      state.layoutKey = sceneLayoutKey(state.scene);
      state.layoutError = false;
      state.layoutRequest += 1;
      hideError();
      renderAll();
      void prepareLayout(state.scene);
    } catch (error) {
      elements.projectLabel.textContent = "Unable to load model";
      elements.modelStatus.textContent = "Error";
      showError(error.message || "The local model could not be loaded.");
    }
  }

  function renderAll() {
    const scene = state.scene;
    elements.projectLabel.textContent = scene.project.root_label + " · " + scene.project.language;
    elements.modelStatus.textContent = scene.status;
    elements.viewDescription.textContent = scene.hierarchy_path.length === 0
      ? "A local-first semantic overview. Non-local references remain available through the boundary summary and imports list."
      : "Viewing hierarchy: " + scene.hierarchy_path.join(" / ");
    renderSummary();
    renderSceneState();
    renderLayerLegend();
    renderReferenceSummary();
    renderGraph();
    renderAccessibleList();
    renderSupportLists();
    renderDetails();
  }

  function renderSummary() {
    const summary = state.scene.summary;
    const values = [
      [summary.visible_node_count, "visible nodes"],
      [summary.visible_relationship_count, "directed relations"],
      [summary.module_count, "canonical modules"],
      [summary.cycle_count, "cycles"],
      [summary.diagnostic_count, "diagnostics"],
      [summary.evidence_count, "evidence links"]
    ];
    elements.summary.innerHTML = values.map(function (item) {
      return '<div class="summary-item"><div class="summary-value">' + escapeHTML(item[0]) + '</div><p class="summary-label">' + escapeHTML(item[1]) + "</p></div>";
    }).join("");
  }

  function renderSceneState() {
    const scene = state.scene;
    const pills = [];
    pills.push('<span class="state-pill ' + (scene.status === "complete" ? "ok" : "warning") + '">' + escapeHTML(scene.status) + " model</span>");
    if (scene.cycle_indicators.length) pills.push('<span class="state-pill error">' + escapeHTML(scene.cycle_indicators.length) + " cycle(s)</span>");
    if (scene.diagnostic_indicators.length) pills.push('<span class="state-pill warning">' + escapeHTML(scene.diagnostic_indicators.length) + " diagnostic(s)</span>");
    pills.push('<span class="state-pill">' + escapeHTML(referenceVisibilityLabel(scene.reference_visibility)) + " references</span>");
    if (state.layout && state.layout.key === sceneLayoutKey(scene)) {
      pills.push('<span class="state-pill ok">ELK layered layout</span>');
    } else if (state.layoutError) {
      pills.push('<span class="state-pill warning">deterministic fallback layout</span>');
    }
    elements.sceneState.innerHTML = pills.join("");
  }

  function renderLayerLegend() {
    elements.layerLegend.innerHTML = state.scene.layer_labels.map(function (layer) {
      return '<span class="layer-pill">' + escapeHTML(layer.label) + " · " + escapeHTML(layer.module_ids.length) + " module(s)</span>";
    }).join("");
  }

  function referenceVisibilityLabel(value) {
    return value === "expanded" ? "Expanded" : value === "aggregated" ? "Aggregated" : "Local-first";
  }

  function referenceScopeLabel(value) {
    return String(value || "reference").replaceAll("_", " ");
  }

  function renderReferenceSummary() {
    const summary = state.scene.reference_summary;
    if (!summary || !summary.total) {
      elements.referenceSummary.innerHTML = '<span class="reference-pill">No non-local references in this projection</span>';
      return;
    }
    const scopePills = (summary.by_scope || []).map(function (scope) {
      const visible = scope.visible_node_count ? " · " + scope.visible_node_count + " shown" : "";
      return '<span class="reference-pill"><strong>' + escapeHTML(scope.reference_count) + '</strong> ' + escapeHTML(referenceScopeLabel(scope.scope)) + escapeHTML(visible) + '</span>';
    }).join("");
    const policyText = summary.hidden_count
      ? summary.hidden_count + " hidden in overview"
      : summary.aggregated_count
        ? summary.aggregated_count + " boundary node(s)"
        : summary.expanded_count + " shown individually";
    elements.referenceSummary.innerHTML = '<span class="reference-pill"><strong>' + escapeHTML(summary.relationship_count) + '</strong> import relation(s) · ' + escapeHTML(policyText) + '</span>' + scopePills;
  }

  function nodeMatches(node) {
    if (!state.query) return true;
    return [node.label, node.id, node.kind, node.reference_scope, (node.hierarchy_path || []).join(" ")].join(" ").toLowerCase().includes(state.query);
  }

  function relationshipMatches(relationship, nodesByID) {
    if (!state.query) return true;
    const from = nodesByID[relationship.from_visible_id];
    const to = nodesByID[relationship.to_visible_id];
    return [relationship.id, relationship.type, relationship.target_scope, from && from.label, to && to.label].join(" ").toLowerCase().includes(state.query);
  }

  function sceneLayoutKey(scene) {
    const nodes = scene.visible_nodes.map(function (node) { return node.id; }).sort().join(",");
    const relationships = scene.visible_relationships.map(function (relationship) {
      return relationship.id + ":" + relationship.from_visible_id + ":" + relationship.to_visible_id;
    }).sort().join(",");
    return [scene.model_revision || scene.model_id, (scene.hierarchy_path || []).join("/"), scene.reference_visibility, nodes, relationships].join("|");
  }

  function fallbackLayout(scene) {
    const nodes = scene.visible_nodes;
    const byLayer = {};
    nodes.forEach(function (node) {
      const layer = node.layer == null ? ((node.layers && node.layers[0]) || 0) : node.layer;
      if (!byLayer[layer]) byLayer[layer] = [];
      byLayer[layer].push(node);
    });
    const layers = Object.keys(byLayer).sort(function (left, right) { return Number(left) - Number(right); });
    const nodeWidth = 190;
    const nodeHeight = 82;
    const columnGap = 84;
    const rowGap = 35;
    const maxRows = Math.max.apply(null, layers.map(function (layer) { return byLayer[layer].length; }).concat([1]));
    const width = Math.max(760, layers.length * (nodeWidth + columnGap) + 80);
    const height = Math.max(430, maxRows * (nodeHeight + rowGap) + 100);
    const positions = {};
    layers.forEach(function (layer, layerIndex) {
      byLayer[layer].sort(function (left, right) { return left.label.localeCompare(right.label); });
      byLayer[layer].forEach(function (node, rowIndex) {
        positions[node.id] = {
          x: 40 + layerIndex * (nodeWidth + columnGap),
          y: 42 + rowIndex * (nodeHeight + rowGap),
          width: nodeWidth,
          height: nodeHeight
        };
      });
    });

    return { engine: "fallback", key: sceneLayoutKey(scene), width: width, height: height, positions: positions, edges: {} };
  }

  function buildELKGraph(scene) {
    const nodeWidth = 190;
    const nodeHeight = 82;
    return {
      id: "root",
      layoutOptions: {
        "elk.algorithm": "layered",
        "elk.direction": "RIGHT",
        "elk.edgeRouting": "ORTHOGONAL",
        "elk.spacing.nodeNode": "35",
        "elk.layered.spacing.nodeNodeBetweenLayers": "84"
      },
      children: scene.visible_nodes.map(function (node) {
        return { id: node.id, width: nodeWidth, height: nodeHeight };
      }),
      edges: scene.visible_relationships.map(function (relationship) {
        return {
          id: relationship.id,
          sources: [relationship.from_visible_id],
          targets: [relationship.to_visible_id]
        };
      })
    };
  }

  function numberOrZero(value) {
    return typeof value === "number" && Number.isFinite(value) ? value : 0;
  }

  function pointWithOffset(point, offset) {
    return { x: numberOrZero(point && point.x) + offset, y: numberOrZero(point && point.y) + offset };
  }

  function appendUniquePoint(points, point) {
    if (!point) return;
    const next = { x: numberOrZero(point.x), y: numberOrZero(point.y) };
    const previous = points[points.length - 1];
    if (!previous || previous.x !== next.x || previous.y !== next.y) points.push(next);
  }

  function adaptELKLayout(scene, result, key) {
    const offset = 24;
    const positions = {};
    (result.children || []).forEach(function (child) {
      positions[child.id] = {
        x: numberOrZero(child.x) + offset,
        y: numberOrZero(child.y) + offset,
        width: numberOrZero(child.width) || 190,
        height: numberOrZero(child.height) || 82
      };
    });

    const edges = {};
    (result.edges || []).forEach(function (edge) {
      const points = [];
      (edge.sections || []).forEach(function (section) {
        appendUniquePoint(points, pointWithOffset(section.startPoint, offset));
        (section.bendPoints || []).forEach(function (point) {
          appendUniquePoint(points, pointWithOffset(point, offset));
        });
        appendUniquePoint(points, pointWithOffset(section.endPoint, offset));
      });
      if (points.length < 2) return;
      const midpoint = points[Math.floor(points.length / 2)];
      edges[edge.id] = {
        points: points,
        labelX: midpoint.x,
        labelY: midpoint.y - 7
      };
    });

    const fallback = fallbackLayout(scene);
    const width = Math.max(760, numberOrZero(result.width) + offset * 2, fallback.width);
    const height = Math.max(430, numberOrZero(result.height) + offset * 2, fallback.height);
    return { engine: "elk.layered", key: key, width: width, height: height, positions: positions, edges: edges };
  }

  async function prepareLayout(scene) {
    const key = sceneLayoutKey(scene);
    state.layoutKey = key;
    if (!scene.visible_nodes.length || typeof window.ELK !== "function") return;
    const request = ++state.layoutRequest;
    let elk;
    try {
      elk = new window.ELK({ workerUrl: "/assets/vendor/elk-worker.min.js" });
      const result = await elk.layout(buildELKGraph(scene));
      if (request !== state.layoutRequest || state.scene !== scene) return;
      state.layout = adaptELKLayout(scene, result, key);
      state.layoutError = false;
      renderSceneState();
      renderGraph();
    } catch (error) {
      if (request !== state.layoutRequest || state.scene !== scene) return;
      state.layout = null;
      state.layoutError = true;
      renderSceneState();
      renderGraph();
    } finally {
      if (elk && typeof elk.terminateWorker === "function") elk.terminateWorker();
    }
  }

  function edgeGeometry(relationship, from, to, route) {
    if (route && route.points && route.points.length > 1) {
      const path = "M " + route.points.map(function (point, index) {
        return (index === 0 ? "" : "L ") + point.x + " " + point.y;
      }).join(" ");
      return { path: path, labelX: route.labelX, labelY: route.labelY };
    }
    let path;
    let labelX;
    let labelY;
    if (relationship.from_visible_id === relationship.to_visible_id) {
      const x = from.x + from.width / 2;
      path = "M " + x + " " + from.y + " C " + (x + 100) + " " + (from.y - 55) + ", " + (x + 100) + " " + (from.y + from.height + 55) + ", " + x + " " + (from.y + from.height);
      labelX = x + 50;
      labelY = from.y + from.height / 2;
    } else {
      const x1 = from.x + from.width;
      const y1 = from.y + from.height / 2;
      const x2 = to.x;
      const y2 = to.y + to.height / 2;
      const bend = Math.max(32, Math.abs(x2 - x1) * 0.35);
      path = "M " + x1 + " " + y1 + " C " + (x1 + bend) + " " + y1 + ", " + (x2 - bend) + " " + y2 + ", " + x2 + " " + y2;
      labelX = (x1 + x2) / 2;
      labelY = (y1 + y2) / 2 - 7;
    }
    return { path: path, labelX: labelX, labelY: labelY };
  }

  function renderGraph() {
    const scene = state.scene;
    const nodes = scene.visible_nodes;
    const nodesByID = {};
    nodes.forEach(function (node) { nodesByID[node.id] = node; });
    const fallback = fallbackLayout(scene);
    const activeLayout = state.layout && state.layout.key === sceneLayoutKey(scene) ? state.layout : fallback;
    const positions = Object.assign({}, fallback.positions, activeLayout.positions);
    const width = activeLayout.width;
    const height = activeLayout.height;

    const edgeMarkup = scene.visible_relationships.map(function (relationship) {
      const from = positions[relationship.from_visible_id];
      const to = positions[relationship.to_visible_id];
      if (!from || !to) return "";
      const selected = state.selected && state.selected.kind === "relationship" && state.selected.id === relationship.id;
      const matches = relationshipMatches(relationship, nodesByID);
      const className = "edge-line " + classForState(relationship.cycle_state) + (selected ? " selected" : "") + (!matches ? " dimmed" : "");
      const geometry = edgeGeometry(relationship, from, to, activeLayout.edges[relationship.id]);
      return '<g class="edge-group" data-edge-id="' + escapeHTML(relationship.id) + '" tabindex="0" role="button" aria-label="' + escapeHTML(relationship.accessible_label) + '">' +
        '<path class="edge-hit" d="' + geometry.path + '"></path><path class="' + className + '" d="' + geometry.path + '" marker-end="url(#arrow)"></path>' +
        '<text class="edge-label ' + classForState(relationship.cycle_state) + (!matches ? " dimmed" : "") + '" x="' + geometry.labelX + '" y="' + geometry.labelY + '" text-anchor="middle">' + escapeHTML(relationship.count) + '</text></g>';
    }).join("");

    const nodeMarkup = nodes.map(function (node) {
      const position = positions[node.id];
      const selected = state.selected && state.selected.kind === "node" && state.selected.id === node.id;
      const matches = nodeMatches(node);
      const diagnosticClass = node.diagnostic_state === "none" ? "" : " " + classForState(node.diagnostic_state);
      const className = "node-shape " + classForState(node.kind) + " " + classForState(node.cycle_state) + diagnosticClass + (selected ? " selected" : "") + (!matches ? " dimmed" : "");
      const layer = node.layer == null ? (node.layers && node.layers.length ? "L" + node.layers.join(", L") : "—") : "L" + node.layer;
      const scope = node.reference_scope ? " · " + referenceScopeLabel(node.reference_scope) : "";
      const subtitle = node.kind + scope + " · " + layer + (node.counts.module_count > 1 ? " · " + node.counts.module_count + " modules" : "");
      const status = nodeStatusText(node);
      return '<g data-node-id="' + escapeHTML(node.id) + '" tabindex="0" role="button" aria-label="' + escapeHTML(node.accessible_label) + '">' +
        '<title>' + escapeHTML(node.accessible_label) + '</title>' +
        '<rect class="' + className + '" x="' + position.x + '" y="' + position.y + '" width="' + position.width + '" height="' + position.height + '" rx="12"></rect>' +
        '<rect class="node-hitzone" x="' + position.x + '" y="' + position.y + '" width="' + position.width + '" height="' + position.height + '" rx="12"></rect>' +
        '<text class="node-label ' + (!matches ? "dimmed" : "") + '" x="' + (position.x + 14) + '" y="' + (position.y + 30) + '">' + escapeHTML(truncate(node.label, 25)) + '</text>' +
        '<text class="node-subtitle" x="' + (position.x + 14) + '" y="' + (position.y + 51) + '">' + escapeHTML(truncate(subtitle, 29)) + '</text>' +
        '<text class="node-subtitle" x="' + (position.x + 14) + '" y="' + (position.y + 68) + '">' + escapeHTML(status) + '</text></g>';
    }).join("");

    const empty = nodes.length === 0 ? '<p class="muted">No visible nodes in this projection.</p>' : "";
    elements.graph.innerHTML = empty + '<svg viewBox="0 0 ' + width + ' ' + height + '" role="img" aria-labelledby="graph-title graph-desc" xmlns="http://www.w3.org/2000/svg"><title id="graph-title">Top-level architecture graph</title><desc id="graph-desc">' + escapeHTML(scene.accessibility.reading_order.length + " semantic items in the current scene") + '</desc><defs><marker id="arrow" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto"><path d="M 0 0 L 9 4.5 L 0 9 z" fill="#7483a9"></path></marker></defs><g class="edges">' + edgeMarkup + '</g><g class="nodes">' + nodeMarkup + '</g></svg>';
    elements.graph.querySelectorAll("[data-node-id]").forEach(function (element) {
      const select = function () { selectEntity("node", element.dataset.nodeId); };
      element.addEventListener("click", select);
      element.addEventListener("keydown", function (event) {
        if (event.key === "Enter" || event.key === " ") { event.preventDefault(); select(); }
      });
    });
    elements.graph.querySelectorAll("[data-edge-id]").forEach(function (element) {
      const select = function () { selectEntity("relationship", element.dataset.edgeId); };
      element.addEventListener("click", select);
      element.addEventListener("keydown", function (event) {
        if (event.key === "Enter" || event.key === " ") { event.preventDefault(); select(); }
      });
    });
  }

  function selectEntity(kind, id) {
    state.selected = { kind: kind, id: id };
    renderGraph();
    renderAccessibleList();
    renderDetails();
  }

  function nodeStatusText(node) {
    if (node.counts.internal_relationship_count > 0) {
      return node.counts.internal_relationship_count + " internal relationship" + (node.counts.internal_relationship_count === 1 ? "" : "s");
    }
    const identity = node.identity_state || "stable";
    if (node.confidence_state === "not_applicable") return (node.cycle_state !== "none" ? node.cycle_state + " · " : "") + "identity " + identity;
    return (node.cycle_state !== "none" ? node.cycle_state + " · " : "") + node.confidence_state + " confidence";
  }

  function nodeListMeta(node) {
    const parts = [node.kind, "identity " + (node.identity_state || "stable")];
    if (node.confidence_state !== "not_applicable") parts.push(node.confidence_state + " confidence");
    if (node.counts.internal_relationship_count > 0) parts.push(node.counts.internal_relationship_count + " internal relationship(s)");
    return parts.join(" · ");
  }

  function allListItems() {
    const scene = state.scene;
    return scene.visible_nodes.map(function (node) { return { kind: "node", id: node.id, title: node.label, meta: nodeListMeta(node), value: node }; })
      .concat(scene.visible_relationships.map(function (relationship) { return { kind: "relationship", id: relationship.id, title: relationship.type, meta: relationship.count + " contributor(s) · " + relationship.confidence_state + " confidence", value: relationship }; }))
      .concat((scene.reference_details || []).map(function (reference) { return { kind: "reference-detail", id: reference.id, title: reference.name, meta: referenceScopeLabel(reference.scope) + " · " + reference.count + " import(s) · " + reference.confidence_state + " confidence", value: reference }; }));
  }

  function renderAccessibleList() {
    const items = allListItems().filter(function (item) {
      if (!state.query) return true;
      return [item.title, item.meta, item.id].join(" ").toLowerCase().includes(state.query);
    });
    elements.listCount.textContent = items.length + " item(s)";
    elements.accessibleList.innerHTML = items.length ? items.map(function (item) {
      const selected = state.selected && state.selected.kind === item.kind && state.selected.id === item.id;
      return '<button class="list-item ' + (selected ? "selected" : "") + '" type="button" data-list-kind="' + item.kind + '" data-list-id="' + escapeHTML(item.id) + '"><span class="list-item-title">' + escapeHTML(item.title) + '</span><span class="list-item-meta">' + escapeHTML(item.meta) + '</span></button>';
    }).join("") : '<p class="list-empty">No items match the current search.</p>';
    elements.accessibleList.querySelectorAll("[data-list-id]").forEach(function (element) {
      element.addEventListener("click", function () { selectEntity(element.dataset.listKind, element.dataset.listId); });
    });
  }

  function detailRow(key, value, extraClass) {
    return '<div class="detail-row"><span class="detail-key">' + escapeHTML(key) + '</span><span class="detail-value ' + (extraClass || "") + '">' + escapeHTML(value) + "</span></div>";
  }

  function renderDetails() {
    const scene = state.scene;
    if (!state.selected) {
      elements.detailsTitle.textContent = "Select an item";
      elements.detailsKind.textContent = "Overview";
      elements.detailsContent.innerHTML = '<p class="muted">The graphic and list use the same renderer-neutral scene. Select a node or directed relationship to inspect stable IDs, aggregation counts, layers, uncertainty, and evidence counts.</p>' + '<div class="detail-table">' + detailRow("Hierarchy", scene.hierarchy_path.length ? scene.hierarchy_path.join(" / ") : "Top level") + detailRow("Model status", scene.status, scene.status === "complete" ? "high" : "warning") + detailRow("Language", scene.project.language) + detailRow("Boundary", scene.project.boundary) + '</div>';
      return;
    }
    if (state.selected.kind === "node") {
      const node = scene.visible_nodes.find(function (item) { return item.id === state.selected.id; });
      if (!node) return;
      elements.detailsTitle.textContent = node.label;
      elements.detailsKind.textContent = node.kind;
      elements.detailsContent.innerHTML = '<div class="detail-table">' +
        detailRow("Stable ID", node.id, "emphasis") +
        detailRow("Hierarchy", formatList(node.hierarchy_path, "Top level")) +
        detailRow("Modules", node.counts.module_count) +
        detailRow("Visible relationships", node.counts.relationship_count) +
        (node.counts.internal_relationship_count ? detailRow("Internal relationships", node.counts.internal_relationship_count + " (collapsed in overview)", "emphasis") : "") +
        (node.internal_relationship_ids && node.internal_relationship_ids.length ? detailRow("Internal IDs", formatList(node.internal_relationship_ids)) : "") +
        detailRow("Layer", node.layer == null ? formatList((node.layers || []).map(function (item) { return "Layer " + item; }), "Unassigned") : "Layer " + node.layer) +
        detailRow("Cycle state", node.cycle_state, node.cycle_state !== "none" ? "cycle" : "") +
        detailRow("Diagnostics", node.diagnostic_state, node.diagnostic_state !== "none" ? "warning" : "") +
        detailRow("Identity", node.identity_state || "stable") +
        detailRow("Relationship confidence", node.confidence_state === "not_applicable" ? "Not aggregated for local node" : node.confidence_state, node.confidence_state === "not_applicable" ? "" : node.confidence_state) +
        (node.reference_scope ? detailRow("Reference scope", referenceScopeLabel(node.reference_scope)) : "") +
        detailRow("Evidence", node.counts.evidence_count) +
        detailRow("Tags", formatList(node.tags)) +
        '</div>';
      return;
    }
    if (state.selected.kind === "reference-detail") {
      const reference = (scene.reference_details || []).find(function (item) { return item.id === state.selected.id; });
      if (!reference) return;
      elements.detailsTitle.textContent = reference.name;
      elements.detailsKind.textContent = "Import detail";
      elements.detailsContent.innerHTML = '<div class="detail-table">' +
        detailRow("Stable ID", reference.id, "emphasis") +
        detailRow("Reference scope", referenceScopeLabel(reference.scope)) +
        detailRow("Imported by", formatList(reference.from_visible_ids)) +
        detailRow("Import count", reference.count) +
        detailRow("Confidence", reference.confidence_state, reference.confidence_state) +
        detailRow("Basis", reference.confidence_basis || "Not supplied") +
        detailRow("Canonical IDs", formatList(reference.relationship_ids)) +
        detailRow("Evidence IDs", formatList(reference.evidence_ids)) +
        '</div>';
      return;
    }
    const relationship = scene.visible_relationships.find(function (item) { return item.id === state.selected.id; });
    if (!relationship) return;
    const nodesByID = {};
    scene.visible_nodes.forEach(function (node) { nodesByID[node.id] = node; });
    elements.detailsTitle.textContent = relationship.type;
    elements.detailsKind.textContent = "Directed relation";
    elements.detailsContent.innerHTML = '<div class="detail-table">' +
      detailRow("Stable ID", relationship.id, "emphasis") +
      detailRow("Direction", (nodesByID[relationship.from_visible_id] || {}).label + " → " + (nodesByID[relationship.to_visible_id] || {}).label, "emphasis") +
      detailRow("Type", relationship.type) +
      detailRow("Contributors", relationship.count) +
      detailRow("Canonical IDs", formatList(relationship.contributor_relationship_ids)) +
      detailRow("Cycle state", relationship.cycle_state, relationship.cycle_state !== "none" ? "cycle" : "") +
      detailRow("Confidence", relationship.confidence_state, relationship.confidence_state) +
      detailRow("Basis", relationship.confidence_basis || "Not supplied") +
      (relationship.target_scope ? detailRow("Target scope", referenceScopeLabel(relationship.target_scope)) : "") +
      detailRow("Evidence", relationship.evidence_ids.length) +
      '</div>';
  }

  function renderSupportLists() {
    const scene = state.scene;
    elements.cycleCount.textContent = scene.cycle_indicators.length;
    elements.diagnosticCount.textContent = scene.diagnostic_indicators.length;
    elements.cycles.innerHTML = scene.cycle_indicators.length ? scene.cycle_indicators.map(function (cycle) {
      return '<div class="support-item cycle"><strong>' + escapeHTML(cycle.label) + '</strong><br><span>' + escapeHTML(cycle.module_ids.length) + ' module(s) · ' + escapeHTML(cycle.relationship_ids.length) + ' relationship(s)</span></div>';
    }).join("") : '<p class="muted">No cycles are present in this model.</p>';
    elements.diagnostics.innerHTML = scene.diagnostic_indicators.length ? scene.diagnostic_indicators.map(function (diagnostic) {
      return '<div class="support-item ' + classForState(diagnostic.severity) + '"><strong>' + escapeHTML(diagnostic.code) + '</strong><br><span>' + escapeHTML(diagnostic.message) + '</span>' + (diagnostic.path ? '<br><code>' + escapeHTML(diagnostic.path) + '</code>' : "") + '</div>';
    }).join("") : '<p class="muted">No diagnostics are attached to this model.</p>';
  }

  document.addEventListener("DOMContentLoaded", init);
}());
