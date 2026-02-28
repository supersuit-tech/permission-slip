# ADR-006: Design System and Component Library

**Status:** Accepted
**Date:** 2026-02-17
**Authors:** SuperSuit team

## Context

Permission Slip needs a frontend dashboard for users to manage agents, generate invite codes, review approval requests, and monitor activity. The dashboard should feel like a modern SaaS application — clean, minimal, and professional — in the style of products like the Anthropic API console, Linear, Vercel, and Clerk.

The frontend is built with React + TypeScript + Vite. We need to choose a design system / component library that provides:

- A polished, modern SaaS aesthetic out of the box
- Components we'll need immediately: cards, data tables, dropdowns, dialogs, badges, command palette
- Good accessibility defaults (keyboard navigation, screen readers, ARIA)
- Flexibility to customize without fighting the library's opinions
- Active maintenance and community support

---

## Decision: shadcn/ui + Tailwind CSS + Radix UI

Use **shadcn/ui** as our component library, which brings along **Tailwind CSS** for styling and **Radix UI** for accessible primitives. Use **Lucide React** for icons.

### The stack

| Layer | Tool | Role |
|---|---|---|
| **Components** | shadcn/ui | Pre-built, copy-paste components (Button, Card, Table, Dialog, Badge, DropdownMenu, Command, Sheet, etc.) |
| **Primitives** | Radix UI | Underlying headless components providing accessibility, focus management, and keyboard navigation |
| **Styling** | Tailwind CSS | Utility-first CSS framework for rapid, consistent styling |
| **Icons** | Lucide React | Clean line icon set, shadcn/ui's default |
| **Theming** | CSS custom properties | shadcn/ui's theming system uses CSS variables for colors, radii, spacing |

### Why shadcn/ui

1. **Not a dependency — it's your code.** shadcn/ui components are copied into your project (`components/ui/`), not installed as a package. You own the source, can modify anything, and are never blocked by upstream release cycles or breaking changes.

2. **Modern SaaS aesthetic by default.** The default theme — muted borders, clean typography, subtle shadows, restrained color palette — matches the Anthropic dashboard style we're targeting. Minimal customization needed to look polished.

3. **Built on Radix primitives.** Every interactive component (Dialog, DropdownMenu, Popover, etc.) inherits Radix's accessibility work: proper ARIA attributes, focus trapping, keyboard navigation, screen reader support. This is critical for a security-sensitive product like Permission Slip.

4. **Tailwind CSS for styling.** Utility classes enable rapid iteration and ensure visual consistency across the dashboard. Tailwind's constraint-based system (spacing scale, color palette, type scale) prevents ad-hoc values from creeping in.

5. **Composable and dashboard-friendly.** shadcn/ui's component set maps directly to our dashboard needs:
   - **Card** — dashboard summary blocks (pending approvals, registered agents)
   - **Data Table** — approval request queue, agent list
   - **Badge** — status indicators (approved, denied, pending, expired)
   - **Dialog** — confirmation flows (generate invite, revoke agent)
   - **DropdownMenu** — user menu, agent actions
   - **Command** — keyboard-driven command palette
   - **Sheet** — mobile navigation, detail panels

6. **Wide adoption.** shadcn/ui is the most popular component approach in the React + Tailwind ecosystem. This means abundant examples, templates, and community knowledge for dashboard patterns.

---

## Consequences

### Positive

- **Fast to ship.** Pre-built components mean we build dashboard pages, not design primitives.
- **Accessible by default.** Radix handles the hardest accessibility problems (focus management, ARIA, keyboard nav).
- **Full ownership.** No version pinning headaches, no waiting for upstream fixes. We modify components directly.
- **Consistent look and feel.** Tailwind's design tokens + shadcn's theme variables enforce visual consistency.
- **Easy to customize.** CSS variables for theming, Tailwind utilities for one-off adjustments, direct source access for structural changes.

### Negative

- **Initial setup cost.** Tailwind CSS, PostCSS, and the shadcn CLI need to be configured in our Vite project. This is a one-time cost.
- **Tailwind learning curve.** Team members unfamiliar with utility-first CSS will need to adjust. Mitigated by Tailwind's excellent documentation and the fact that shadcn components come pre-styled.
- **Manual updates.** Since components are copied into the project, upstream improvements to shadcn/ui must be manually pulled in. In practice this is rare — once a component works, it rarely needs upstream updates.

### Neutral

- **Bundle size.** We only include the components we use (tree-shaking is inherent to the copy-paste model). Tailwind purges unused utilities in production. Net bundle impact is minimal.
- **No runtime CSS-in-JS.** Tailwind is compiled at build time, avoiding runtime style injection overhead. This is a performance advantage over libraries like Chakra UI or styled-components.

---

## Alternatives Considered

### 1. Material UI (MUI)

The most widely-used React component library. Comprehensive component set, strong accessibility.

**Rejected because:** MUI's visual language is distinctly Google Material Design. Achieving a non-Google aesthetic requires extensive theme overrides and `sx` prop usage that fights the library's defaults. The bundle size is also significantly larger, and the styling approach (Emotion CSS-in-JS) adds runtime overhead.

### 2. Ant Design

Full-featured enterprise component library from Alibaba. Excellent data-dense components (tables, forms, date pickers).

**Rejected because:** Ant Design has a distinctive enterprise/corporate aesthetic that's hard to customize away from. The library is opinionated about layout and interaction patterns in ways that make it difficult to achieve the clean, minimal SaaS look we want. It also adds a large dependency footprint.

### 3. Chakra UI

Well-designed component library with good developer experience and accessibility. Similar composable philosophy to Radix.

**Rejected because:** Chakra UI has lost momentum relative to shadcn/ui in the React ecosystem. Its runtime CSS-in-JS approach (Emotion) is a performance disadvantage compared to Tailwind's build-time compilation. The component set is also less dashboard-oriented than shadcn/ui's recent additions (Data Table, Command palette).

### 4. Headless UI + custom styles

Use Headless UI (from the Tailwind team) for accessible primitives and build all visual styling from scratch.

**Rejected because:** Headless UI has a smaller component set than Radix (no data table, no command palette, limited form components). Building all visual styling from scratch is significantly more work than starting from shadcn/ui's pre-styled components. This approach makes sense for highly custom marketing sites, not for a dashboard where speed to ship matters.

### 5. Build everything from scratch

Use raw HTML elements and Tailwind CSS with no component library.

**Rejected because:** Accessibility is hard to get right. Focus trapping, keyboard navigation, ARIA attributes, and screen reader support require significant expertise and testing. Radix has invested years into these problems — we should leverage that work, not reinvent it.
