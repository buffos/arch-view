function normalizedPath(node) {
  if (!Array.isArray(node?.hierarchy_path)) return [];
  return node.hierarchy_path.map((part) => String(part).trim()).filter(Boolean);
}

function isProperPrefix(prefix, path) {
  return prefix.length > 0 && prefix.length < path.length
    && prefix.every((part, index) => part === path[index]);
}

export function visibleHierarchy(nodes) {
  const entries = (nodes || []).map((node) => ({ id: node.id, path: normalizedPath(node) }));
  const ids = new Set();
  for (const entry of entries) {
    if (!entry.id || ids.has(entry.id)) throw new Error("Visible hierarchy contains a missing or duplicate node ID.");
    ids.add(entry.id);
  }
  const parentByNode = {};
  const childrenByNode = Object.fromEntries(entries.map((entry) => [entry.id, []]));
  for (const entry of entries) {
    const candidates = entries.filter((candidate) => candidate.id !== entry.id && isProperPrefix(candidate.path, entry.path));
    candidates.sort((left, right) => right.path.length - left.path.length || left.id.localeCompare(right.id));
    const parent = candidates[0]?.id || null;
    parentByNode[entry.id] = parent;
    if (parent) childrenByNode[parent].push(entry.id);
  }
  Object.values(childrenByNode).forEach((children) => children.sort());
  return {
    parentByNode,
    childrenByNode,
    roots: entries.filter((entry) => !parentByNode[entry.id]).map((entry) => entry.id).sort(),
    containerNodeIDs: entries.filter((entry) => childrenByNode[entry.id].length > 0).map((entry) => entry.id).sort()
  };
}

export function nestELKNodes(flatNodes, hierarchy) {
  const byID = new Map((flatNodes || []).map((node) => [node.id, node]));
  if (byID.size !== (flatNodes || []).length) throw new Error("ELK input contains duplicate node IDs.");
  for (const id of Object.keys(hierarchy.parentByNode)) if (!byID.has(id)) throw new Error("Visible hierarchy has no matching ELK node: " + id);
  const build = (id) => {
    const source = byID.get(id);
    const children = hierarchy.childrenByNode[id].map(build);
    if (!children.length) return { ...source };
    const headerHeight = Number.isFinite(source.height) ? source.height : 82;
    return {
      ...source,
      layoutOptions: {
        ...(source.layoutOptions || {}),
        "elk.padding": "[top=" + (headerHeight + 20) + ",left=18,bottom=18,right=18]"
      },
      children
    };
  };
  return hierarchy.roots.map(build);
}
