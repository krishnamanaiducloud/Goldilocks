import { defineConfig } from 'vitepress'
import { fileURLToPath } from 'node:url'

export default defineConfig({
  title: 'Goldilocks Documentation',
  description: "Documentation for Fairwinds' Goldilocks",
  cleanUrls: true,
  // These are intentional examples of endpoints exposed by a local Goldilocks install.
  ignoreDeadLinks: [/^http:\/\/localhost(?::\d+)?(?:\/.*)?$/],
  vite: {
    publicDir: fileURLToPath(new URL('../.vuepress/public', import.meta.url))
  },
  head: [
    ['link', { rel: 'icon', href: '/favicon.png' }],
    ['script', { src: '/scripts/marketing.js' }]
  ],
  themeConfig: {
    logo: '/img/fairwinds-logo.svg',
    nav: [
      { text: 'View on GitHub', link: 'https://github.com/FairwindsOps/goldilocks' }
    ],
    sidebar: [
      { text: 'Goldilocks', link: '/' },
      { text: 'Installation', link: '/installation' },
      { text: 'FAQ', link: '/faq' },
      { text: 'Advanced Usage', link: '/advanced' },
      {
        text: 'Contributing',
        items: [
          { text: 'Guide', link: '/contributing/guide' },
          { text: 'Code of Conduct', link: '/contributing/code-of-conduct' }
        ]
      }
    ],
    editLink: {
      pattern: 'https://github.com/FairwindsOps/goldilocks/edit/master/docs/:path',
      text: 'Help us improve this page'
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/FairwindsOps/goldilocks' }
    ],
    footer: {
      message: '<a href="https://fairwinds.com">Learn more about Fairwinds</a> · <a href="https://fairwinds.com/insights">Try Fairwinds Insights</a>',
      copyright: '<a href="https://www.fairwinds.com/privacy-policy">Privacy Policy</a>'
    }
  }
})
