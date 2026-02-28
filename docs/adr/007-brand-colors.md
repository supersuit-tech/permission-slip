# ADR-007: Brand Color Palette

**Status:** Accepted
**Date:** 2026-02-18
**Authors:** SuperSuit team

## Context

Permission Slip needs a distinctive visual identity that conveys trust, authority, and clarity. As an authorization middle-man between AI agents and external services, the product must feel both premium and approachable. The dashboard UI (built with shadcn/ui + Tailwind CSS per ADR-006) needs a cohesive color palette that:

- Differentiates Permission Slip from competitors in the auth/permissions space
- Communicates security and authority without feeling cold or corporate
- Works well for a dashboard-heavy product where clarity is paramount
- Maps cleanly to shadcn/ui's CSS custom property theming system

---

## Decision: Purple + Gold on White

Adopt a **Royal Purple + Rich Gold** palette on a **white/lavender** canvas.

### The palette

| Role | Color | Hex | OKLCH | Usage |
|---|---|---|---|---|
| **Background** | Light Lavender | `#F5F0FA` | `oklch(0.962 0.014 308.296)` | Page background — ties to purple without being heavy |
| **Surface** | White | `#FFFFFF` | `oklch(1 0 0)` | Cards, modals, form fields, navbar |
| **Primary** | Royal Purple | `#6A2C91` | `oklch(0.430 0.162 309.325)` | Nav accents, headings, primary buttons, links, focus rings |
| **Secondary** | Rich Gold | `#D4A843` | `oklch(0.753 0.127 84.932)` | CTAs, badges ("Approved"), highlights, active indicators |
| **Text** | Near Black | `#1A1A2E` | `oklch(0.228 0.038 282.932)` | Body text — slightly warm dark for readability |
| **Muted** | Cool Gray | `#9CA3AF` | `oklch(0.714 0.019 261.325)` | Secondary text, inactive nav items, subtle borders |

### Why this works for Permission Slip

1. **Purple as primary accent.** Authority, security, and "granting permission" all feel right. Purple is historically associated with governance and trust — fitting for a product that mediates access control.

2. **Gold as secondary accent.** Signals approval, premium quality, and draws the eye to key actions (buttons, badges, status indicators). The "approved" badge in gold is immediately legible as a positive status.

3. **White-dominant layout.** Permissions UIs need to feel clear and uncluttered. White cards on a subtle lavender background let the content breathe while maintaining visual hierarchy.

4. **The combination is distinctive.** Almost no one in the auth/permissions space uses purple + gold. Products like Auth0 (blue), Okta (blue), Clerk (purple/blue) — this palette is instantly recognizable and avoids category confusion.

5. **Accessible by default.** Near Black (#1A1A2E) on White (#FFFFFF) exceeds WCAG AAA contrast requirements. Royal Purple on White meets AA for large text. Gold is used for accents and badges where contrast requirements are less strict.

### Where each color lives

| Element | Color(s) |
|---|---|
| Page background | Light Lavender |
| Cards, navbar, modals | White surface with visible borders |
| Primary buttons | Purple background, white text |
| Secondary buttons / CTAs | Gold background, dark text |
| Badges (approved, pro) | Gold background, dark text |
| Active nav indicator | Gold underline |
| Table headers | Purple background, white text |
| Table row alternation | Light Lavender stripes |
| Focus rings | Purple |
| Headings, links | Purple or Near Black |
| Body text | Near Black |
| Secondary/muted text | Cool Gray |

---

## Implementation

Colors are stored as CSS custom properties in OKLCH color space (matching shadcn/ui's theming approach) in `frontend/src/index.css`. They map to shadcn/ui's semantic token system:

```
--primary       → Royal Purple
--secondary     → Rich Gold
--background    → Light Lavender (page)
--card          → White (surfaces)
--foreground    → Near Black
--muted-foreground → Cool Gray
```

No changes to shadcn/ui component source code are required — the palette integrates entirely through CSS custom properties.

---

## Consequences

### Positive

- **Immediate brand recognition.** Purple + gold is distinctive in the auth/permissions market.
- **Clean information hierarchy.** Lavender background → white cards → purple/gold accents creates clear visual layers.
- **Seamless design system integration.** Maps directly to shadcn/ui's existing CSS variable theming without modifying any component code.
- **Dashboard-friendly.** The muted lavender background and white cards reduce eye strain for users who spend extended time in the dashboard.

### Negative

- **Purple requires careful dark mode treatment.** A bright purple that works on white needs to be lightened for dark backgrounds. Dark mode values will need separate tuning.
- **Gold can be tricky for colorblind users.** Gold/amber can appear similar to certain greens for deuteranopia. Badges should include text labels, not rely on color alone (already the case with shadcn/ui's Badge component).

### Neutral

- **Limited to six core colors.** This is intentional — a small palette enforces consistency and prevents ad-hoc color choices from creeping in. Additional shades can be derived via OKLCH lightness adjustments if needed.

---

## Alternatives Considered

### 1. Blue + White (Auth0 / Okta style)

The industry default for auth products. Clean and professional.

**Rejected because:** Every auth product uses blue. Permission Slip would be visually indistinguishable from Auth0, Okta, and dozens of others. Blue also carries a "corporate enterprise" connotation that doesn't match our product personality.

### 2. Green + Dark (Matrix / terminal aesthetic)

Technical, developer-focused, security-oriented feel.

**Rejected because:** Too niche and potentially off-putting for non-technical users who will interact with Permission Slip's approval flows. Also implies "hacker tool" rather than "trusted middle-man."

### 3. Monochrome (Linear / Vercel style)

Black, white, and grays only. Ultra-minimal and modern.

**Rejected because:** While elegant, a monochrome palette makes it harder to create visual hierarchy in a dashboard with many status states (approved, denied, pending, expired). Permission Slip needs color to communicate status at a glance.
