export function createAPI(context) {
  async function getJSON(path) {
    if (context.embeddedExport) return embeddedJSON(path);
    const response = await fetch(path, { headers: { Accept: "application/json" } });
    const body = await response.json();
    if (!response.ok) throw new Error(body && body.error && body.error.message ? body.error.message : "The viewer request failed.");
    return body;
  }

  function embeddedJSON(path) {
    const url = new URL(path, window.location.href);
    const modelPath = "/v1/models/" + encodeURIComponent(context.modelID);
    if (url.pathname === modelPath) return context.embeddedExport.model;
    if (url.pathname === modelPath + "/source-index") {
      if (context.embeddedExport.model && context.embeddedExport.model.source_index) return context.embeddedExport.model.source_index;
      throw new Error("The exported model has no source-facts attachment.");
    }
    if (url.pathname === modelPath + "/source-index/files" || url.pathname === modelPath + "/source-index/symbols" || url.pathname === modelPath + "/source-index/documentation") {
      return embeddedSourcePage(context, url);
    }
    if (url.pathname.indexOf(modelPath + "/source-index/evidence/") === 0) {
      return embeddedSourceEvidence(context, url);
    }
    if (url.pathname === modelPath + "/projection") {
      const pathValue = url.searchParams.getAll("path");
      const visibility = url.searchParams.get("reference_visibility") || "hidden";
      const key = JSON.stringify(pathValue) + "|" + visibility;
      if (context.embeddedExport.scenes && context.embeddedExport.scenes[key]) return context.embeddedExport.scenes[key];
      throw new Error("The exported hierarchy path or reference view is unavailable.");
    }
    if (url.pathname === "/v1/layout/config") return { schema_version: "arch-view.config/v1", layout: { algorithm: "layered", options: {} }, origin: "session", status: "valid", can_save: false, can_save_as: false, diagnostics: [] };
    if (url.pathname === "/v1/layout/options") return { schema_version: "arch-view.config/v1", adapter: { id: "export", version: "embedded", source: "export" }, algorithms: [], categories: [], options: [] };
    throw new Error("The self-contained export does not require network access.");
  }

  async function postJSON(path, value) {
    const response = await fetch(path, { method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify(value) });
    const body = await response.json();
    if (!response.ok) throw new Error(body && body.error && body.error.message ? body.error.message : "The viewer request failed.");
    return body;
  }

  async function putJSON(path, value) {
    const response = await fetch(path, { method: "PUT", headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify(value) });
    const body = await response.json();
    if (!response.ok) throw new Error(body && body.error && body.error.message ? body.error.message : "The viewer request failed.");
    return body;
  }

  function currentModelID() {
    const state = context.state;
    if (state.model && state.model.model_id) return state.model.model_id;
    if (state.scene && state.scene.model_id) return state.scene.model_id;
    return context.modelID;
  }

  return { currentModelID, getJSON, postJSON, putJSON };
}

function embeddedSourceIndex(context) {
  const model = context.embeddedExport && context.embeddedExport.model;
  if (!model || !model.source_index) throw new Error("The exported model has no source-facts attachment.");
  return model.source_index;
}

function embeddedSourceSnapshot(index, url) {
  const requestedScope = url.searchParams.get("scope") || "";
	const requestedSnapshot = url.searchParams.get("snapshot_id") || "";
	if (!requestedSnapshot && (!requestedScope || requestedScope === "all") && index.projection) return index.projection;
  const snapshots = Array.isArray(index.snapshots) ? index.snapshots : [];
	const matches = snapshots.filter(function (snapshot) {
		if (requestedSnapshot && snapshot.snapshot_id !== requestedSnapshot) return false;
		return !requestedScope || requestedScope === "all" || snapshot.scope_context && snapshot.scope_context.scope_id === requestedScope;
	});
	return matches.length === 1 ? matches[0] : null;
}

function embeddedIDs(url, key) {
  return Array.from(new Set(url.searchParams.getAll(key).map(function (value) { return value.trim(); }).filter(Boolean)));
}

