export function mapELKNodes(nodes, transform) {
  return (nodes || []).map((node) => {
    const mapped = { ...node };
    if (Array.isArray(node.children)) mapped.children = mapELKNodes(node.children, transform);
    return transform(mapped);
  });
}
