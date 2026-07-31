# Project agent context

## Stack

- **Framework**: Next.js (App Router), TypeScript
- **Styles**: Tailwind CSS v4 — config is CSS-first via `@theme` in `globals.css`, no `tailwind.config.ts`
- **Components**: shadcn/ui (Radix primitives) — components live in `src/components/ui/`, owned and editable
- **Package manager**: pnpm — always use `pnpm`, never `npm` or `yarn`
- **Deployment**: Vercel

## Setup (for new projects)

```bash
# 1. Scaffold
pnpm create next-app@latest my-app --yes
cd my-app

# 2. shadcn
pnpm dlx shadcn@latest init
# → accept defaults: Slate base, CSS variables yes, pick any style

# 3. Prettier
pnpm add -D prettier prettier-plugin-tailwindcss eslint-config-prettier
```

Add `"prettier"` to the end of `extends` in your ESLint config to prevent conflicts.

## Config files

**.prettierrc**
```json
{
  "plugins": ["prettier-plugin-tailwindcss"],
  "tailwindStylesheet": "./src/app/globals.css",
  "semi": false,
  "singleQuote": true,
  "printWidth": 100
}
```

**package.json** — enforce package manager and Node version:
```json
"engines": {
  "node": ">=20",
  "pnpm": ">=9"
}
```

**.npmrc** — prevent accidental npm installs:
```ini
engine-strict=true
```

## Fonts

MatterXH and MatterSemiMono are commercial fonts, not on Google Fonts — self-host them
with `next/font/local` instead of `next/font/google`. Never use `@fontsource` or CDN
font links in production Next.js.

Font files live under `src/app/fonts/` (`.otf`/`.ttf` as supplied — swap in `.woff2`
if/when available for smaller payloads).

```tsx
// src/app/layout.tsx
import localFont from 'next/font/local'

const display = localFont({
  src: [
    { path: './fonts/MatterXHLight.otf', weight: '300', style: 'normal' },
    { path: './fonts/MatterXHRegular.otf', weight: '400', style: 'normal' },
    { path: './fonts/MatterXHMedium.ttf', weight: '500', style: 'normal' },
  ],
  variable: '--font-display',
})

const body = localFont({
  src: [
    { path: './fonts/MatterSemiMonoRegular.otf', weight: '400', style: 'normal' },
    { path: './fonts/MatterSemiMonoMedium.otf', weight: '500', style: 'normal' },
  ],
  variable: '--font-sans',
})
```

Apply both variables to `<html>`: `className={`${display.variable} ${body.variable}`}`

## Tailwind theme — brand tokens

Add to the `@theme` block in `globals.css`, alongside whatever shadcn generated:

```css
@theme inline {
  /* Fonts — reference CSS vars injected by next/font */
  --font-display: var(--font-display);
  --font-sans:    var(--font-sans);

  /* Brand colours — customise per project */
  --color-midnight:  #071524;  /* dark hero bg    */
  --color-sky:       #7cc4e8;  /* accent on dark  */
  --color-sky-text:  #1565a0;  /* accent on light, accessible */
}
```

Classes `font-display`, `font-sans`, `bg-midnight`, `text-sky`, `text-sky-text` etc.
are then available automatically as Tailwind utilities.

## File structure

```
src/
  app/
    layout.tsx          ← fonts, metadata, Nav + Footer wrappers
    globals.css         ← Tailwind import, @theme tokens, shadcn vars
    page.tsx
    [route]/
      page.tsx
  components/
    nav.tsx             ← 'use client' for usePathname active state
    footer.tsx
    ui/                 ← shadcn-generated, commit and edit freely
```

## Conventions

- **Server components by default** — only add `'use client'` where needed (nav active state, forms, interactivity)
- **Metadata**: use `export const metadata` per page, with `template: '%s — Site Name'` in root layout
- **Images**: use `next/image` for local assets; pull remote images into `/public` before go-live
- **Routing**: always use proper App Router routes (`/app/[route]/page.tsx`), not single-page anchor scrolling
- **shadcn components**: added individually via `pnpm dlx shadcn@latest add [component]`, not bulk-installed
- **`cn()` utility** (clsx + tailwind-merge): use for all conditional class merging, never string concatenation

## Accessible colour usage

- Light backgrounds: use `text-sky-text` (`#1565a0`) — passes WCAG AA
- Dark backgrounds (`bg-midnight`): use `text-sky` (`#7cc4e8`) — passes WCAG AA
- Never use `text-sky` on light or `text-sky-text` on dark

## Not included / deliberate omissions

- **Biome**: not used — `prettier-plugin-tailwindcss` class sorting has no Biome equivalent yet
- **T3 stack / tRPC**: not needed for marketing/content sites
- **`tailwind.config.ts`**: not needed in Tailwind v4 — all config in CSS
