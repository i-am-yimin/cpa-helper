"""Stage and enable the native group scheduler without logging credentials."""
import json
import fnmatch
import os
from pathlib import Path
import re
import subprocess
import time
import urllib.error
import urllib.request


PLUGIN = 'cpa-group-scheduler'
CORE = 'http://127.0.0.1:8317/v8/management'


def core_request(key, method, path, payload=None, revision=None):
    headers = {'Authorization': 'Bearer ' + key, 'Content-Type': 'application/json'}
    if revision:
        headers['If-Match'] = revision
    req = urllib.request.Request(CORE + path, method=method, headers=headers,
                                 data=None if payload is None else json.dumps(payload).encode())
    with urllib.request.urlopen(req, timeout=20) as response:
        raw = response.read()
        return (json.loads(raw) if raw else None), response.headers.get('ETag', '')


def routing_strategy(key):
    try:
        routing, _ = core_request(key, 'GET', '/config/routing')
    except urllib.error.HTTPError as error:
        if error.code == 404:
            return 'round-robin'  # An omitted routing subtree uses Core's default.
        raise RuntimeError('Cannot inspect Core routing strategy (HTTP %d)' % error.code)
    strategy = routing.get('strategy', 'round-robin')
    if strategy not in ('round-robin', 'fill-first'):
        raise RuntimeError('The group scheduler does not support the current routing strategy')
    return strategy


def prepare_scheduler(root, deployment, management_key):
    """Called before services stop. Only stage release artifacts and directories."""
    root, deployment = Path(root), Path(deployment)
    env = dict(line.split('=', 1) for line in (deployment.parent / '.env').read_text().splitlines()
               if '=' in line and not line.startswith('#'))
    image = env.get('CPA_HELPER_IMAGE', '')
    core_image = env.get('CLI_PROXY_IMAGE', '')
    if not re.fullmatch(r'ghcr\.io/i-am-yimin/cpa-helper:sha-[0-9a-f]{40}@sha256:[0-9a-f]{64}', image):
        raise RuntimeError('Refusing an unverified scheduler image')
    if not re.fullmatch(r'eceasy/cli-proxy-api@sha256:[0-9a-f]{64}', core_image):
        raise RuntimeError('Refusing an unpinned Core image')
    libc = subprocess.check_output([
        'docker', 'run', '--rm', '--network', 'none', '--read-only', '--cap-drop', 'ALL',
        '--security-opt', 'no-new-privileges:true', '--entrypoint', '/bin/sh', core_image,
        '-c', 'ldd --version 2>&1'], stderr=subprocess.STDOUT).decode(errors='replace')
    if not any(marker in libc for marker in ('GLIBC', 'GNU libc', 'GNU C Library')):
        raise RuntimeError('Core libc does not match the Debian-built scheduler')
    strategy = routing_strategy(management_key) if (root / 'secrets' / 'configured').exists() else 'round-robin'
    plugin_dir = deployment / 'plugins'
    plugin_dir.mkdir(mode=0o755, exist_ok=True)
    (root / 'helper' / 'scheduler').mkdir(mode=0o700, parents=True, exist_ok=True)
    container = subprocess.check_output(['docker', 'create', '--entrypoint', '/bin/true', image], universal_newlines=True).strip()
    staged = plugin_dir / (PLUGIN + '.so.staged')
    try:
        subprocess.run(['docker', 'cp', container + ':/app/plugins/' + PLUGIN + '.so', str(staged)], check=True)
        with staged.open('rb') as handle:
            if handle.read(4) != b'\x7fELF':
                raise RuntimeError('Scheduler artifact is not an ELF library')
        staged.chmod(0o644)
        os.replace(str(staged), str(plugin_dir / (PLUGIN + '.so')))
        subprocess.run(['docker', 'cp', container + ':/app/plugins/LICENSE-CPA-Core.txt',
                        str(plugin_dir / 'LICENSE-CPA-Core.txt')], check=True)
    finally:
        subprocess.run(['docker', 'rm', container], check=True, stdout=subprocess.DEVNULL)
    (deployment / 'group-scheduler.env').write_text('CPA_GROUP_SCHEDULER_STRATEGY=' + strategy + '\n')
    print('Native scheduler staged for this immutable release; Core libc: glibc; strategy: ' + strategy)


