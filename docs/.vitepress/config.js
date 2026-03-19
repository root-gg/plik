import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

export default withMermaid(defineConfig({
    title: 'Plik',
    description: 'Temporary file upload system',
    base: '/plik/',

    head: [
        ['meta', { name: 'theme-color', content: '#3eaf7c' }],
    ],

    themeConfig: {
        logo: '/logo.png',

        nav: [],

        sidebar: {
            '/': [
                {
                    text: 'Guide',
                    items: [
                        { text: 'Installation', link: '/guide/installation' },
                        { text: 'Configuration', link: '/guide/configuration' },
                        { text: 'Docker Deployment', link: '/guide/docker' },
                        { text: 'Kubernetes (Helm)', link: '/guide/kubernetes' },
                        { text: 'Security', link: '/guide/security' },
                        { text: 'Send to Plik', link: '/guide/windows-send-to' },
                    ],
                },
                {
                    text: 'Features',
                    items: [
                        { text: 'CLI', link: '/features/cli-client' },
                        { text: 'Web UI', link: '/features/web-ui' },
                        { text: 'Authentication', link: '/features/authentication' },
                        { text: 'Encryption', link: '/features/encryption' },
                        { text: 'Streaming', link: '/features/streaming' },
                        { text: 'MCP Server', link: '/features/mcp' },
                        { text: 'Internationalization', link: '/features/internationalization' },
                    ],
                },
                {
                    text: 'Backend',
                    items: [
                        { text: 'Data Backends', link: '/backends/data' },
                        { text: 'Metadata Backends', link: '/backends/metadata' },
                    ],
                },
                {
                    text: 'Reference',
                    items: [
                        { text: 'HTTP API', link: '/reference/api' },
                        { text: 'Go Library', link: '/reference/go-library' },
                        { text: 'Prometheus Metrics', link: '/reference/metrics' },
                    ],
                },
                {
                    text: 'Architecture',
                    items: [
                        { text: 'System-wide', link: '/architecture/system' },
                        { text: 'Server', link: '/architecture/server' },
                        { text: 'CLI', link: '/architecture/client' },
                        { text: 'Go Library', link: '/architecture/library' },
                        { text: 'Web UI', link: '/architecture/webapp' },
                        { text: 'Testing', link: '/architecture/testing' },
                        { text: 'Releaser', link: '/architecture/releaser' },
                        { text: 'GitHub Actions', link: '/architecture/github' },
                    ],
                },
                {
                    text: 'Operations',
                    items: [
                        { text: 'Reverse Proxy', link: '/operations/reverse-proxy' },
                        { text: 'Server CLI', link: '/operations/server-cli' },
                        { text: 'Import / Export', link: '/operations/import-export' },
                        { text: 'Cross Compilation', link: '/operations/cross-compilation' },
                    ],
                },
                {
                    text: 'Contributing',
                    link: '/contributing',
                },
            ],
        },

        socialLinks: [
            {
                icon: { svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10c5.51 0 10-4.48 10-10S17.51 2 12 2zm4.64 6.8c-.15 1.58-.8 5.42-1.13 7.19-.14.75-.42 1-.68 1.03-.58.05-1.02-.38-1.58-.75-.88-.58-1.38-.94-2.23-1.5-.99-.65-.35-1.01.22-1.59.15-.15 2.71-2.48 2.76-2.69a.2.2 0 00-.05-.18c-.06-.05-.14-.03-.21-.02-.09.02-1.49.95-4.22 2.79-.4.27-.76.41-1.08.4-.36-.01-1.04-.2-1.55-.37-.63-.2-1.12-.31-1.08-.66.02-.18.27-.36.74-.55 2.92-1.27 4.86-2.11 5.83-2.51 2.78-1.16 3.35-1.36 3.73-1.36.08 0 .27.02.39.12.1.08.13.19.14.27-.01.06.01.24 0 .38z"/></svg>' },
                link: 'https://t.me/plik_rootgg',
                ariaLabel: 'Telegram',
            },
            {
                icon: { svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M13.983 11.078h2.119a.186.186 0 00.186-.185V9.006a.186.186 0 00-.186-.186h-2.119a.186.186 0 00-.185.186v1.887c0 .102.083.185.185.185m-2.954-5.43h2.118a.186.186 0 00.186-.186V3.574a.186.186 0 00-.186-.185h-2.118a.186.186 0 00-.185.185v1.888c0 .102.082.186.185.186m0 2.716h2.118a.187.187 0 00.186-.186V6.29a.186.186 0 00-.186-.185h-2.118a.186.186 0 00-.185.185v1.887c0 .102.082.186.185.186m-2.93 0h2.12a.186.186 0 00.184-.186V6.29a.185.185 0 00-.185-.185H8.1a.186.186 0 00-.185.185v1.887c0 .102.083.186.185.186m-2.964 0h2.119a.186.186 0 00.185-.186V6.29a.186.186 0 00-.185-.185H5.136a.186.186 0 00-.186.185v1.887c0 .102.084.186.186.186m5.893 2.715h2.118a.186.186 0 00.186-.185V9.006a.186.186 0 00-.186-.186h-2.118a.186.186 0 00-.185.186v1.887c0 .102.082.185.185.185m-2.93 0h2.12a.185.185 0 00.184-.185V9.006a.185.185 0 00-.184-.186h-2.12a.185.185 0 00-.184.186v1.887c0 .102.083.185.185.185m-2.964 0h2.119a.186.186 0 00.185-.185V9.006a.186.186 0 00-.185-.186H5.136a.186.186 0 00-.186.186v1.887c0 .102.084.185.186.185m-2.92 0h2.12a.185.185 0 00.184-.185V9.006a.185.185 0 00-.184-.186h-2.12a.186.186 0 00-.186.186v1.887c0 .102.084.185.186.185M23.763 9.89c-.065-.051-.672-.51-1.954-.51-.338.001-.676.03-1.01.087-.248-1.7-1.653-2.53-1.716-2.566l-.344-.199-.226.327c-.284.438-.49.922-.612 1.43-.23.97-.09 1.882.403 2.661-.595.332-1.55.413-1.744.42H.751a.751.751 0 00-.75.748 11.687 11.687 0 00.692 4.062c.545 1.428 1.355 2.48 2.41 3.124 1.18.723 3.1 1.137 5.275 1.137.983.003 1.963-.086 2.93-.266a12.33 12.33 0 003.823-1.389c.98-.567 1.86-1.288 2.61-2.136 1.252-1.418 1.998-2.997 2.553-4.4h.221c1.372 0 2.215-.549 2.68-1.009.309-.293.55-.65.707-1.046l.098-.288z"/></svg>' },
                link: 'https://hub.docker.com/r/rootgg/plik',
                ariaLabel: 'Docker Hub',
            },
            { icon: 'github', link: 'https://github.com/root-gg/plik' },
        ],

        search: {
            provider: 'local',
        },

        editLink: {
            pattern: 'https://github.com/root-gg/plik/edit/master/docs/:path',
        },

        footer: {
            message: 'Released under the MIT License.',
            copyright: 'Copyright © root-gg',
        },
    },
}))
