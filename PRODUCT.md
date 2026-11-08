# CPA-Helper

<!-- impeccable:product-schema 1 -->

## Platform
web

## Stack
Vue 3, TypeScript, Reka UI, Tailwind 4; Go and SQLite / PostgreSQL.

## Users and purpose
Administrators manage CLIProxyAPI upstreams, OAuth credentials, quotas and scheduling priorities. Account users inspect their own API keys, usage and balances.

## Confirmed requirements
- One upstream management entry for API Key upstreams and OAuth credentials, with directly editable priorities.
- Antigravity participates in quota-aware scheduling, including its GPT / Claude model quota groups.
- Show actual quota windows and locally collected request, token and cost statistics. Unknown data must remain unknown.
- Rebuild the platform visual style using Looseleaf's paper and ink design language, retaining Vue and existing functions.

## Constraints
CPA-Helper manages CPA and collects usage; CPA performs model routing. Higher priority is preferred; matching models and provider capabilities still constrain routing. Existing admin and account access controls remain authoritative.
