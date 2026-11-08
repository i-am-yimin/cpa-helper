# CPA-Helper

<!-- impeccable:product-schema 1 -->

## Platform
web

## Stack
Vue 3, TypeScript, Reka UI, Tailwind 4; Go and SQLite / PostgreSQL.

## Users and purpose
Administrators manage CLIProxyAPI upstream connections, OAuth credentials, quotas and scheduling priorities. Account users inspect their own API keys, usage and balances.

## Confirmed requirements

- One upstream management entry combines API Key upstreams and OAuth credentials, real counts, filters and directly editable priorities. Connection details, quota state and model-group priority form a readable working sequence.
- Use the explicitly requested BoardUI visual system: neutral surfaces, local Inter, blue interactions, regular corners and contact shadows. Adapt the pinned, free MIT source to the existing Vue 3 / Reka UI / Tailwind 4 application and preserve existing behavior and permissions.
- Antigravity model quota groups keep independent priorities and quota-aware routing, including Gemini and Claude / GPT groups. An account-level priority does not replace a group's own priority.
- Show actual quota windows and locally collected request, token and cost statistics. Unknown data remains unknown.
- Plugin loaded/idle status comes from the Core plugin registry. Heartbeat and recent activity are separate observations; lack of a recent heartbeat must not be presented as proof that a plugin is not loaded.
- The upstream hub adapts to available content width so controls remain visible on desktop, tablet and mobile.

## Codex low-quota guard

- A configurable guard checks the 5-hour and weekly remaining-quota windows. If either enabled window is at or below its configured threshold, disable that credential's routing and persist priority `-1`. The guard runs every 30 seconds and applies independently of inspection dry-run mode.
- Global rules can be enabled with explicitly configured thresholds or switched off. Each credential can inherit global rules, use custom rules or turn its rule off. No threshold percentage was chosen by the user; the new global rule is off on upgrade. A displayed editor value is not evidence of a saved active threshold.
- A quota pause persists through quota recovery, quota reset and service restart. Disabling a threshold does not automatically resume routing.
- While the effective rule still triggers, manually raising priority cannot bypass it. Once the rule is safe or off, the administrator must explicitly choose a priority, including `0`, to resume.
- Separate manual disable remains authoritative: clear manual disable first, then explicitly choose a priority to clear a quota pause. The UI distinguishes a requested pause, confirmed CPA disable and synchronization errors.

## Constraints

CPA-Helper manages CPA and collects usage; CPA performs model routing. Higher priority is preferred; matching models and provider capabilities still constrain routing. Existing admin and account access controls remain authoritative. Historical fixture screenshots and component previews are not live account or routing evidence.
