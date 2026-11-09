# Rainyun AI Gateway deployment

Production is `https://aigateway.nigulas.top` on `154.9.254.30`.
The `ext` branch is the deployment source. `main` continues to mirror walkingddd;
neither upstream repository is a deployment target or push destination.

## Components

- CPA-Helper: host loopback `127.0.0.1:18317`, persistent SQLite data in `/opt/ai-gateway/helper`.
- CLIProxyAPI: host loopback `127.0.0.1:8317`, persistent configuration, OAuth credentials and logs under `/opt/ai-gateway`.
- nginx-ui's OpenResty node: `/` serves CPA-Helper; `/v1/` serves the model API with streaming and WebSocket support.
  Raw `/v0/management`, `/v8/management` and the bundled management page are blocked at the public proxy.
- TLS: a dedicated Let's Encrypt certificate, renewed by `cpa-helper-certbot-renew.timer`.

The API base URL is `https://aigateway.nigulas.top/v1`.
Upstream provider credentials are not bundled or copied from other services.
Add them through the administrator's upstream/credential management pages before sending model requests.

## CI/CD

`Deploy AI Gateway` runs after relevant pushes to `ext`, or manually on that branch:

1. GitHub-hosted Linux runs frontend checks and Go regression tests.
2. BuildKit builds a Linux amd64 image and pushes an immutable SHA tag to this repository's GHCR package.
3. A dedicated repository-scoped runner (`cpa-helper-deploy`) invokes only the installed root-owned deployment helper.
4. The helper checks that the commit still equals `origin/ext`, validates the image digest/revision, and pulls both images before stopping any service.
5. On upgrades, it stops this Compose project and snapshots its persistent state before migrations.
6. It starts the stack, checks readiness, management authentication and model API authentication, and advances `/opt/ai-gateway/current` only on success.
   Failed upgrades restore the previous state and images; failed first installations are stopped for inspection.

Registry credentials exist only for the duration of deployment. The runner is not a Docker-group member;
its sudo rule allows only `/usr/local/sbin/cpa-helper-deploy`, whose image/repository arguments are constrained.
No pull-request workflow targets this production runner.

The production host uses CentOS 7. `install-runner` reuses the already installed compatible public
runner 2.338.0 runtime and its private compatibility library without copying any existing runner credentials.
Its small service supervisor uses the existing, checksum-verified CentOS-compatible Node 16 binary;
application builds and JavaScript actions execute only on the current GitHub-hosted Linux runner.
Automatic runner updates are disabled; update this isolated runtime deliberately when required by GitHub.

One-time installation as root, with a short-lived repository registration token on stdin:

```sh
./deploy/install-runner /path/to/source
```

## Private bootstrap

The first deployment initializes administrator `i-am-yimin`, a randomly generated password,
a management key, and an initial account-owned API key. The panel is connected to
`http://cli-proxy-api:8317` and usage collection is enabled automatically.
Credentials are stored only in `/opt/ai-gateway/secrets/bootstrap.json` (root-only, mode 0600).
They are never printed in CI logs or committed to Git. The password can be changed in the panel;
later deployments preserve the current administrator and settings.

## CAS account-center integration

Normal browser sign-in uses the existing `cas.nigulas.top` authentik service
through native OIDC authorization-code + PKCE. Normal `/login` requests enter
OIDC. Explicit logout revokes the gateway session, calls the identity provider's
end-session endpoint when available, then returns to CAS authorization with
`prompt=login`. Users remain at CAS until they interactively sign in again.
A secure, HTTP-only reauthentication marker also prevents returning to the
ordinary gateway entry from silently reusing an old provider session. It clears
only after successful interactive OIDC sign-in. The explicit recovery URL
`/login?skipsso=true` remains available; logout no longer lands on that local form.
The public password-login API stays blocked. Model API requests continue using API keys.

The `CPA-Helper OIDC` provider has strict authorization/logout callbacks and its
`AI Gateway` application is restricted to `infra-admin`. The explicitly selected
CAS account `nigulas` is bound by verified immutable user UUID to the existing
local administrator `i-am-yimin`; existing local roles, API keys and history are retained.
Other authorized CAS identities provision ordinary accounts, never administrators.

