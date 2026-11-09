---
name: CPA-Helper — Apple-inspired workspace
description: A frosted glass gateway workspace with directional light and responsive interaction.
colors:
  primary: "#0066df"
  primary-hover: "#0058c5"
  primary-foreground: "#ffffff"
  background: "#dce5f2"
  card: "#ffffff"
  foreground: "#1d2433"
  muted: "#eef1f5"
  muted-foreground: "#626c7d"
  border: "#e5e9f0"
  input: "#d3d9e3"
  accent: "#e5efff"
  accent-foreground: "#0058c5"
  destructive: "#cf3448"
  success: "#21815b"
  warning: "#96610e"
  background-dark: "#080c14"
  card-dark: "#1b1e25"
  foreground-dark: "#f5f7fb"
  muted-foreground-dark: "#a2adbd"
  border-dark: "#303640"
  input-dark: "#434c5a"
  primary-dark: "#409cff"
  accent-dark: "#203853"
  accent-foreground-dark: "#81bdff"
  destructive-dark: "#ff7a87"
  success-dark: "#66c69d"
  warning-dark: "#f2bc67"
  chart-1: "#0066df"
  chart-2: "#3695c5"
  chart-3: "#91b6d1"
  chart-4: "#5a79a8"
  chart-5: "#b2bfce"
typography:
  headline:
    fontFamily: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Inter Variable", "PingFang SC", "Microsoft YaHei", sans-serif
    fontSize: 30px
    fontWeight: 700
    lineHeight: 1.22
    letterSpacing: -0.035em
  headline-mobile:
    fontSize: 24px
    fontWeight: 700
    lineHeight: 1.22
  title:
    fontSize: 18px
    fontWeight: 650
    letterSpacing: -0.025em
  body:
    fontSize: 14px
    fontWeight: 400
    lineHeight: 1.55
  control:
    fontSize: 14px
    fontWeight: 550
  label:
    fontSize: 12px
    fontWeight: 550
  mono:
    fontFamily: '"Cascadia Mono", "SFMono-Regular", Consolas, monospace'
rounded:
  badge: 7px
  small-control: 9px
  control: 12px
  floating: 16px
  card-mobile: 20px
  card: 24px
  dialog: 26px
  auth: 30px
spacing:
  xs: 4px
  sm: 8px
  md: 12px
  lg: 16px
  panel-mobile: 18px
  xl: 24px
  canvas-inline: 32px
components:
  button-primary:
    textColor: "{colors.primary-foreground}"
    typography: "{typography.control}"
    rounded: "{rounded.control}"
    padding: 0 16px
    height: 40px
  button-outline:
    textColor: "{colors.foreground}"
    typography: "{typography.control}"
    rounded: "{rounded.control}"
    padding: 0 16px
    height: 40px
  input:
    textColor: "{colors.foreground}"
    rounded: "{rounded.control}"
    padding: 0 12px
    height: 40px
  card:
    textColor: "{colors.foreground}"
    rounded: "{rounded.card}"
    padding: 24px
  tab-active:
    textColor: "{colors.foreground}"
    rounded: "{rounded.small-control}"
  badge:
    textColor: "{colors.foreground}"
    typography: "{typography.label}"
    rounded: "{rounded.badge}"
---

# Design System: CPA-Helper

## Overview

**Creative North Star: "Light through glass"**

The full-site Apple-inspired workspace uses a continuous silver-blue or midnight-blue light field behind frosted navigation, panels, controls and floating UI. Upper-left daylight, a lower-right cool reflection and a restrained warm reflection establish one physical lighting scene. Background color must remain visibly transmitted through the material.

This replaces the former flat white-card presentation. `frontend/src/styles/tokens.css` owns semantic colors and optical recipes; `frontend/src/styles/apple.css` owns shared components. Vue 3 and Reka UI preserve product behavior and keyboard interaction. Historical BoardUI sources and MIT notices remain as provenance assets at `frontend/src/styles/boardui/LICENSE` and `frontend/public/licenses/boardui.txt`.

**Key Characteristics:**

- Directional ambient light with clear light and dark material hierarchies.
- Frosted panels with bright inner rims, quiet outer edges and contact shadows.
- Blue actions and data, continuous selection motion and tactile feedback.
- Dense, readable information and usable mobile controls.

## Colors

Blue identifies actions, focus and selection. The light canvas combines silver, pale blue and cyan with a restrained warm reflection; the dark canvas uses midnight blue and low-luminance blue-teal reflected light. These broad fields provide something for glass to transmit without drawing attention away from data.

Success green, warning amber and destructive red communicate real state alongside wording. Related chart blues and semantic error red remain distinct. The original brand mark is preserved. Primary button gradients stay dark enough for white labels.