function embeddedMembership(snapshot, url) {
  const moduleIDs = new Set(embeddedIDs(url, "module_id"));
  const fileIDs = new Set(embeddedIDs(url, "file_id"));
  if (moduleIDs.size) {
    (snapshot.relations || []).forEach(function (relation) {
      if (relation.category !== "contains" || !relation.from_ref || relation.from_ref.kind !== "module" || !relation.to_ref || relation.to_ref.kind !== "file") return;
      if (moduleIDs.has(relation.from_ref.id)) fileIDs.add(relation.to_ref.id);
    });
  }
  if (!moduleIDs.size && !fileIDs.size) return null;
  return fileIDs;
}

function embeddedSymbolMembership(snapshot, files) {
  if (!files) return null;
  const symbols = new Set();
  (snapshot.relations || []).forEach(function (relation) {
    if ((relation.category !== "contains" && relation.category !== "declares") || !relation.from_ref || relation.from_ref.kind !== "file" || !relation.to_ref || relation.to_ref.kind !== "symbol") return;
    if (files.has(relation.from_ref.id)) symbols.add(relation.to_ref.id);
  });
  return symbols;
}

function embeddedPage(items, snapshot, url) {
	const limitText = url.searchParams.get("limit") || "50";
	if (!/^\d+$/.test(limitText)) throw new Error("The source-facts query limit must be an integer.");
	const limit = Number(limitText);
	if (limit < 1 || limit > 200) throw new Error("The source-facts query limit is outside the allowed bound.");
  let start = 0;
  const cursor = url.searchParams.get("cursor");
  if (cursor) {
    try {
      const decoded = atob(cursor.replace(/-/g, "+").replace(/_/g, "/"));
	  if (!/^v1:\d+$/.test(decoded)) throw new Error("invalid cursor");
	  start = Number(decoded.slice(3));
	} catch (_) { throw new Error("The source-facts query cursor is malformed."); }
  }
	if (start > items.length) throw new Error("The source-facts query cursor is outside the result set.");
  const end = Math.min(items.length, start + limit);
  const next = end < items.length ? btoa("v1:" + end).replace(/=/g, "").replace(/\+/g, "-").replace(/\//g, "_") : "";
  return { snapshot_id: snapshot.snapshot_id, scope_id: snapshot.scope_context && snapshot.scope_context.scope_id || "", items: items.slice(start, end), total: items.length, next_cursor: next || undefined, coverage: snapshot.coverage || undefined };
}

function embeddedSourcePage(context, url) {
  const index = embeddedSourceIndex(context);
  const snapshot = embeddedSourceSnapshot(index, url);
  if (!snapshot) throw new Error("The exported source-facts scope is unavailable.");
  const files = embeddedMembership(snapshot, url);
  const symbols = embeddedSymbolMembership(snapshot, files);
  let items;
  if (url.pathname.endsWith("/files")) {
    items = (snapshot.files || []).filter(function (file) {
      return !files || files.has(file.id);
    }).filter(function (file) {
      const language = url.searchParams.get("language");
      const prefix = url.searchParams.get("path_prefix");
      return (!language || String(file.language && file.language.id || "").toLowerCase().indexOf(language.toLowerCase()) >= 0) && (!prefix || String(file.path || "").toLowerCase().indexOf(prefix.toLowerCase()) === 0);
    }).sort(function (left, right) { return String(left.path).localeCompare(String(right.path)) || String(left.id).localeCompare(String(right.id)); });
  } else if (url.pathname.endsWith("/symbols")) {
    const paths = {};
    (snapshot.files || []).forEach(function (file) { paths[file.id] = file.path; });
    items = (snapshot.symbols || []).filter(function (symbol) {
      if (!symbols) return true;
      return symbols.has(symbol.id);
    }).filter(function (symbol) {
      const name = url.searchParams.get("name");
      const languageKind = url.searchParams.get("language_kind");
      return (!name || String(symbol.name || "").toLowerCase().indexOf(name.toLowerCase()) >= 0) &&
        (!languageKind || String(symbol.language_kind || "").toLowerCase().indexOf(languageKind.toLowerCase()) >= 0);
    }).sort(function (left, right) {
      const leftPath = left.locations && left.locations[0] ? paths[left.locations[0].span.file_id] || "" : "";
      const rightPath = right.locations && right.locations[0] ? paths[right.locations[0].span.file_id] || "" : "";
      return leftPath.localeCompare(rightPath) || String(left.name).localeCompare(String(right.name)) || String(left.id).localeCompare(String(right.id));
    });
  } else {
    const subjectIDs = new Set(embeddedIDs(url, "subject_id"));
    items = (snapshot.documentation || []).filter(function (record) {
      if (subjectIDs.size && (!record.subject_ref || !subjectIDs.has(record.subject_ref.id))) return false;
      if (!files) return true;
      if (!record.subject_ref) return false;
      return record.subject_ref.kind === "file" ? files.has(record.subject_ref.id) : record.subject_ref.kind === "symbol" && symbols.has(record.subject_ref.id);
    }).map(function (record) {
	  return compactEmbeddedDocumentation(record, url.searchParams.get("include_documentation_text") === "true");
	}).sort(function (left, right) { return String(left.id).localeCompare(String(right.id)); });
  }
  return embeddedPage(items, snapshot, url);
}

function embeddedSourceEvidence(context, url) {
  const index = embeddedSourceIndex(context);
  const snapshot = embeddedSourceSnapshot(index, url);
  if (!snapshot) throw new Error("The exported source-facts scope is unavailable.");
  const entityID = decodeURIComponent(url.pathname.slice(url.pathname.lastIndexOf("/") + 1));
	const includeDocumentationText = url.searchParams.get("include_documentation_text") === "true";
  const result = { snapshot_id: snapshot.snapshot_id, scope_id: snapshot.scope_context && snapshot.scope_context.scope_id || "", entity_id: entityID, spans: [], source_reference_ids: [], documentation: [], relations: [] };
  (snapshot.symbols || []).forEach(function (symbol) {
    if (symbol.id !== entityID) return;
    (symbol.locations || []).forEach(function (location) {
      result.spans.push(location.span);
      (location.source_reference_ids || []).forEach(function (id) { result.source_reference_ids.push(id); });
    });
  });
  (snapshot.documentation || []).forEach(function (record) {
    if (record.id === entityID || record.subject_ref && record.subject_ref.id === entityID) {
		result.documentation.push(compactEmbeddedDocumentation(record, includeDocumentationText));
      (record.spans || []).forEach(function (span) { result.spans.push(span); });
	  (record.source_reference_ids || []).forEach(function (id) { result.source_reference_ids.push(id); });
    }
  });
	(snapshot.occurrences || []).forEach(function (occurrence) {
		if (occurrence.id === entityID && occurrence.source_span) result.spans.push(occurrence.source_span);
	});
  (snapshot.relations || []).forEach(function (relation) {
    if (relation.id === entityID || relation.from_ref && relation.from_ref.id === entityID || relation.to_ref && relation.to_ref.id === entityID) {
      result.relations.push(relation);
      (relation.evidence_spans || []).forEach(function (span) { result.spans.push(span); });
    }
  });
  if (!result.spans.length && !result.documentation.length && !result.relations.length) throw new Error("The source-fact evidence was not found in the exported model.");
  result.source_reference_ids = Array.from(new Set(result.source_reference_ids)).sort();
  return result;
}

function compactEmbeddedDocumentation(record, includeText) {
	if (includeText) return record;
	const copy = Object.assign({}, record);
	delete copy.raw_text;
	const characters = Array.from(String(copy.normalized_text || ""));
	if (characters.length > 512) {
		copy.normalized_text = characters.slice(0, 512).join("");
		copy.completeness = "truncated";
	}
	return copy;
}
