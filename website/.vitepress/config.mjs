import { defineConfig } from 'vitepress'

const base = process.env.DOCS_BASE ?? (process.env.CI ? '/arch-view-golang/' : '/')

export default defineConfig({
  lang: 'en-US',
  title: 'Arch View',
  description: 'A simple guide to understanding your codebase with Arch View.',
  base,
  cleanUrls: true,
  lastUpdated: true,
  appearance: true,
  markdown: {
    lineNumbers: true
  },
  themeConfig: {
    logo: '/logo.svg',
    siteTitle: 'Arch View',
    nav: [
      { text: 'Guide', link: '/guide/quick-start' },
      { text: 'Viewer', link: '/viewer/overview' },
      { text: 'Quality', link: '/quality/overview' },
      { text: 'CLI', link: '/cli/overview' },
      { text: 'Live & MCP', link: '/mcp/installation' },
      { text: 'Reference', link: '/reference/settings' },
      { text: 'Demo', link: '/demo' }
    ],
    sidebar: {
      '/guide/': [
        {
          text: 'Start here',
          items: [
            { text: 'What is Arch View?', link: '/guide/what-is-arch-view' },
            { text: 'Installation', link: '/guide/installation' },
            { text: 'Quick start', link: '/guide/quick-start' },
            { text: 'How Arch View thinks about code', link: '/guide/core-concepts' }
          ]
        }
      ],
      '/viewer/': [
        {
          text: 'Use the viewer',
          items: [
            { text: 'Viewer overview', link: '/viewer/overview' },
            { text: 'Read the graph', link: '/viewer/read-the-graph' },
            { text: 'Inspect a node', link: '/viewer/inspection' },
            { text: 'Change the layout', link: '/viewer/layout' }
          ]
        }
      ],
      '/quality/': [
        {
          text: 'Quality checks',
          items: [
            { text: 'Quality checks overview', link: '/quality/overview' },
            { text: 'Profiles', link: '/quality/profiles' },
            { text: 'Findings and coverage', link: '/quality/findings' },
            { text: 'Baselines', link: '/quality/baselines' },
            { text: 'Rule reference', link: '/quality/rules' }
          ]
        }
      ],
      '/cli/': [
        {
          text: 'Command line',
          items: [
            { text: 'CLI overview', link: '/cli/overview' },
            { text: 'analyze', link: '/cli/analyze' },
            { text: 'open', link: '/cli/open' },
            { text: 'export', link: '/cli/export' },
            { text: 'quality baseline', link: '/cli/quality-baseline' },
            { text: 'model commands', link: '/cli/model' },
            { text: 'analyzers', link: '/cli/analyzers' }
          ]
        }
      ],
      '/mcp/': [
        {
          text: 'Live analysis and MCP',
          items: [
            { text: 'MCP installation', link: '/mcp/installation' },
            { text: 'MCP tools', link: '/mcp/tools' },
            { text: 'Agent skill', link: '/mcp/agent-skill' },
            { text: 'HTTP transport', link: '/mcp/http' },
            { text: 'Safety and freshness', link: '/mcp/security' }
          ]
        }
      ],
      '/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'Settings', link: '/reference/settings' },
            { text: 'Quality profile JSON', link: '/formats/quality-profile' },
            { text: 'Baseline JSON', link: '/formats/baseline' },
            { text: 'Model JSON', link: '/formats/model' },
            { text: 'Analyzer options', link: '/reference/analyzer-options' },
            { text: 'Layout options', link: '/reference/layout-options' }
          ]
        }
      ],
      '/analyzers/': [
        {
          text: 'Analyzers',
          items: [
            { text: 'Analyzer overview', link: '/analyzers/overview' },
            { text: 'Go', link: '/analyzers/go' },
            { text: 'Python', link: '/analyzers/python' },
            { text: 'TypeScript', link: '/analyzers/typescript' },
            { text: 'Rust', link: '/analyzers/rust' },
            { text: 'Clojure', link: '/analyzers/clojure' },
            { text: 'Plugins', link: '/analyzers/plugins' }
          ]
        }
      ],
      '/troubleshooting/': [
        {
          text: 'Troubleshooting',
          items: [
            { text: 'Common questions', link: '/troubleshooting/common-questions' }
          ]
        }
      ]
    },
    outline: 'deep',
    search: { provider: 'local' },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/buffo/arch-view' }
    ],
    footer: {
      message: 'Arch View is in pre-alpha. The documentation describes the current version.',
      copyright: 'Copyright © Arch View contributors'
    },
    editLink: {
      pattern: 'https://github.com/buffo/arch-view/edit/master/website/:path'
    }
  }
})
