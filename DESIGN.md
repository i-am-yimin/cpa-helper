---
name: CPA-Helper — Looseleaf
description: A paper-and-ink workspace for upstreams, credentials and usage.
colors:
  desk: "#a8c8bb"
  paper: "#f8f5ee"
  slip: "#fbf8f1"
  ink: "#2a2420"
  popover: "#fffdf6"
  primary: "#a84422"
  primary-foreground: "#fffdf6"
  secondary: "#ece6da"
  muted: "#eee9df"
  muted-foreground: "#60584f"
  accent: "#f5e59b"
  accent-foreground: "#2a2420"
  highlight: "#f8dc4c"
  destructive: "#b43a28"
  border: "#c8bfb0"
  input: "#8e8375"
  ring: "#2c4f9e"
  success: "#3b7a4b"
  warning: "#8a6208"
  chart-gold: "#a87f0c"
  chart-purple: "#7c5a87"
  sidebar: "#e7ecdf"
  sidebar-border: "#a4ad9c"
  desk-dark: "#263a33"
  paper-dark: "#292822"
  slip-dark: "#302e27"
  ink-dark: "#f3ebdc"
  popover-dark: "#38352d"
  primary-dark: "#f1a37f"
  primary-foreground-dark: "#2a2420"
  secondary-dark: "#403d32"
  muted-dark: "#38362e"
  muted-foreground-dark: "#c7beaf"
  accent-dark: "#635727"
  accent-foreground-dark: "#fff4cb"
  highlight-dark: "#635727"
  destructive-dark: "#f29b8e"
  border-dark: "#5d574b"
  input-dark: "#8b8170"
  ring-dark: "#9fbce7"
  success-dark: "#a6cb99"
  warning-dark: "#e8ce7e"
  chart-gold-dark: "#e8ce7e"
  chart-purple-dark: "#c9acd4"
  sidebar-dark: "#28342c"
  sidebar-border-dark: "#53604f"
typography:
  headline:
    fontFamily: '"Kevin Sans", "Microsoft YaHei", "PingFang SC", sans-serif'
    fontSize: "29px"
    fontWeight: 700
    lineHeight: 1.18
    letterSpacing: "-0.02em"
  title:
    fontFamily: '"Kevin Sans", "Microsoft YaHei", "PingFang SC", sans-serif'
    fontSize: "18px"
    fontWeight: 500
    lineHeight: 1.375
    letterSpacing: "-0.01em"
  body:
    fontFamily: '"Kevin Sans", "Microsoft YaHei", "PingFang SC", sans-serif'
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.45
  control:
    fontFamily: '"Kevin Sans", "Microsoft YaHei", "PingFang SC", sans-serif'
    fontSize: "14px"
    fontWeight: 500
    lineHeight: "20px"
  label:
    fontFamily: '"Kevin Sans", "Microsoft YaHei", "PingFang SC", sans-serif'
    fontSize: "12px"
    fontWeight: 500
    lineHeight: "16px"
  mono:
    fontFamily: '"Cascadia Mono", "SFMono-Regular", Consolas, monospace'
rounded:
  base: "0.5rem"
  button: "5px 8px 6px 9px / 8px 5px 9px 6px"
  field: "5px 8px 4px 7px"
  card: "7px 11px 6px 9px"
  overlay: "8px 11px 7px 9px"
  badge: "4px 6px 3px 5px"
  navigation-active: "4px 9px 5px 7px"
spacing:
  xs: "4px"
  sm: "8px"
  control-inline: "10px"
  md: "12px"
  lg: "16px"
  panel: "20px"
  xl: "24px"
  canvas-inline: "32px"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.primary-foreground}"
    typography: "{typography.control}"
    rounded: "{rounded.button}"
    padding: "0 10px"
    height: "36px"
  button-outline:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    typography: "{typography.control}"
    rounded: "{rounded.button}"
    padding: "0 10px"
    height: "36px"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    typography: "{typography.control}"
    rounded: "{rounded.button}"
    padding: "0 10px"
    height: "36px"
  input:
    backgroundColor: "{colors.popover}"
    textColor: "{colors.ink}"
    rounded: "{rounded.field}"
    padding: "4px 10px"
    height: "36px"
  navigation-active:
    backgroundColor: "{colors.highlight}"
    textColor: "{colors.ink}"
    typography: "{typography.control}"
    rounded: "{rounded.navigation-active}"
    padding: "8px"
    height: "32px"
  badge-outline:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.badge}"
    padding: "2px 8px"
    height: "20px"
  card:
    backgroundColor: "{colors.slip}"
    textColor: "{colors.ink}"
    rounded: "{rounded.card}"
    padding: "20px"
  tab-active:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    typography: "{typography.control}"
    rounded: "0"
    padding: "10px 3px"
