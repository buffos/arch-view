(function (global) {
  "use strict";

  function optionAppliesToAlgorithm(option, algorithm) {
    const algorithms = option && Array.isArray(option.algorithms) ? option.algorithms : [];
    return !algorithms.length || algorithms.includes("all") || algorithms.includes(algorithm);
  }

  function isEditableRootOption(option, algorithm) {
    return Boolean(option)
      && option.editable === true
      && option.renderer_support === "supported"
      && Array.isArray(option.targets)
      && option.targets.includes("PARENTS")
      && optionAppliesToAlgorithm(option, algorithm);
  }

  function buildRootLayoutOptions(profile, catalog) {
    const selectedProfile = profile || { algorithm: "layered", options: {} };
    const algorithm = selectedProfile.algorithm || "layered";
    const profileOptions = selectedProfile.options || {};
    const rootLayoutOptions = {
      "elk.algorithm": algorithm.startsWith("org.eclipse.elk.") ? algorithm : "org.eclipse.elk." + algorithm,
      "elk.direction": "RIGHT",
      "elk.edgeRouting": "ORTHOGONAL"
    };
    // These application defaults are explicit ELK spacing values. Omit them
    // when the user selects the base spacing option so that ELK can derive the
    // individual spacing values from it instead of shadowing it.
    if (!Object.prototype.hasOwnProperty.call(profileOptions, "org.eclipse.elk.layered.spacing.baseValue")) {
      rootLayoutOptions["elk.spacing.nodeNode"] = "35";
      rootLayoutOptions["elk.layered.spacing.nodeNodeBetweenLayers"] = "84";
    }
    const catalogOptions = catalog && Array.isArray(catalog.options) ? catalog.options : [];
    Object.keys(profileOptions).forEach(function (key) {
      const value = profileOptions[key];
      const option = catalogOptions.find(function (item) { return item.id === key; });
      if (value !== undefined && value !== null && isEditableRootOption(option, algorithm)) {
        rootLayoutOptions[key] = String(value);
      }
    });
    return rootLayoutOptions;
  }

  function buildELKGraph(scene, profile, catalog) {
    const nodeWidth = 190;
    const nodeHeight = 82;
    return {
      id: "root",
      layoutOptions: buildRootLayoutOptions(profile, catalog),
      children: scene.visible_nodes.map(function (node) {
        return { id: node.id, width: nodeWidth, height: nodeHeight };
      }),
      edges: scene.visible_relationships.map(function (relationship) {
        return { id: relationship.id, sources: [relationship.from_visible_id], targets: [relationship.to_visible_id] };
      })
    };
  }

  global.ArchViewELKRequest = {
    buildELKGraph: buildELKGraph,
    buildRootLayoutOptions: buildRootLayoutOptions
  };
})(typeof window !== "undefined" ? window : globalThis);