def configure_scheduler(key, deployment):
    settings, revision = core_request(key, 'GET', '/config/plugins')
    if not revision or revision.startswith('W/'):
        raise RuntimeError('Core did not return a strong configuration revision')
    settings = settings or {}
    configs = settings.get('configs') or {}
    if any(name != PLUGIN and config.get('enabled', True) for name, config in configs.items()):
        raise RuntimeError('Refusing to activate or replace another configured plugin implicitly')
    directory = settings.get('dir') or 'plugins'
    if directory.rstrip('/') not in ('plugins', './plugins', '/CLIProxyAPI/plugins'):
        raise RuntimeError('Core plugin directory does not match the release mount')
    strategy = routing_strategy(key)
    staged_strategy = (Path(deployment) / 'group-scheduler.env').read_text().strip().split('=', 1)[1]
    if strategy != staged_strategy:
        raise RuntimeError('Core routing changed during deployment; refusing a stale plugin strategy')
    current = configs.get(PLUGIN) or {}
    if not settings.get('enabled') or not current.get('enabled'):
        patch = {'plugins': {'enabled': True, 'dir': directory, 'configs': {
            PLUGIN: {'enabled': True, 'priority': current.get('priority', 1)}}}}
        core_request(key, 'PATCH', '/config', patch, revision=revision)
    for attempt in range(30):
        try:
            check_scheduler(key)
            return
        except RuntimeError:
            if attempt == 29:
                raise
            time.sleep(1)


def check_scheduler(key):
    inventory, _ = core_request(key, 'GET', '/plugins')
    match = next((item for item in inventory.get('plugins', []) if item.get('id') == PLUGIN), None)
    if not match or not match.get('registered') or not match.get('effective_enabled'):
        raise RuntimeError('Native group scheduler did not register and become effective in Core')
    if (match.get('metadata') or {}).get('version') != '1.0.0':
        raise RuntimeError('Unexpected group scheduler version in Core')
    print('Core verified native group scheduler: registered, effective, version 1.0.0')


def probe_scheduler(root, api_key, models):
    """At most one tiny real request; quota/provider failures do not undo a healthy deployment."""
    snapshot = Path(root) / 'helper' / 'scheduler' / 'group-scheduler.json'
    rules = []
    for attempt in range(45):
        try:
            rules = json.loads(snapshot.read_text()).get('rules', [])
        except (OSError, ValueError):
            pass
        if rules:
            break
        time.sleep(1)
    candidates = [item['id'] for item in models.get('data', []) if item.get('id') and any(
        rule.get('effective_priority', -1) >= 0 and any(fnmatch.fnmatchcase(item['id'].lower(), pattern.lower())
        for pattern in rule.get('models', [])) for rule in rules)]
    if not candidates:
        print('Native scheduler is loaded; no available matching quota group for a traffic probe.')
        return
    candidates.sort(key=lambda model: ('flash' not in model.lower(), model))
    started = time.time()
    req = urllib.request.Request('http://127.0.0.1:8317/v1/chat/completions', method='POST',
        headers={'Authorization': 'Bearer ' + api_key, 'Content-Type': 'application/json'},
        data=json.dumps({'model': candidates[0], 'messages': [{'role': 'user', 'content': 'Reply OK.'}],
                         'max_tokens': 1, 'stream': False}).encode())
    status = None
    try:
        with urllib.request.urlopen(req, timeout=35) as response:
            response.read()
            status = response.status
    except urllib.error.HTTPError as error:
        status = error.code
    except (urllib.error.URLError, OSError):
        pass
    heartbeat = Path(str(snapshot) + '.status')
    engaged = heartbeat.exists() and heartbeat.stat().st_mtime >= started - 1
    report = {'checked_at_unix': int(time.time()), 'native_registered': True,
              'traffic_engaged': engaged, 'probe_http_status': status}
    (snapshot.parent / 'deployment-check.json').write_text(json.dumps(report))
    print('Group scheduler traffic probe: HTTP %s; fresh native routing heartbeat: %s'
          % (status if status is not None else 'unavailable', 'yes' if engaged else 'not observed'))
