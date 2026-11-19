import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';
import { gitConfig } from './shared';
import { CloudWeaveLogo } from '@/components/CloudWeaveLogo';

export function baseOptions(): BaseLayoutProps {
  return {
    nav: {
      title: <CloudWeaveLogo />,
      transparentMode: 'top',
    },
    links: [
      // {
      //   text: 'Ask AI',
      //   url: '#',
      //   icon: (
      //     <svg viewBox="0 0 24 24" className="w-4 h-4 text-brand" fill="none" stroke="currentColor" strokeWidth="2">
      //       <path d="M12 2v4m0 12v4M4.93 4.93l2.83 2.83m8.48 8.48l2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83" />
      //     </svg>
      //   ),
      // },
      // {
      //   text: 'English',
      //   url: '#',
      // },
    ],
    githubUrl: `https://github.com/${gitConfig.user}/${gitConfig.repo}`,
  };
}
