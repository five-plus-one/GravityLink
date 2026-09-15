import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/GravityLink/',
  lang: 'zh-CN',
  title: 'GravityLink',
  description: '高性能短链接与活码管理系统 — 部署与使用文档',
  cleanUrls: true,
  lastUpdated: true,
  srcExclude: ['DESIGN.md'],
  head: [
    ['meta', { name: 'theme-color', content: '#070B14' }],
    ['meta', { name: 'og:type', content: 'website' }],
    ['meta', { name: 'og:title', content: 'GravityLink 文档' }],
  ],
  themeConfig: {
    logo: { light: '/logo-light.svg', dark: '/logo-dark.svg', alt: 'GravityLink' },
    siteTitle: 'GravityLink',
    nav: [
      { text: '指南', link: '/guide/quick-start', activeMatch: '/guide/' },
      { text: '功能', link: '/features/', activeMatch: '/features/' },
      { text: '参考', link: '/reference/env', activeMatch: '/reference/' },
    ],
    sidebar: {
      '/guide/': [
        {
          text: '开始使用',
          items: [
            { text: '快速开始', link: '/guide/quick-start' },
            { text: '服务器部署', link: '/guide/deploy' },
            { text: '域名与 HTTPS', link: '/guide/nginx' },
          ],
        },
        {
          text: '日常配置',
          items: [
            { text: '初始化向导', link: '/guide/setup' },
            { text: '域名管理', link: '/guide/domains' },
            { text: '备份与升级', link: '/guide/ops' },
          ],
        },
      ],
      '/features/': [
        {
          text: '核心能力',
          items: [
            { text: '总览', link: '/features/' },
            { text: '短链接', link: '/features/short-link' },
            { text: '渠道码', link: '/features/channel-code' },
            { text: '群活码', link: '/features/live-qr' },
            { text: '落地页', link: '/features/landing-page' },
            { text: '统计分析', link: '/features/statistics' },
          ],
        },
        {
          text: '运营工具',
          items: [
            { text: '素材库', link: '/features/materials' },
            { text: '卡密分发', link: '/features/kami' },
            { text: '微信分享卡片', link: '/features/share-cards' },
            { text: '访客记录', link: '/features/visitors' },
          ],
        },
      ],
      '/reference/': [
        {
          text: '参考',
          items: [
            { text: '环境变量', link: '/reference/env' },
            { text: '架构摘要', link: '/reference/architecture' },
            { text: '端口与拓扑', link: '/reference/ports' },
          ],
        },
      ],
    },
    outline: { level: [2, 3], label: '本页目录' },
    docFooter: { prev: '上一篇', next: '下一篇' },
    darkModeSwitchLabel: '外观',
    sidebarMenuLabel: '菜单',
    returnToTopLabel: '回到顶部',
    search: {
      provider: 'local',
      options: {
        translations: {
          button: { buttonText: '搜索', buttonAriaLabel: '搜索文档' },
          modal: {
            noResultsText: '没有找到相关内容',
            resetButtonTitle: '清除',
            footer: { selectText: '选择', navigateText: '切换', closeText: '关闭' },
          },
        },
      },
    },
    socialLinks: [{ icon: 'github', link: 'https://github.com/five-plus-one/GravityLink' }],
    footer: {
      message: 'Go + Vue 3 · 自托管短链接与活码平台',
      copyright: 'GravityLink',
    },
  },
})