---

# Design System: CPA-Helper

## Overview

**Creative North Star: "Looseleaf — paper, ink and a working desk"**

The user-selected Looseleaf component-library language makes CPA-Helper a quiet working desk: warm paper, sage surroundings, readable ink and small yellow highlights. Slightly uneven outlines give controls a human character while the layout remains orderly and dense enough for daily administration. Retain the existing CPA-Helper logo.

Source data, quotas, permissions and missing-data truth take precedence over decoration. Keep labels, numbers and state explanations easy to compare. Dark mode is a warm-paper variant of the same workspace, with lighter ink and restrained highlights.

**Key Characteristics:**
- Warm paper and sage surfaces with clear ink contrast.
- Controlled irregular corners and lightweight SVG ink outlines.
- Compact controls, calm tables and tabular numbers.
- Explicit operational states in both Chinese and English.

Implementation authority is `frontend/src/styles/looseleaf.css`, loaded after `tokens.css` by `main.ts`. This document records that implementation; update the CSS first, then refresh both this file and `.impeccable/design.json`. The `-dark` colors are the `:root.dark` counterparts; component tokens describe light defaults, while runtime CSS variables select the theme.

## Colors

### Primary

Burnt clay (`primary`) marks standard primary actions and the active tab underline. Its peach counterpart serves the same role in dark mode. The upstream configuration action has an intentional yellow highlight override.

### Secondary

Sage (`desk`, `sidebar`) frames the workspace; warm neutral `secondary` supports quiet actions. Yellow `highlight` identifies selected sidebar items and text selection; softer `accent` carries notices such as unsaved changes. Use the corresponding foreground tokens, especially in dark mode.

### Tertiary

Green `success`, ochre `warning` and red `destructive` convey state alongside text. Blue `ring` is the keyboard-focus color and third chart series. Chart order is primary, success, ring, chart-gold, chart-purple; the purple token also backs the existing purple accent.

### Neutral

`paper` is the main canvas and header, `slip` is the card surface, and `popover` backs fields and overlays. `ink` supplies text; `muted-foreground` supplies secondary descriptions. `border` separates rows and panels, `input` provides stronger field strokes, and `sidebar-border` separates the sage navigation.

**The State Label Rule.** Pair status color with a label or value. A missing quota stays unavailable; it must never become a reassuring full bar.

## Typography

Kevin Sans is served locally from `frontend/public/fonts/looseleaf/` under its included MIT license. Regular files cover declared weights (400–600); bold files cover (700–900). Microsoft YaHei and PingFang SC provide Chinese fallback. Preserve this complete stack; code and technical identifiers use the existing monospace stack.

Use `headline` for page titles, `title` for card headings, `body` for the root reading size, `control` for compact interface text and `label` for badges. Tables and many controls use the smaller interface size rather than the root body size. Inputs retain the larger text size below the Tailwind `md` breakpoint. Page-title size remains the Looseleaf value on narrow screens; the inherited compact line-height becomes (1.16) at widths up to (720px).

Use tabular numerals in table cells, quota values, priorities and usage summaries. Keep English and Chinese copy at the same hierarchy; allow secondary descriptions and long model patterns to wrap rather than shrinking the type.

## Layout

The shell fills the viewport with a fixed header and an independently scrolling content area. Desktop navigation supports an expanded rail (16rem) and icon rail (3rem); the mobile navigation sheet is (18rem). Desktop header height is (56px), with content insets (28px 32px 36px). The page container fills available width up to (1920px). The main page rhythm is (20px), reducing to (12px) at widths up to (720px).

At shell widths up to (768px), the header uses the centered compact brand treatment and content insets become (14px 16px 18px). Keep actions and filters wrapping within their container. The upstream table uses a minimum width of (940px) with horizontal scrolling contained inside the table wrapper on tablet widths. Below (768px), its header disappears and each upstream becomes a vertical, card-like grid row; details and group-policy fields stack. Do not replace this with viewport-wide overflow.

