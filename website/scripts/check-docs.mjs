import fs from 'node:fs'
import path from 'node:path'

const websiteRoot = path.resolve(import.meta.dirname, '..')
const inventoryPath = path.join(websiteRoot, '.generated', 'reference-inventory.json')
const inventory = JSON.parse(fs.readFileSync(inventoryPath, 'utf8'))
const markdownFiles = collectMarkdown(websiteRoot)
const textByPath = new Map(markdownFiles.map((file) => [file, fs.readFileSync(file, 'utf8')]))
const allText = [...textByPath.values()].join('\n')
const errors = []

const requiredFiles = [
  'index.md',
  'guide/quick-start.md',
  'guide/core-concepts.md',
  'viewer/inspection.md',
  'quality/overview.md',
  'quality/rules.md',
  'cli/analyze.md',
  'cli/open.md',
  'reference/settings.md',
  'formats/model.md',
  'formats/okf.md',
  'okf/overview.md',
  'okf/profiles.md',
  'okf/exploration.md',
  'okf/configuration.md',
  'reference/okf-api.md',
  'viewer/advanced-rendering.md',
  'demo.md'
]
for (const relative of requiredFiles) {
  if (!fs.existsSync(path.join(websiteRoot, relative))) errors.push(`missing required page: ${relative}`)
}

if (allText.includes('/docs/architecture') || allText.includes('.okf/')) {
  errors.push('public documentation must not link to internal .okf or architecture documents')
}

for (const flag of inventory.cli_flags) {
  const page = path.join(websiteRoot, `${flag.docs_path.replace(/^\//, '')}.md`)
  const content = textByPath.get(page)
  if (!content) {
    errors.push(`CLI flag ${flag.command} --${flag.name} points to missing page ${flag.docs_path}`)
    continue
  }
  if (!content.includes(`--${flag.name}`)) errors.push(`CLI flag ${flag.command} --${flag.name} is not explained on ${flag.docs_path}`)
  if (flag.docs_anchor && !hasAnchor(content, flag.docs_anchor)) {
    errors.push(`CLI flag ${flag.command} --${flag.name} points to missing anchor ${flag.docs_path}${flag.docs_anchor}`)
  }
}

for (const analyzer of inventory.analyzers) {
  const page = path.join(websiteRoot, 'analyzers', `${languagePage(analyzer.language)}.md`)
  const content = textByPath.get(page)
  if (!content) errors.push(`missing analyzer page for ${analyzer.language}`)
  for (const option of analyzer.options) {
    const flagName = analyzerFlagName(option.name)
    if (!content?.includes(flagName) && !textByPath.get(path.join(websiteRoot, 'cli/analyze.md'))?.includes(flagName)) {
      errors.push(`analyzer option ${analyzer.id}:${option.name} is not explained`)
    }
  }
}

for (const rule of inventory.quality_rules) {
  if (!allText.includes(rule.id)) errors.push(`quality rule ${rule.id} is not explained`)
  for (const field of Object.keys(rule.parameter_schema?.fields ?? {})) {
    if (!allText.includes(field.replaceAll('_', '-')) && !allText.includes(field)) {
      errors.push(`quality parameter ${rule.id}:${field} is not mentioned`)
    }
  }
}

const formatDocumentation = [
  ['model_fields', 'formats/model.md'],
  ['quality_profile_fields', 'formats/quality-profile.md'],
  ['rule_binding_fields', 'formats/quality-profile.md'],
  ['typed_config_block_fields', 'formats/quality-profile.md'],
  ['baseline_fields', 'formats/baseline.md'],
  ['baseline_entry_fields', 'formats/baseline.md']
]
for (const [inventoryKey, relativePage] of formatDocumentation) {
  const content = textByPath.get(path.join(websiteRoot, relativePage)) ?? ''
  for (const field of inventory.formats?.[inventoryKey] ?? []) {
    if (!content.includes(field)) errors.push(`format field ${field} is not documented in ${relativePage}`)
  }
}

