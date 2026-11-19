import React from 'react';
import defaultMdxComponents from 'fumadocs-ui/mdx';
import type { MDXComponents } from 'mdx/types';

function cn(...classes: (string | undefined | null | false)[]) {
  return classes.filter(Boolean).join(' ');
}

export function getMDXComponents(components?: MDXComponents) {
  const DefaultH1 = defaultMdxComponents.h1;
  const DefaultH2 = defaultMdxComponents.h2;
  const DefaultH3 = defaultMdxComponents.h3;
  const DefaultH4 = defaultMdxComponents.h4;
  const DefaultA = defaultMdxComponents.a;
  const DefaultCard = defaultMdxComponents.Card;

  return {
    ...defaultMdxComponents,

    h1: ({ className, ...props }) => (
      <DefaultH1
        className={cn(
          'text-[32px] leading-[1.2] font-semibold tracking-[-0.02em] text-heading mt-8 mb-3 scroll-mt-24',
          className
        )}
        {...props}
      />
    ),

    h2: ({ className, ...props }) => (
      <DefaultH2
        className={cn(
          'text-[22px] leading-[1.3] font-semibold text-heading mt-[56px] mb-[16px] scroll-mt-24',
          className
        )}
        {...props}
      />
    ),

    h3: ({ className, ...props }) => (
      <DefaultH3
        className={cn(
          'text-[17px] leading-[1.4] font-semibold text-heading mt-[32px] mb-[12px] scroll-mt-24',
          className
        )}
        {...props}
      />
    ),

    h4: ({ className, ...props }) => (
      <DefaultH4
        className={cn(
          'eyebrow mt-6 mb-2 scroll-mt-24',
          className
        )}
        {...props}
      />
    ),

    // Paragraph text (15px / 1.75 / 400, color --text-muted, tight gap before blocks)
    p: ({ className, ...props }) => (
      <p
        className={cn('text-[15px] text-muted leading-[1.75] font-normal mb-3', className)}
        {...props}
      />
    ),

    // Strong text (weight 500, color --text. Never 700)
    strong: ({ className, ...props }) => (
      <strong className={cn('text-text font-medium', className)} {...props} />
    ),

    // Links with brand accent
    a: ({ className, ...props }) => (
      <DefaultA
        className={cn(
          'text-brand hover:underline underline-offset-4 decoration-brand-border hover:decoration-brand transition-colors font-medium',
          className
        )}
        {...props}
      />
    ),

    // 2-Column Cards Grid matching design tokens
    Cards: ({ className, ...props }) => (
      <div
        className={cn('grid grid-cols-1 md:grid-cols-2 gap-4 my-6', className)}
        {...props}
      />
    ),

    Card: ({ className, title, description, children, href, ...props }) => (
      <DefaultCard
        title={title}
        description={description}
        href={href}
        className={cn(
          'rounded-lg border border-border bg-surface hover:border-brand-border p-4 transition-colors text-left shadow-none group',
          className
        )}
        {...props}
      >
        {children}
      </DefaultCard>
    ),

    // Minimal Blockquote: at most one 1px border, no nested boxes
    blockquote: ({ className, ...props }) => (
      <blockquote
        className={cn(
          'border-l-2 border-brand pl-4 py-1 my-6 text-muted italic text-[14px] leading-relaxed',
          className
        )}
        {...props}
      />
    ),

    // Horizontal rules between sections: remove, spacing does that job
    hr: () => null,

    // Tables: header 13px/500/--text-muted, cells 14px/400, first column 500 (not bold).
    // Only horizontal 1px --border lines, no vertical lines, no outer card.
    table: ({ className, ...props }) => (
      <div className="w-full mt-4 mb-6 overflow-x-auto">
        <table className={cn('w-full border-collapse text-left bg-transparent border-none', className)} {...props} />
      </div>
    ),
    th: ({ className, ...props }) => (
      <th
        className={cn(
          'border-b border-border py-2 px-3 first:pl-0 last:pr-0 text-[13px] font-medium text-muted bg-transparent border-t-0 border-l-0 border-r-0 text-left',
          className
        )}
        {...props}
      />
    ),
    td: ({ className, ...props }) => (
      <td
        className={cn(
          'border-b border-border py-2.5 px-3 first:pl-0 last:pr-0 text-[14px] font-normal text-muted first:font-medium first:text-text bg-transparent border-t-0 border-l-0 border-r-0 text-left',
          className
        )}
        {...props}
      />
    ),

    ...components,
  } satisfies MDXComponents;
}

export const useMDXComponents = getMDXComponents;

declare global {
  type MDXProvidedComponents = ReturnType<typeof getMDXComponents>;
}
