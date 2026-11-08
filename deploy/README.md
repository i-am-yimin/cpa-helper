# Rainyun AI Gateway deployment

Production is `https://aigateway.nigulas.top` on `154.9.254.30`.
The `ext` branch is the deployment source. `main` continues to mirror walkingddd;
neither upstream repository is a deployment target or push destination.

## Components

- CPA-Helper: host loopback `127.0.0.1:18317`, persistent SQLite data in `/opt/ai-gateway/helper`.
- CLIProxyAPI: host loopback `127.0.0.1:8317`, persistent configuration, OAuth credentials and logs under `/opt/ai-gateway`.
- nginx-ui's OpenResty node: `/` serves CPA-Helper; `/v1/` serves the model API with streaming and WebSocket support.
  Raw `/v0/management`, `/v8/management` and the bundled management page are blocked at the public proxy.
- TLS: a dedicated Let's Encrypt certificate, renewed by the existing host Certbot timer.

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

## Operations

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
