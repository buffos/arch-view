import fs from 'node:fs'
import path from 'node:path'

const websiteRoot = path.resolve(import.meta.dirname, '..')
const inventoryPath = path.join(websiteRoot, '.generated', 'reference-inventory.json')
const outputPath = path.join(websiteRoot, 'reference', 'layout-options.generated.md')
const inventory = JSON.parse(fs.readFileSync(inventoryPath, 'utf8'))
const lines = [
  '# Complete layout catalog',
  '',
  'This appendix is built from the pinned layout catalog and shared renderer feature registry. It lists every known option and advanced feature. The simple explanation is intentionally short; options marked catalog-only are not editable in the current viewer.',
  '',
  `Total options: ${inventory.layout.options.length}.`,
  ''
]

const friendly = {
  'org.eclipse.elk.direction': 'Choose the main direction in which the graph should be read.',
  'org.eclipse.elk.edgeRouting': 'Choose the shape used by dependency lines.',
  'org.eclipse.elk.spacing.nodeNode': 'Choose the minimum space between nearby nodes.',
  'org.eclipse.elk.spacing.edgeNode': 'Choose the minimum space between an edge and a node.',
  'org.eclipse.elk.spacing.edgeEdge': 'Choose the minimum space between two edges.',
  'org.eclipse.elk.layered.spacing.nodeNodeBetweenLayers': 'Choose the space between nodes in different layers.',
  'org.eclipse.elk.layered.thoroughness': 'Choose how much effort the layered algorithm may spend searching for a layout.',
  'org.eclipse.elk.layered.layering.strategy': 'Choose how the layered algorithm assigns nodes to layers.',
  'org.eclipse.elk.layered.cycleBreaking.strategy': 'Choose how the layered algorithm handles cycles while drawing.',
  'org.eclipse.elk.layered.crossingMinimization.strategy': 'Choose how the layout tries to reduce crossing lines.',
  'org.eclipse.elk.layered.nodePlacement.strategy': 'Choose how nodes are positioned inside their layers.',
  'org.eclipse.elk.randomSeed': 'Choose the repeatable seed for a layout that uses randomness.',
  'org.eclipse.elk.separateConnectedComponents': 'Choose whether separate graph components are laid out independently.'
}

const algorithmExplanations = {
  fixed: 'Keep the positions already supplied by the model. Use this when you want a stable starting arrangement.',
  box: 'Place nodes in a simple box-like arrangement. Use this for a compact overview without dependency layers.',
  random: 'Place nodes with a repeatable random arrangement. Use this as a quick way to separate overlapping nodes.',
  layered: 'Place dependencies in layers. Use this when the direction of dependencies is the main story.',
  mrtree: 'Arrange the graph as a tree where possible. Use this when you want parent and child relationships to stand out.',
  stress: 'Move nodes to reduce overall connection length. Use this to reveal broad groups and clusters.',
  radial: 'Place related nodes around a center. Use this when one central module is the main focus.',
  force: 'Use a force-like arrangement that pushes connected and unrelated nodes into readable positions.',
  sporeOverlap: 'Resolve node overlaps while keeping the existing overall shape.',
  sporeCompaction: 'Reduce unused space while keeping nodes from overlapping.',
  rectpacking: 'Pack separate graph areas into a compact rectangle.'
}

for (const algorithm of inventory.layout.algorithms) {
  lines.push(`## ${algorithm.name}`)
  lines.push('')
  lines.push(algorithmExplanations[algorithm.id] || 'This algorithm chooses positions for the graph nodes.')
  lines.push('')
}

lines.push('## Advanced renderer features')
lines.push('')
lines.push('Features are opt-in presentation capabilities. Their support is determined by the pinned renderer, compatible algorithms, prerequisites, and export surface.')
lines.push('')

for (const feature of inventory.layout.features ?? []) {
  lines.push(`<a id="feature-${feature.id.replace(/[^a-z0-9]+/gi, '-').replace(/^-|-$/g, '').toLowerCase()}"></a>`)
  lines.push(`### ${feature.name}`)
  lines.push('')
  lines.push(`**Technical ID:** ${feature.id}.`)
  lines.push('')
  lines.push(`**Status:** ${feature.status || 'not specified'}.`)
  lines.push('')
  lines.push(`**Supported algorithms:** ${formatList(feature.algorithms)}.`)
  lines.push('')
  lines.push(`**Prerequisites:** ${formatList(feature.prerequisites)}.`)
  lines.push('')
  lines.push(`**Supported surfaces:** ${formatList(feature.surfaces)}.`)
  lines.push('')
  lines.push(`**Owned geometry:** ${formatList(feature.owns)}.`)
  lines.push('')
  lines.push(`**Fallback:** ${feature.fallback || 'not specified'}.`)
  lines.push('')
}

for (const option of inventory.layout.options) {
  const id = `layout-${option.id.replace(/[^a-z0-9]+/gi, '-').replace(/^-|-$/g, '').toLowerCase()}`
  const what = friendly[option.id] || `This setting controls ${option.name.toLowerCase()} in the layout engine.`
  const availability = option.editable && option.renderer_support === 'supported'
    ? 'Editable in the current viewer.'
    : 'Catalog-only or unsupported in the current viewer. It cannot be edited safely here.'
  lines.push(`<a id="${id}"></a>`)
  lines.push(`### ${option.name}`)
  lines.push('')
  lines.push(`**What it does:** ${what}`)
  lines.push('')
  lines.push(`**Type:** ${option.type || 'not specified'}.`)
  lines.push('')
  lines.push(`**Default:** ${formatValue(option.default)}.`)
  lines.push('')
  lines.push(`**Used by:** ${option.algorithms?.length ? option.algorithms.join(', ') : 'the shared layout stage'}.`)
  lines.push('')
  lines.push(`**Applies to:** ${option.targets?.length ? option.targets.join(', ') : 'the layout graph'}.`)
  lines.push('')
  lines.push(`**Availability:** ${availability}`)
  lines.push('')
  if (option.allowed_values?.length) {
    lines.push(`**Allowed values:** ${option.allowed_values.map(formatValue).join(', ')}.`)
    lines.push('')
  }
  lines.push(`Technical ID: ${option.id}.`)
  lines.push('')
}

fs.writeFileSync(outputPath, `${lines.join('\n')}\n`)
console.log(`Wrote ${path.relative(websiteRoot, outputPath)} (${inventory.layout.options.length} options).`)

function formatValue(value) {
  if (value === null || value === undefined) return 'engine default'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function formatList(values) {
  return Array.isArray(values) && values.length ? values.join(', ') : 'none declared'
}
