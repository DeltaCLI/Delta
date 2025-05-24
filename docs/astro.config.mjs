import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import mdx from '@astrojs/mdx';

export default defineConfig({
  integrations: [
    mdx(),
    starlight({
      title: 'Delta CLI Documentation',
      social: {
        github: 'https://github.com/yourusername/deltacli',
      },
      sidebar: [
        {
          label: 'Get Started',
          items: [
            { label: 'Introduction', link: '/introduction/' },
            { label: 'Installation', link: '/installation/' },
            { label: 'Quick Start', link: '/quick-start/' },
          ],
        },
        {
          label: 'Features',
          items: [
            { label: 'AI Assistant', link: '/features/ai-assistant/' },
            { label: 'Jump Navigation', link: '/features/jump-navigation/' },
            { label: 'Memory System', link: '/features/memory-system/' },
            { label: 'Tokenizer', link: '/features/tokenizer/' },
            { label: 'Inference System', link: '/features/inference-system/' },
            { label: 'Vector Database', link: '/features/vector-database/' },
            { label: 'Embedding System', link: '/features/embedding-system/' },
            { label: 'Speculative Decoding', link: '/features/speculative-decoding/' },
            { label: 'Knowledge Extraction', link: '/features/knowledge-extraction/' },
            { label: 'Agent System', link: '/features/agent-system/' },
            { label: 'Configuration System', link: '/features/configuration-system/' },
            { label: 'Spell Checker', link: '/features/spell-checker/' },
            { label: 'History Analysis', link: '/features/history-analysis/' },
          ],
        },
        {
          label: 'Advanced',
          items: [
            { label: 'Custom Agents', link: '/advanced/custom-agents/' },
            { label: 'Training Custom Models', link: '/advanced/training-custom-models/' },
            { label: 'Privacy Settings', link: '/advanced/privacy-settings/' },
            { label: 'Configuration', link: '/advanced/configuration/' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { label: 'Command Reference', link: '/reference/commands/' },
            { label: 'Configuration Reference', link: '/reference/configuration/' },
            { label: 'API Reference', link: '/reference/api/' },
          ],
        },
        {
          label: 'User Guide',
          link: '/user-guide/',
        },
      ],
      customCss: ['./src/styles/custom.css'],
    }),
  ],
});