const layoutPage = textByPath.get(path.join(websiteRoot, 'reference/layout-options.generated.md'))
if (!layoutPage) errors.push('generated layout option appendix is missing')
for (const option of inventory.layout.options) {
  if (!layoutPage?.includes(`Technical ID: ${option.id}.`)) errors.push(`layout option ${option.id} is not documented`)
}

for (const feature of inventory.layout.features ?? []) {
  if (!layoutPage?.includes(`**Technical ID:** ${feature.id}.`)) errors.push(`layout feature ${feature.id} is not documented`)
  if (!layoutPage?.includes(`**Status:** ${feature.status}.`)) errors.push(`layout feature ${feature.id} status is not documented`)
  if (!layoutPage?.includes(`**Supported algorithms:** ${formatList(feature.algorithms)}.`)) errors.push(`layout feature ${feature.id} algorithm support is not documented`)
  if (!layoutPage?.includes(`**Prerequisites:** ${formatList(feature.prerequisites)}.`)) errors.push(`layout feature ${feature.id} prerequisites are not documented`)
  if (!layoutPage?.includes(`**Supported surfaces:** ${formatList(feature.surfaces)}.`)) errors.push(`layout feature ${feature.id} surfaces are not documented`)
}

for (const link of findInternalLinks(allText)) {
  const target = resolveLink(link.path)
  if (target && !hasPage(target)) errors.push(`broken internal documentation link: ${link.path}${link.anchor}`)
  if (target && link.anchor) {
    const content = textByPath.get(path.join(websiteRoot, target)) ?? ''
    if (!hasAnchor(content, link.anchor)) errors.push(`broken internal documentation anchor: ${link.path}${link.anchor}`)
  }
}

if (errors.length) {
  console.error(`Documentation check failed with ${errors.length} error(s):`)
  for (const error of errors) console.error(`- ${error}`)
  process.exit(1)
}

console.log(`Documentation check passed: ${markdownFiles.length} pages, ${inventory.cli_flags.length} CLI flags, ${inventory.quality_rules.length} quality rules, ${inventory.layout.options.length} layout options.`)

function collectMarkdown(root) {
  const results = []
  for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
    if (entry.name === '.vitepress' || entry.name === '.generated' || entry.name === 'node_modules') continue
    const absolute = path.join(root, entry.name)
    if (entry.isDirectory()) results.push(...collectMarkdown(absolute))
    else if (entry.isFile() && entry.name.endsWith('.md')) results.push(absolute)
  }
  return results
}

function languagePage(language) {
  return language === 'typescript' ? 'typescript' : language
}

function analyzerFlagName(name) {
  const aliases = {
    build_tags: 'build-tag',
    features: 'feature',
    source_roots: 'source-root'
  }
  return `--${aliases[name] || name.replaceAll('_', '-')}`
}

function findInternalLinks(text) {
  return [...text.matchAll(/\]\((\/[^)#\s]+)(#[^)\s]+)?\)/g)].map((match) => ({ path: match[1], anchor: match[2] ?? '' }))
}

function resolveLink(link) {
  if (link === '/') return 'index.md'
  const clean = link.replace(/^\//, '').replace(/\/$/, '')
  return `${clean || 'index'}.md`
}

function hasPage(relative) {
  return fs.existsSync(path.join(websiteRoot, relative)) || fs.existsSync(path.join(websiteRoot, relative.replace(/\.md$/, '/index.md')))
}

function hasAnchor(content, anchor) {
  const wanted = anchor.replace(/^#/, '').toLowerCase()
  if (!wanted) return true
  if (content.includes(`id="${wanted}"`)) return true
  return [...content.matchAll(/^#{1,6}\s+(.+)$/gm)].some((match) => slugify(match[1]) === wanted)
}

function slugify(value) {
  return value
    .replace(/[`*_~]/g, '')
    .replace(/<[^>]+>/g, '')
    .trim()
    .toLowerCase()
    .replace(/[^\p{L}\p{N}\s-]/gu, '')
    .replace(/\s+/g, '-')
}

function formatList(values) {
  return Array.isArray(values) && values.length ? values.join(', ') : 'none declared'
}
