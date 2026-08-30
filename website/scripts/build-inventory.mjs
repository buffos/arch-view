import fs from 'node:fs'
import path from 'node:path'
import { spawnSync } from 'node:child_process'

const websiteRoot = path.resolve(import.meta.dirname, '..')
const repositoryRoot = path.resolve(websiteRoot, '..')
const outputPath = path.join(websiteRoot, '.generated', 'reference-inventory.json')

function runGo(args, allowNonZero = false) {
  const result = spawnSync('go', args, {
    cwd: repositoryRoot,
    encoding: 'utf8',
    windowsHide: true
  })
  if (result.error) throw result.error
  const output = result.stdout + result.stderr
  if (result.status !== 0 && (!allowNonZero || !output.includes('Usage of'))) {
    throw new Error(`go ${args.join(' ')} failed with exit code ${result.status}\n${output}`)
  }
  return output
}

const inventory = JSON.parse(runGo(['run', './cmd/docs-inventory', '-output', '-']))
const cliFlags = []

for (const command of inventory.cli_commands) {
  const help = runGo(['run', './cmd/arch-view', ...command.help_args], true)
  const flags = [...help.matchAll(/^\s+-([a-z][a-z0-9-]*)\s+/gim)].map((match) => match[1])
  for (const name of [...new Set(flags)].sort()) {
    cliFlags.push({
      command: command.name,
      name,
      docs_path: flagDocsPath(command.name, name, command.docs_path),
      docs_anchor: flagDocsAnchor(command.name, name)
    })
  }
}

inventory.cli_flags = cliFlags
inventory.cli_commands = inventory.cli_commands.map(({ name, docs_path, docs_anchor }) => ({ name, docs_path, docs_anchor }))
fs.mkdirSync(path.dirname(outputPath), { recursive: true })
fs.writeFileSync(outputPath, `${JSON.stringify(inventory, null, 2)}\n`)
console.log(`Wrote ${path.relative(repositoryRoot, outputPath)} (${cliFlags.length} CLI flags).`)

function flagDocsPath(command, name, defaultPath) {
  if (command !== 'open') return defaultPath
  if (name === 'model' || name === 'project' || name === 'port') return defaultPath
  return '/cli/analyze'
}

function flagDocsAnchor(command, name) {
  if (command === 'analyzers') return '#options'
  if (command === 'quality baseline') return '#options'
  if (command === 'model normalize') return '#normalize'
  if (command === 'model validate') return '#validate'
  if (command === 'model projection') return '#projection'
  if (command === 'open') {
    if (name === 'project' || name === 'model') return '#input-options'
    if (name === 'port') return '#server-option'
    return analyzeFlagAnchor(name)
  }
  if (command === 'export') {
    if (name === 'input' || name === 'format' || name === 'output') return '#required-options'
    if (name === 'quality-exit-on' || name === 'quality-exit-status') return '#quality-exit-options'
    return '#display-options'
  }
  return analyzeFlagAnchor(name)
}

function analyzeFlagAnchor(name) {
  const sourceFlags = new Set([
    'module', 'crate', 'target', 'config', 'source-root', 'exclude',
    'build-tag', 'feature', 'features', 'include-js', 'include-tests',
    'include-examples', 'include-generated', 'include-external', 'safe-mode',
    'python-version', 'include-stubs', 'platform', 'runtime'
  ])
  const outputFlags = new Set([
    'format', 'reference-visibility', 'view-path', 'reference-scope',
    'deterministic', 'overwrite', 'embed-source'
  ])
  const qualityFlags = new Set([
    'quality-profile', 'quality-baseline', 'quality-exit-on', 'quality-exit-status'
  ])
  if (sourceFlags.has(name)) return '#source-selection'
  if (outputFlags.has(name)) return '#output-options'
  if (qualityFlags.has(name)) return '#quality-options'
  if (name === 'project' || name === 'output') return '#required-options'
  return '#analyzer-selection'
}
