---
name: CPA-Helper — BoardUI
description: A readable BoardUI workspace for upstream connections, quota and routing priorities.
colors:
  primary: oklch(54.6% 0.245 262.881)
  primary-deep: oklch(48.8% 0.243 264.376)
  primary-pressed: oklch(42.4% 0.199 265.638)
  primary-foreground: "#fff"
  ring: oklch(62.3% 0.214 259.815)
  background: "#f7f7f7"
  card: "#fff"
  foreground: oklch(14.5% 0 none)
  muted-foreground: oklch(43.9% 0 none)
  border: "#ebebeb"
  input: oklch(87% 0 none)
  accent: oklch(97% 0.014 254.604)
  accent-foreground: oklch(48.8% 0.243 264.376)
  destructive: color-mix(in srgb, oklch(50.5% 0.213 27.518) 80%, oklch(43.9% 0 none))
  success: oklch(52.7% 0.154 150.069)
  warning: oklch(47.3% 0.137 46.201)
  background-dark: oklch(20.5% 0 none)
  card-dark: oklch(26.9% 0 none)
  foreground-dark: oklch(98.5% 0 none)
  muted-foreground-dark: oklch(70.8% 0 none)
  border-dark: oklch(37.1% 0 none)
  input-dark: oklch(55.6% 0 none)
  primary-dark: "#3392ff"
  accent-dark: color-mix(in srgb, oklch(62.3% 0.214 259.815) 15%, oklch(26.9% 0 none))
  accent-foreground-dark: oklch(80.9% 0.105 251.813)
  destructive-dark: color-mix(in srgb, oklch(63.7% 0.237 25.331) 80%, oklch(55.6% 0 none))
  success-dark: oklch(79.2% 0.209 151.711)
  warning-dark: oklch(82.8% 0.189 84.429)
  chart-teal: oklch(70.4% 0.14 182.503)
  chart-lime: oklch(64.8% 0.2 131.684)
  chart-pink: oklch(65.6% 0.241 354.308)
  chart-sky: oklch(68.5% 0.169 237.323)
  chart-purple: oklch(62.7% 0.265 303.9)
typography:
  headline:
    fontFamily: '"Inter Variable", "Microsoft YaHei", "PingFang SC", sans-serif'
    fontSize: 28px
    fontWeight: 700
    lineHeight: 1.18
    letterSpacing: -0.03em
  headline-mobile:
    fontFamily: '"Inter Variable", "Microsoft YaHei", "PingFang SC", sans-serif'
    fontSize: 22px
    fontWeight: 700
    lineHeight: 1.16
    letterSpacing: -0.03em
  title:
    fontFamily: '"Inter Variable", "Microsoft YaHei", "PingFang SC", sans-serif'
    fontSize: 18px
    fontWeight: 600
    lineHeight: 1.375
    letterSpacing: -0.02em
  canvas-body:
    fontFamily: '"Inter Variable", "Microsoft YaHei", "PingFang SC", sans-serif'
    fontSize: 16px
    fontWeight: 400
    lineHeight: 1.45
  body:
    fontFamily: '"Inter Variable", "Microsoft YaHei", "PingFang SC", sans-serif'
    fontSize: 14px
    fontWeight: 400
    lineHeight: 20px
  control:
    fontFamily: '"Inter Variable", "Microsoft YaHei", "PingFang SC", sans-serif'
    fontSize: 14px
    fontWeight: 500
    lineHeight: 20px
    letterSpacing: "0"
  label:
    fontFamily: '"Inter Variable", "Microsoft YaHei", "PingFang SC", sans-serif'
    fontSize: 12px
    fontWeight: 500
    lineHeight: 16px
  mono:
    fontFamily: '"Cascadia Mono", "SFMono-Regular", Consolas, monospace'
rounded:
  xs: 4px
  sm: 8px
  control: 10px
  floating: 12px
  card: 24px
  pill: 999px
spacing:
  xs: 4px
  sm: 8px
  control-inline: 10px
  md: 12px
  lg: 16px
  panel: 20px
  xl: 24px
  canvas-inline: 32px
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.primary-foreground}"
    typography: "{typography.control}"
    rounded: "{rounded.control}"
    padding: 0 10px
    height: 36px
  button-outline:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    typography: "{typography.control}"
    rounded: "{rounded.control}"
    padding: 0 10px
    height: 36px
  button-ghost:
    backgroundColor: transparent
    textColor: "{colors.foreground}"
    typography: "{typography.control}"
    rounded: "{rounded.control}"
    padding: 0 10px
    height: 36px
  input:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.control}"
    padding: 4px 10px
    height: 36px
  navigation-active:
    backgroundColor: "{colors.background}"
    textColor: "{colors.foreground}"
    typography: "{typography.control}"
    rounded: "{rounded.control}"
    padding: 8px
    height: 32px
  badge-outline:
    backgroundColor: transparent
    textColor: "{colors.foreground}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: 2px 8px
    height: 20px
  card:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.card}"
    padding: 20px
  tab-active:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    typography: "{typography.control}"
    rounded: "{rounded.sm}"
    padding: 8px 12px
  quota-pause:
    backgroundColor: "color-mix(in oklch, oklch(47.3% 0.137 46.201) 14%, #f7f7f7)"
    textColor: "{colors.warning}"
    rounded: "{rounded.floating}"
    padding: 12px 16px