Default buttons and fields are (36px) high; sidebar entries are deliberately compact (32px). Existing explicit small, extra-small and icon button sizes remain supported. Cards use (20px) vertical and content padding, or (16px) in their small variant.

## Elevation & Depth

Depth comes primarily from paper tone, ink edges and fine separators. Cards use `--cpa-shadow-card`; quieter lifted surfaces can use the existing `--cpa-shadow-soft`. Solid buttons have a shallow contact shadow, then move down (1px) and lose that shadow when pressed. The exact shadow values live in the sidecar. Headers are opaque paper with no backdrop blur.

**The Paper Depth Rule.** Keep shadows shallow and structural; preserve readable boundaries in dark mode and forced-color mode. Forced-color mode supplies a system-color border and removes card/button shadows.

## Shapes

Use the role-specific asymmetric radii in the frontmatter. Buttons and cards add `InkOutline.vue`, a resize-aware, noninteractive SVG path with a non-scaling stroke. Button ink uses the foreground stroke (1.15px); subtle card ink uses the input stroke (1px). The outline is hidden from assistive technology and does not intercept pointer events. CSS suppresses the duplicate native border when the ink SVG is present.

Tabs remain flat with a straight underline; do not add ink outlines to every separator. Regular paper panels use their normal fine border. Retain the logo asset inside the existing small framed mark.

## Components

### Buttons

Primary buttons use clay and its matching foreground. Outline buttons use the canvas with a muted hover surface; their existing dark variant uses a translucent input-colored surface. Ghost and link variants omit the ink outline and contact shadow. Secondary buttons use the secondary surface; destructive buttons use a tinted destructive surface and explicit wording. Reuse the current variant API rather than inventing a second button family.

State transitions last (150ms). Keep the pressed displacement, disabled opacity (50%) and disabled interaction handling. Preserve the global focus-visible outline (2px, offset 3px) and existing invalid-state semantics. Icon-only actions need accessible names.

### Inputs / Fields

Fields use popover paper, an input-colored stroke and the field radius. Keep keyboard focus visible and retain invalid-state attributes with readable error text; disabled fields retain the existing opacity and interaction rules. Keep visible labels or accessible names. Group priorities and model patterns must preserve edit permissions and draft/save feedback.

### Cards / Containers

Cards are lightly raised slips with subtle ink outlines. Default content uses the panel spacing; small cards use the compact variant. Paper panels share the card surface and radius with a regular border. Use dashed separators for the implemented grouped-detail patterns rather than adding nested decorative cards around every value.

### Navigation / Tabs

Sidebar entries use sage at rest and yellow when active; icon collapse keeps tooltips. Admin and account navigation follow the current permission checks. Tabs use muted labels, a primary underline and ink text when selected; narrow upstream tabs scroll within their own strip. Keep Reka UI keyboard and focus behavior.

### Badges / Data States

Badges are compact labels with their own irregular radius. Use outline badges for quiet states; reserve tinted destructive treatment for an actual problem. Preserve loading, empty, filtered-empty, unavailable, stale, error and retry messages. Show the quota source/window and plugin pending/effective distinctions; do not infer business status from fixture screenshots. Scheduling and validation details belong in `docs/antigravity-group-scheduler.md`.

Motion supports interaction: controls transition quickly, routes fade with a small vertical displacement, and pending navigation has a slim progress indicator. Respect the existing reduced-motion rules, which shorten global transitions and suppress the route-progress animation. Exact timings and breakpoints are in the sidecar.

## Do's and Don'ts

### Do:

- **Do** reuse the Looseleaf variables, local font stack and existing logo.
- **Do** preserve compact density, contained table scrolling and the upstream mobile row layout.
- **Do** put source, quota window, permissions and missing-data explanations ahead of decoration.
- **Do** keep focus, invalid, disabled and reduced-motion states visible and functional.
- **Do** use irregular ink outlines where the existing button and card primitives supply them.

### Don't:

- **Don't** substitute a generic cool-blue or glass visual system for the chosen paper-and-ink world.
- **Don't** turn missing quota or usage data into zero, a full meter or a forecast.
- **Don't** use color alone to communicate health, selection or errors.
- **Don't** let decoration obscure labels, values or actionable state explanations.
- **Don't** treat fixture screenshots or preview snippets as evidence of live accounts, routing or quota results.
