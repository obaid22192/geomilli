import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

export default withMermaid(
  defineConfig({
    base: process.env.VITEPRESS_BASE || '/docs/',
    title: 'Geomilli',
    description: 'Open-Source Geospatial Search & OpenStreetMap Intelligence Powered by Meilisearch',
    lang: 'en-US',
    lastUpdated: true,
    cleanUrls: true,
    ignoreDeadLinks: true,
    head: [
      ['link', { rel: 'icon', href: '/favicon.ico' }],
      ['meta', { name: 'theme-color', content: '#3b82f6' }]
    ],
  themeConfig: {
    siteTitle: 'Geomilli',
    logo: '/logo.svg',
    nav: [
      { text: 'Guide', link: '/guide/quickstart' },
      { text: 'Harvester', link: '/harvester/configuration' },
      { text: 'API Reference', link: '/api/overview' },
      { text: 'Studio UI', link: '/studio/playground-and-metrics' },
      {
        text: 'v1.0.0',
        items: [
          { text: 'GitHub Release', link: 'https://github.com/geomilli/geomilli/releases' },
          { text: 'OpenAPI 3.1 Spec', link: '/api/openapi.yaml' }
        ]
      }
    ],
    sidebar: {
      '/guide/': [
        {
          text: 'Getting Started',
          items: [
            { text: '60-Second Quickstart', link: '/guide/quickstart' },
            { text: 'System Architecture', link: '/guide/architecture' },
            { text: 'O(1) Streaming & Memory', link: '/guide/streaming-and-memory' },
            { text: 'Meilisearch: Local vs Cloud', link: '/guide/meilisearch-local-vs-cloud' }
          ]
        }
      ],
      '/harvester/': [
        {
          text: 'Harvester Pipeline',
          items: [
            { text: 'Configuration & Countries', link: '/harvester/configuration' },
            { text: 'Taxonomy & Languages', link: '/harvester/taxonomy-and-languages' },
            { text: 'Enrichment & WAL Caching', link: '/harvester/enrichment-and-caching' }
          ]
        }
      ],
      '/api/': [
        {
          text: 'REST API',
          items: [
            { text: 'API Overview & Auth', link: '/api/overview' },
            { text: 'Autocomplete & Search', link: '/api/autocomplete-and-search' },
            { text: 'Geospatial & POI APIs', link: '/api/geospatial-apis' }
          ]
        }
      ],
      '/studio/': [
        {
          text: 'Geomilli Studio',
          items: [
            { text: 'Playground & Metrics', link: '/studio/playground-and-metrics' }
          ]
        }
      ]
    },
    search: {
      provider: 'local'
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/geomilli/geomilli' }
    ],
    footer: {
      message: 'Released under the Apache 2.0 License.',
      copyright: 'Copyright © 2026 Geomilli Contributors'
    }
  }
}))
