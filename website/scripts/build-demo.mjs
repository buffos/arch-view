import fs from 'node:fs'
import path from 'node:path'
import { spawnSync } from 'node:child_process'

const websiteRoot = path.resolve(import.meta.dirname, '..')
const repositoryRoot = path.resolve(websiteRoot, '..')
const outputPath = path.join(websiteRoot, 'public', 'interactive-demo', 'index.html')

fs.mkdirSync(path.dirname(outputPath), { recursive: true })
const result = spawnSync('go', ['run', './cmd/docs-demo', '-output', outputPath], {
  cwd: repositoryRoot,
  encoding: 'utf8',
  windowsHide: true
})
if (result.error) throw result.error
if (result.status !== 0) {
  console.error(result.stdout)
  console.error(result.stderr)
  process.exit(result.status || 1)
}
const html = fs.readFileSync(outputPath, 'utf8')
if (!html.includes('<style>') || !html.includes('<script>')) {
  throw new Error('static demo is missing inline styles or scripts')
}
if (/<(?:script|link)\b[^>]*(?:src|href)=/i.test(html)) {
  throw new Error('static demo contains an external script or stylesheet reference')
}
console.log('Wrote ' + path.relative(repositoryRoot, outputPath) + '.')