---

# Design System: CPA-Helper

## Overview

**Creative North Star: "BoardUI — a readable routing workspace"**

BoardUI is the confirmed visual authority: neutral work surfaces, Inter, blue interactions, regular rounded geometry and restrained contact shadows. The system keeps dense administrative information readable while leaving the routing controls within reach.

The MIT-licensed BoardUI theme is adapted to the existing Vue 3, Reka UI and Tailwind 4 application. Reuse the shared primitives and application aliases so the light and dark themes remain coherent.

This records the implemented system after the approved replacement of the earlier design. Source authority: [BoardUI](https://github.com/BoardUI/boardui), pinned to commit `ec5c5eb3e29c2441be63b9422cd54f411ef23466`. The vendored MIT notice is in `frontend/src/styles/boardui/LICENSE` and `frontend/public/licenses/boardui.txt`. `frontend/src/styles/boardui/theme.css` supplies the upstream theme; `typography.css` supplies the upstream type utilities; `frontend/src/styles/boardui.css` supplies the final Vue application aliases and component adaptations. This was an explicit library choice, with no generated visual comp or directional seed.

**Key Characteristics:**

- Neutral canvas with distinct, opaque working surfaces.
- Inter text with a compact operational type hierarchy.
- Blue interactive states, regular corners and shallow contact shadows.
- Responsive controls and explicit, readable data states.

## Colors

The palette is neutral white and charcoal with a blue interaction accent. Status and chart colors carry data meaning. Frontmatter values resolve the final semantic aliases to the imported Tailwind 4 primitives, retaining BoardUI's explicit neutral and blue overrides.

### Primary

- **Interaction blue:** primary actions, links and focus treatment. The solid primary token is the first stop of the primary button; its gradient continues to primary-deep. Hover holds the primary tone and active progresses from primary-deep to primary-pressed. The darker stops are a deliberate application adaptation for white-label contrast.
- **Soft blue selection:** accent surfaces and their paired text; dark mode mixes the blue interaction tone into the card surface.

### Neutral

- **Working canvas:** the background and muted surfaces; **raised surface:** cards, popovers and the sidebar.
- **Primary ink:** foreground text; **supporting ink:** muted text. Borders and input edges use distinct neutral steps.
- Dark mode uses charcoal canvas, a lighter charcoal card, near-white ink and the paired dark muted/border/input tokens. It is a semantic theme, not a filter applied to light colors.

### Semantic data colors

- **Success green** and **warning amber** communicate actual states alongside words. Destructive text uses BoardUI's red/neutral mix.
- The five chart accents are teal, lime, pink, sky and purple; their aliases remain separate from the interaction accent.

**The Semantic Alias Rule.** Use the application aliases for surfaces and text; boardui.css is the final application mapping over the imported BoardUI theme.

**The Contrast Rule.** Keep the implemented darker primary gradients with white labels, neutral-600 muted text in light mode, neutral-400 in dark mode, and amber-800 / amber-400 warning text.

## Typography

**Display and body font:** locally bundled Inter Variable, from `@fontsource-variable/inter` (installed version 5.3.0), with Microsoft YaHei and PingFang SC for Chinese, then sans-serif.

**Label font:** the same Inter stack. **Code font:** Cascadia Mono / SFMono-Regular / Consolas / monospace. The upstream theme's Next.js loading comments describe its original environment; CPA-Helper loads Inter through its local CSS package.

### Hierarchy

- **Headline:** page titles use the headline role, reducing to headline-mobile at the small-page breakpoint.
- **Title:** card titles use a restrained semibold heading with tighter tracking.
- **Canvas body:** global body text; **body:** compact table and component copy; **control:** medium-weight button and tab labels; **label:** compact metadata and badges.
- Imported BoardUI utilities include a broader type ramp; the reused small-button control uses its body-medium utility. Do not promote the auth wordmark's oversized type into a general heading role.
- Quota and table values use tabular numbers; identifiers can use the existing code stack.

**The One Family Rule.** Use the local Inter stack for headings, body and controls; reserve the existing monospace stack for code and machine-readable values.

## Layout

The shell fills the viewport with a fixed-height header (56px) and an independently scrolling content area. Desktop navigation supports an expanded rail (16rem) and icon rail (3rem); the mobile sheet is (18rem). Content insets are (28px 32px 36px), and the page fills available width up to (1920px). The shared page gap is (20px), reducing to (12px) at viewport widths up to (720px); page titles also reduce at this breakpoint. At shell widths up to (768px), content insets become (14px 16px 18px).

The upstream hub owns an inline-size container. At available content widths up to (959px), the desktop table becomes labelled stacked rows, group policies become a single column, actions wrap and filters remain usable. Its desktop table minimum is (940px); the narrow layout removes that minimum. Use this content-based threshold rather than assuming a tablet viewport leaves enough space beside the sidebar.

Default cards use the panel spacing with a compact variant (16px vertical padding). The default button and input height is (36px); small buttons are (32px), extra-small buttons (24px). Icon-only sizes remain those exposed by the shared button API.

The authentication shell uses a neutral brand panel beside an opaque form surface. At viewport widths up to (900px), it hides the brand panel and presents the form in one column.

## Elevation & Depth

Depth combines tonal surfaces, fine borders and shallow contact shadows. Cards use the BoardUI card contact shadow; floating menus use its two-layer dropdown shadow; fields use its inset shadow. Dark mode preserves the geometry and raises shadow alpha. Exact shadow recipes are in the sidecar.

**The Contact Depth Rule.** Use the shared contact and dropdown shadows to separate surfaces; headers stay opaque and the application header has no backdrop blur.

Motion supports state changes: button adaptation uses (160ms) transitions, the primary gradient's hover overlay fades over (150ms), and route transitions use (180ms) opacity plus a small vertical displacement (6px). The global reduced-motion rule shortens animations and transitions and uses automatic scrolling. Forced-color mode provides system-color borders and removes card/button shadows.

## Shapes

Use regular rounded rectangles. Cards, ordinary panels, dialogs and sheets share the large card radius; dropdowns, popovers and select panels use the floating radius. Default buttons and input surfaces use the control radius; small buttons use the small radius; extra-small buttons use the extra-small radius. Badges are pills. Tabs have a softly rounded track with inset, rounded active segments.

These are source-backed role choices. The logo stays inside its framed mark; large authentication lettering is local brand treatment.

## Components

### Buttons

Compact, clear controls with a subtle physical edge. Primary buttons use the blue gradient and white text, with the disabled gradient and disabled label token supplied by BoardUI. Outline and secondary variants use the raised surface, neutral border and a subtle neutral hover. Ghost buttons reveal a muted hover surface; links use the interaction color and an underline on hover. Destructive actions use a tinted destructive surface.

Reuse the shared variant and size API. Preserve the (3px) focus-visible ring, invalid-state feedback, disabled interaction handling and (50%) disabled opacity. Ordinary pressable buttons translate downward (1px); popup triggers retain their popup semantics. Icon-only actions require accessible names.

### Inputs / Fields

Opaque card surface, input-colored stroke, inset shadow and a compact minimum height. Desktop text is compact; narrow screens retain the larger input text from the shared primitive. Keep visible labels, focus rings and error wording. Select triggers, textareas and input groups share the field treatment.

### Cards / Containers

Neutral work surfaces with a fine border and contact shadow. Standard cards use panel padding, with the existing small-card variant for denser content. Do not add repeated decorative containers around each value; expanded hub detail and quota rows already use ordinary separators.

### Navigation / Tabs

The sidebar is an opaque raised surface. Active entries use a muted neutral fill and semibold ink; icon collapse keeps its tooltip behavior. Tabs sit inside a muted, bordered track; the active segment uses the card surface, foreground ink and contact shadow. Preserve Reka keyboard navigation and focus behavior. Narrow hub tabs scroll within their strip.

### Badges / Data States

Compact pill labels use outline, neutral, interaction or destructive treatments according to state. Use readable state wording with each color. Keep loading, empty, filtered-empty, unknown, stale, failure, retry and pending-versus-confirmed synchronization states explicit.

### Quota Policy Editor

The editor combines labelled window toggles, compact numeric fields, mode selection and explicit draft/save actions. Paused state uses the floating radius, a weak amber surface and warning text; its message includes the reason and synchronization state. Inherit/custom/off modes and manual-resume behavior are product rules in PRODUCT.md, not visual defaults. On narrow screens its heading, mode selector and save footer stack.

## Do's and Don'ts

### Do:

- **Do** reuse BoardUI semantic aliases, the local Inter stack and the shared Vue/Reka primitives.
- **Do** keep default controls compact and preserve their focus, invalid, disabled and reduced-motion states.
- **Do** let available content width determine the upstream row layout and keep every routing control reachable.
- **Do** pair status color with explicit text and distinguish unknown, stale, loading, failed and confirmed states.

### Don't:

- **Don't** reintroduce the discarded paper-and-ink palette, irregular outlines or replacement font family.
- **Don't** lighten the primary button gradient or muted and warning text beyond the implemented contrast adjustments.
- **Don't** substitute decorative panels for labels, values or actionable state explanations.
- **Don't** treat fixture screenshots or preview snippets as evidence of live accounts, routing or quota results.
