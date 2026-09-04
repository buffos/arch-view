import { shapeDimensions } from "./app/node_shape.js";

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

  function isEditableTargetOption(option, algorithm, target) {
    return Boolean(option)
      && option.editable === true
      && option.renderer_support === "supported"
      && Array.isArray(option.targets)
      && option.targets.includes(target)
      && optionAppliesToAlgorithm(option, algorithm);
  }

  function buildTargetLayoutOptions(profile, catalog, target) {
    const selectedProfile = profile || { algorithm: "layered", options: {} };
    const algorithm = selectedProfile.algorithm || "layered";
    const profileOptions = selectedProfile.options || {};
    const catalogOptions = catalog && Array.isArray(catalog.options) ? catalog.options : [];
    const targetLayoutOptions = {};
    Object.keys(profileOptions).sort().forEach(function (key) {
      const value = profileOptions[key];
      const option = catalogOptions.find(function (item) { return item.id === key; });
      if (value !== undefined && value !== null && isEditableTargetOption(option, algorithm, target)) {
        targetLayoutOptions[key] = String(value);
      }
    });
    return targetLayoutOptions;
  }

  function copyLayoutOptions(element, options) {
    if (Object.keys(options).length) element.layoutOptions = Object.assign({}, options);
    return element;
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
    Object.keys(profileOptions).sort().forEach(function (key) {
      const value = profileOptions[key];
      const option = catalogOptions.find(function (item) { return item.id === key; });
      if (value !== undefined && value !== null && isEditableRootOption(option, algorithm)) {
        // ELK's short root keys are the canonical request keys for these
        // built-in options. Avoid sending a default short key together with a
        // conflicting fully-qualified alias so the selected value is applied
        // deterministically by the engine.
        const requestKey = key === "org.eclipse.elk.direction" || key === "org.eclipse.elk.edgeRouting"
          ? key.replace("org.eclipse.elk.", "elk.")
          : key;
        rootLayoutOptions[requestKey] = String(value);
      }
    });
    return rootLayoutOptions;
  }

  function nodeDimensions(node) {
    if (!Array.isArray(node && node.presentation_fields)) return { width: 190, height: 82 };
    return shapeDimensions(node.shape_definition || node.shape || node.presentation_style?.shape, { width: 210, height: 58 + Math.min(node.presentation_fields.length, 3) * 18 });
  }

  function buildELKGraph(scene, profile, catalog) {
    const nodeLayoutOptions = buildTargetLayoutOptions(profile, catalog, "NODES");
    const edgeLayoutOptions = buildTargetLayoutOptions(profile, catalog, "EDGES");
    return {
      id: "root",
      layoutOptions: buildRootLayoutOptions(profile, catalog),
      children: scene.visible_nodes.map(function (node) {
        const dimensions = nodeDimensions(node);
        return copyLayoutOptions({ id: node.id, width: dimensions.width, height: dimensions.height }, nodeLayoutOptions);
      }),
      edges: scene.visible_relationships.map(function (relationship) {
        return copyLayoutOptions({ id: relationship.id, sources: [relationship.from_visible_id], targets: [relationship.to_visible_id] }, edgeLayoutOptions);
      })
    };
  }

export { buildELKGraph, buildRootLayoutOptions, buildTargetLayoutOptions, nodeDimensions };
