# Workspace glass redesign

Mode: Operate. User direction, November 10, 2026: the previous flat redesign did not convey Apple frosted glass; background and lighting must be designed. No additional reduced-motion work requested.

## Direction contract

THESIS: A gateway workspace viewed through layered frosted glass, lit from the upper left. Background transmission, illuminated edges and contact shadows make depth visible while operational data stays clear.

OWN-WORLD: Silver-blue daylight with a restrained warm reflection; midnight blue with teal reflected light in dark mode. Four material roles: navigation, panels, controls and floating UI. Fine bright inner rims and dark outer edges replace flat white sheets.

STORY: Users locate usage, inspect routing and quota state, then act through familiar controls. All permissions, labels, server states and save behavior remain authoritative.

FIRST VIEWPORT: A 246px navigation plane, a 64px translucent header over the scrolling canvas, page title and actions, then frosted filters, statistics and charts. Mobile keeps a compact header and sheet navigation. A centered login pane samples directional architectural light bands.

FORM: User-pinned Apple-inspired frosted glass; Gemini 4 Argon High directs and reviews, GPT 6.1 Sol high implements. seed: none (user-pinned revision).

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## Explicit brief exceptions

Backdrop blur, atmospheric background gradients, optical control gradients and layered shadows are intentional user requirements. Generic detector bans against these effects must not erase the selected material. Avoid decorative spheres, purple blobs, nested blur, moving wallpapers and illegible text. All browser review data is synthetic; no raster wallpaper is required or shipped.

## Finish review — November 10, 2026

Gemini 4 Argon High returned **ship** after inspecting final light/dark and desktop/tablet/mobile renders. The initial inner authentication patch, settings fieldset/legend patches and opaque ranking rows were corrected before acceptance. Actual screenshots live in `.taste-skill/glass-final/`; the design vocabulary is documented in root DESIGN.md and `.impeccable/design.json`.

Validation: 96 synthetic route/theme/viewport captures, no browser exceptions or outer horizontal overflow; 20 existing mocked/static Playwright cases and 6 interaction checks passed. Production build, ESLint and i18n smoke checks passed. Additional material checks confirmed transparent canvas ancestors, active blur on outer panels/header, content overlapping the scrolling header, and reachable authentication controls at 390×440 in both themes. Prefix-only Safari CSS compatibility was added; Safari itself was not executed.

The user-selected background/material treatment intentionally overrides design-taste gradient, glass, heavy-shadow and button-fx defaults. Automated inspection of usage, upstreams and login found no remaining FAIL with those allowances. Supporting 11–12px metadata, centered login whitespace and the mobile login background color share retain explicit rationale in `.taste-skill/REVIEW.md`.