**The Semantic Theme Rule.** Use semantic variables and existing `--cpa-*` aliases. Opaque `--card` remains a fallback for sticky headers and occluding inner elements; shared outer panels use the separate glass material. Full gradients, backdrop filters and shadows are recorded in the sidecar because they exceed the frontmatter component schema.

## Typography

Use the system font stack throughout. Apple platforms can use installed system fonts; local Inter Variable and Chinese system fonts supply cross-platform fallbacks. No SF font files are bundled or downloaded.

Page headings use the headline role; compact page headings use headline-mobile. Cards use semibold titles; charts and compact sections use 15–16px headings. Operational copy uses body/control roles. Supporting metadata, axes and badges use 11–12px; primary data stays larger. Tables, percentages and changing values use tabular numerals. Reserve monospace for code and identifiers.

## Layout

The shell fills the viewport with a 64px overlapping glass header, a 246px desktop sidebar and independently scrolling content. Content uses a 64px header allowance plus 28px 32px 36px insets and a 1920px maximum. At widths up to 768px it uses a 64px header allowance plus 20px 16px 24px insets and sheet navigation. Page gaps reduce from 24px to 18px.

The upstream hub responds to available content width. Below 960px its table becomes structured cards. From 600px to 959px identity and switch occupy the first row, followed by provider, quota and priority columns. Below 600px those details stack. Expanded content, policy controls and actions remain reachable without page-wide horizontal scrolling.

The usage dashboard groups six values in one statistics strip: six desktop columns, three below 980px and two on mobile. Charts and supporting panels reflow below it. Long tables may retain their own horizontal scroll area.

Authentication uses one centered form with a separate brand mark. Mobile places the form near the top with safe-area spacing. Keep the screen scrollable on short viewports.

## Elevation & Depth

The fixed viewport background supplies the lighting scene. The app shell, main area and scrolling canvas remain transparent so they do not interrupt transmission. The translucent 64px header overlaps scrolling content, revealing its blur as content moves underneath.

Four optical roles define depth: navigation chrome, frosted panels, thin controls and floating dialogs or menus. Each uses a directional translucent gradient, an illuminated inner top edge, a subtle outer edge and appropriate ambient/contact shadows. Light and dark modes have separately tuned opacity and reflection. Login adds static diagonal light bands behind its frosted pane.

**The Transmission Rule.** Preserve a continuous path from the viewport light field to outer glass surfaces. Apply backdrop blur to outer planes and floating UI; nested controls and rows inherit their parent material and use thin translucent fills without repeated blur. Keep occluding elements solid where content needs masking. Avoid extra blur on text, animated wallpapers and decorative spheres.

Motion explains state: sidebar selection slides between destinations, Reka measures the tab indicator, buttons press to 0.98, switches slide, dialogs scale/fade, routes move a few pixels and quota/ranking bars transition. No additional reduced-motion adaptation was requested.

## Shapes

Use regular rounded rectangles with role-specific corners from the frontmatter. Mobile cards use the smaller card radius. Tabs have an inset translucent track and a raised glass selection surface. Switches and progress tracks remain fully rounded. Preserve the original logo inside its frame.

## Components

### Buttons and fields

Primary buttons use an optical blue gradient, white labels, a fine upper highlight and brief press feedback. Outline controls and fields use thin glass; ghost controls reveal a hover surface, and destructive actions use semantic tint. Preserve shared APIs, disabled handling, visible focus and error feedback.

Fields, input groups and select triggers share a 40px minimum height and 12px corners. Mobile inputs use at least 44px and 16px text. Native upstream filters retain keyboard semantics with thin custom chevrons. Icon actions require accessible names.

### Navigation and tabs

The sidebar has a moving blue selection beneath the active item, with compact icon and mobile sheet modes. Tabs preserve Reka keyboard behavior and its measured indicator so variable-width and translated labels stay aligned. Long tab strips scroll within their container.

### Cards, charts and data states

Reuse the card primitive. The usage statistics strip uses a semantic description list. Chart legends retain labels, values and percentages. Ranking names can wrap accompanying metrics onto another line. Unknown, loading, empty, stale, failed and pending versus confirmed states keep explicit wording.

### Quota policy controls

Keep quota bars and priority controls separate in responsive layouts. Editors retain labelled toggles, mode selection, draft/save actions and synchronization wording. Routing rules, inheritance and manual resume remain defined in PRODUCT.md.

## Do's and Don'ts

- **Do** reuse semantic themes and shared Vue/Reka primitives.
- **Do** preserve visible background transmission and distinguish the four material roles.
- **Do** let available content width determine upstream layout.
- **Do** pair status color with explicit state wording.
- **Don't** add competing theme mappings, nested blur or unrelated glowing blobs.
- **Don't** use color or animation to imply routing state that the API has not confirmed.
- **Don't** treat fixture screenshots or component previews as live account evidence.