The root-only `/opt/ai-gateway/secrets/oidc.json` stores the application credentials
and verified identity. Registration is reproducible with
`python3 deploy/configure-sso.py register` on the production host. It uses the
existing root-owned identity administration token and does not alter other applications.

CPA's provider explicitly uses `default-invalidation-flow` with its user logout
stage. The provider-only flow inherited from Nginx UI retains the CAS session
and can silently sign the user back in even with `prompt=login`. Every deployment
reconciles only CPA's `invalidation_flow` and verifies the user logout stage;
shared flows and other providers are not edited. `configure-sso.py reconcile-logout`
and `check-logout` expose the same bounded reconciliation and health check.

For initial binding, build `deploy/bind-oidc.go` from `backend` using the existing
Go module, then run `python3 deploy/configure-sso.py enable /path/to/bind-oidc` as root.
The binder uses the same modern SQLite driver as the application, takes a private
consistent database snapshot, and refuses disabled/non-admin targets, conflicting
identity ownership or replacement of a different existing identity.
The identity-binding regression tests run in production CI.

Emergency password recovery requires an SSH tunnel to `127.0.0.1:18317` and
`/login?skipsso=true`; password sign-in remains unavailable through the public reverse proxy.

The matching site in the nginx-ui operations repository is applied by its
existing infrastructure workflow, including backup, configuration validation,
reload and rollback. CI runs this versioned site against a real nginx process, including
logout refresh, ordinary OIDC redirects, the blocked password API and hidden
management APIs.

## Operations

### Antigravity quota

The credential page queries Antigravity's `retrieveUserQuotaSummary` through
CLIProxyAPI's authenticated server-side `api-call`. It shows the groups and quota
windows returned by the provider, including remaining percentages and reset times.
OAuth tokens stay in CLIProxyAPI; project lookup and credential downloads remain
server-side. Quota queries never change credential status or its global priority.
They publish per-model-group routing observations to the native scheduler.

The immutable Helper image includes a Linux glibc scheduler built for the Debian
Core image. Before downtime, deployment verifies the Core libc and extracts this
release's plugin into its own `deploy/plugins` directory. Compose mounts that
directory read-only; only `helper/scheduler` is shared for quota snapshots and
routing heartbeats, not the user database or OAuth files. The snapshot directory
is covered by the existing Helper-state backup and the old release keeps its own
plugin mount for rollback.

Post-start bootstrap enables only `cpa-group-scheduler` using the V8 strong ETag
and verifies that Core reports it registered and effective. Conflicting enabled
plugins, unsupported routing strategies or failed registration abort deployment
and invoke the existing rollback. The original Codex policy remains in place.
Background refresh is explicitly enabled by the configured snapshot path and
continues without an open browser. The deployment health check makes at most one
one-token model request when matching healthy groups exist and reports whether
the native routing heartbeat advanced; provider quota errors are distinguished
from plugin load failures.

Successful queries are cached for five minutes or until the next reset, whichever
comes first. Concurrent queries share a request. Failures back off for one minute
and retain the last successful result with a stale marker. Administrators can
refresh a single credential or a mixed Codex/Antigravity selection; account-status
viewers can read cached quotas but cannot force a refresh. The UI regression runs
in production CI with simulated quota responses, without production credentials.

```sh
cd /opt/ai-gateway/current
docker compose --project-name ai-gateway --env-file .env -f deploy/compose.yml ps
docker compose --project-name ai-gateway --env-file .env -f deploy/compose.yml logs --tail 100
curl -fsS http://127.0.0.1:18317/api/ready
```

Backups and failed-upgrade state are private under `/opt/ai-gateway/backups`.
Keep a separate off-host copy before major upgrades. Review backup retention and available disk regularly.
Container logs are bounded at three 5 MB files per service. Never remove persistent directories or Docker volumes during cache cleanup.
Core upgrades require an explicit edit of `core-image.txt`; this pins the audited image digest rather than following `latest` on every restart.

The OpenResty site is mirrored into the user's nginx-ui operations repository so infrastructure
reapplication retains it. `renew-certificate` is installed as a dedicated Certbot deploy hook;
it tests and reloads OpenResty only when this domain's certificate is renewed.
