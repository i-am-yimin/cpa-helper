#!/usr/bin/env python3
"""Initialize this installation without writing credentials to CI output."""
import http.cookiejar
import json
import os
from pathlib import Path
import secrets
import sys
import urllib.request

from group_scheduler import prepare_scheduler, configure_scheduler, check_scheduler, probe_scheduler

ROOT = Path('/opt/ai-gateway')
SECRET = ROOT / 'secrets' / 'bootstrap.json'
BASE = 'http://127.0.0.1:18317/api'


def request(opener, method, path, payload=None):
    data = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(BASE + path, data=data, method=method,
                                 headers={'Content-Type': 'application/json'})
    with opener.open(req, timeout=30) as response:
        return json.load(response)


def prepare():
    os.umask(0o077)
    for name in ('core', 'auths', 'logs', 'helper', 'secrets'):
        (ROOT / name).mkdir(parents=True, exist_ok=True)
    if not SECRET.exists():
        value = {'username': 'i-am-yimin', 'password': secrets.token_urlsafe(32),
                 'management_key': secrets.token_urlsafe(48),
                 'probe_api_key': 'sk-' + secrets.token_urlsafe(36)}
        with SECRET.open('x') as handle:
            json.dump(value, handle, indent=2)
        os.chmod(str(SECRET), 0o600)
    value = json.loads(SECRET.read_text())
    config = ROOT / 'core' / 'config.yaml'
    if not config.exists():
        # JSON-quoted secrets are valid YAML scalars. No YAML dependency is needed.
        content = '''config-version: 8
server:
  host: "0.0.0.0"
  port: 8317
management:
  allow-remote: true
  secret-key: %s
  disable-control-panel: true
access:
  api-keys: [%s]
oauth:
  auth-dir: "/root/.cli-proxy-api"
observability:
  logs:
    logging-to-file: false
    request-log: false
    error-logs-max-files: 5
  usage:
    usage-statistics-enabled: true
    redis-usage-queue-retention-seconds: 3600
plugins:
  enabled: false
''' % (json.dumps(value['management_key']), json.dumps(value['probe_api_key']))
        with config.open('x') as handle:
            handle.write(content)
        os.chmod(str(config), 0o600)
    prepare_scheduler(ROOT, Path(__file__).resolve().parent, value['management_key'])
    print('Persistent directories and private configuration are ready.')


def configure():
    value = json.loads(SECRET.read_text())
    configure_scheduler(value['management_key'], Path(__file__).resolve().parent)
    marker = ROOT / 'secrets' / 'configured'
    if marker.exists():
        print('Existing administrator and settings retained.')
        return
    value = json.loads(SECRET.read_text())
    opener = urllib.request.build_opener(
        urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
    state = request(opener, 'GET', '/auth/setup')
    credentials = {k: value[k] for k in ('username', 'password')}
    if state['setup_required']:
        request(opener, 'POST', '/auth/setup', dict(credentials, nickname='i-am-yimin'))
    else:
        request(opener, 'POST', '/auth/login', credentials)
    request(opener, 'PUT', '/settings', {
        'cliaproxy_url': 'http://cli-proxy-api:8317',
        'management_key': value['management_key'],
        'model_request_url': 'https://aigateway.nigulas.top/v1',
        'collector_enabled': True,
    })
    settings = request(opener, 'GET', '/settings')
    if not settings['management_key_set'] or not settings['collector_enabled']:
        raise RuntimeError('Panel integration was not persisted')
    if not value.get('helper_api_key'):
        key = request(opener, 'POST', '/api-keys', {'description': 'Initial API key'})
        value['helper_api_key'] = key['api_key']
        value['probe_api_key'] = key['api_key']
        temporary = SECRET.with_suffix('.tmp')
        with temporary.open('w') as handle:
            json.dump(value, handle, indent=2)
        os.chmod(str(temporary), 0o600)
        os.replace(str(temporary), str(SECRET))
    marker.touch(mode=0o600)
    print('Administrator initialized and CPA-Helper connected to CLIProxyAPI.')


def check():
    value = json.loads(SECRET.read_text())
    check_scheduler(value['management_key'])
    req = urllib.request.Request('http://127.0.0.1:8317/v1/models', headers={
        'Authorization': 'Bearer ' + value['probe_api_key']})
    with urllib.request.urlopen(req, timeout=20) as response:
        models = json.load(response)
    req = urllib.request.Request('http://127.0.0.1:8317/v0/management/api-keys',
                                 headers={'Authorization': 'Bearer ' + value['management_key']})
    with urllib.request.urlopen(req, timeout=20) as response:
        json.load(response)
    print('Authenticated model endpoint and internal management endpoint passed; model count: %d'
          % len(models.get('data', [])))
    probe_scheduler(ROOT, value['probe_api_key'], models)


if __name__ == '__main__':
    {'prepare': prepare, 'configure': configure, 'check': check}[sys.argv[1]]()
