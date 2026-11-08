#!/usr/bin/env python3
"""Register this gateway with the existing CAS account center (native OIDC).

Run as root on Rainyun. Secrets stay in private files and stdin, never argv/logs.
"""
import http.cookiejar
import json
import os
from pathlib import Path
import secrets
import subprocess
import sys
from urllib.parse import urlencode
import urllib.request

ROOT = Path('/opt/ai-gateway')
DOCUMENT = ROOT / 'secrets' / 'oidc.json'
ISSUER = 'https://cas.nigulas.top/application/o/cpa-helper/'
PUBLIC = 'https://aigateway.nigulas.top'


def private_json(path, value):
    temporary = path.with_suffix('.pending')
    with temporary.open('w') as handle:
        json.dump(value, handle, indent=2)
    os.chmod(str(temporary), 0o600)
    os.replace(str(temporary), str(path))


def idp_request(path, payload=None, method='GET'):
    environment = {}
    for line in open('/etc/nigulas/apps/authentik.env'):
        if '=' in line and not line.lstrip().startswith('#'):
            key, value = line.strip().split('=', 1)
            environment[key] = value.strip().strip('"').strip("'")
    config = 'url = ' + json.dumps('https://cas.nigulas.top/api/v3/' + path) + '\n'
    config += 'header = ' + json.dumps('Authorization: Bearer ' + environment['AUTHENTIK_BOOTSTRAP_TOKEN']) + '\n'
    config += 'header = "Content-Type: application/json"\n'
    if payload is not None:
        config += 'data = ' + json.dumps(json.dumps(payload)) + '\n'
    result = subprocess.run([
        '/usr/local/bin/curl', '--resolve', 'cas.nigulas.top:443:127.0.0.1',
        '--config', '-', '--request', method, '--silent', '--show-error',
        '--max-time', '30', '--write-out', '\n%{http_code}',
    ], input=config.encode(), stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True)
    raw, code = result.stdout.rsplit(b'\n', 1)
    if int(code) >= 400:
        # A provider response can contain client secrets. Do not dump its body.
        raise RuntimeError('Identity API %s returned HTTP %s' % (path, code.decode()))
    return json.loads(raw) if raw else None


def matching(path, **filters):
    result = idp_request(path + '?' + urlencode(filters))['results']
    if len(result) > 1:
        raise RuntimeError('Ambiguous identity configuration: ' + path)
    return result[0] if result else None


def register():
    owner = matching('core/users/', username='nigulas')
    group = matching('core/groups/', name='infra-admin')
    reference = matching('providers/oauth2/', name='Nginx UI OIDC')
    if not owner or not owner['is_active'] or not group or group['pk'] not in owner['groups']:
        raise RuntimeError('The explicitly selected CAS owner is not an active infrastructure administrator')
    if not reference or not reference.get('signing_key') or len(reference['property_mappings']) != 3:
        raise RuntimeError('Existing signed OIDC reference configuration is unavailable')
    provider = matching('providers/oauth2/', name='CPA-Helper OIDC')
    client_secret = provider.get('client_secret') if provider else None
    if DOCUMENT.exists():
        previous = json.loads(DOCUMENT.read_text())
        client_secret = previous['client_secret']
        if previous['subject'] != owner['uuid']:
            raise RuntimeError('CAS owner UUID changed; refusing to reassign administrator access')
        if not provider or previous['client_id'] != provider['client_id']:
            raise RuntimeError('Stored provider identity does not match the account center')
    elif provider:
        raise RuntimeError('Provider exists without its private deployment document; inspect before adoption')
    fields = {key: reference[key] for key in (
        'authorization_flow', 'invalidation_flow', 'signing_key', 'property_mappings')}
    fields.update({
        'name': 'CPA-Helper OIDC', 'client_type': 'confidential',
        'grant_types': ['authorization_code'], 'include_claims_in_id_token': True,
        'sub_mode': 'user_uuid', 'issuer_mode': 'per_provider',
        'redirect_uris': [
            {'matching_mode': 'strict', 'url': PUBLIC + '/oidc/callback', 'redirect_uri_type': 'authorization'},
            {'matching_mode': 'strict', 'url': PUBLIC + '/oidc/logged-out', 'redirect_uri_type': 'logout'},
        ],
    })
    if provider:
        provider = idp_request('providers/oauth2/%s/' % provider['pk'], fields, 'PATCH')
    else:
        fields.update(client_id='cpa-helper-' + secrets.token_hex(16), client_secret=secrets.token_urlsafe(48))
        client_secret = fields['client_secret']
        provider = idp_request('providers/oauth2/', fields, 'POST')
    app = matching('core/applications/', slug='cpa-helper')
    if app and app.get('provider') != provider['pk']:
        raise RuntimeError('Application slug belongs to a different provider')
    if not app:
        app = idp_request('core/applications/', {
            'name': 'AI Gateway', 'slug': 'cpa-helper', 'provider': provider['pk'],
            'meta_launch_url': PUBLIC, 'meta_description': 'CPA-Helper and CLIProxyAPI',
            'meta_hide': True, 'policy_engine_mode': 'all',
        }, 'POST')
    target = app.get('pbm_uuid') or app['pk']
    bindings = idp_request('policies/bindings/?' + urlencode({'target': target}))['results']
    if any(b.get('group') != group['pk'] or b.get('negate') or not b.get('enabled') for b in bindings):
        raise RuntimeError('Unexpected existing application policy; refusing to broaden access')
    if not bindings:
        idp_request('policies/bindings/', {
            'target': target, 'group': group['pk'], 'enabled': True,
            'negate': False, 'order': 0, 'failure_result': False,
        }, 'POST')
    idp_request('core/applications/cpa-helper/', {'meta_hide': False}, 'PATCH')
    private_json(DOCUMENT, {
        'issuer': ISSUER, 'subject': owner['uuid'], 'identity_username': 'nigulas',
        'local_username': 'i-am-yimin', 'client_id': provider['client_id'],
        'client_secret': provider.get('client_secret') or client_secret,
        'provider_id': provider['pk'], 'application_slug': 'cpa-helper',
    })
    print('CAS application registered; strict callbacks and infra-admin access restriction configured.')


def enable():
    config = json.loads(DOCUMENT.read_text())
    bootstrap = json.loads((ROOT / 'secrets' / 'bootstrap.json').read_text())
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def panel(method, path, payload=None):
        data = None if payload is None else json.dumps(payload).encode()
        request = urllib.request.Request('http://127.0.0.1:18317/api' + path, data=data,
                                         method=method, headers={'Content-Type': 'application/json'})
        with opener.open(request, timeout=20) as response:
            return json.load(response)

    panel('POST', '/auth/login', {key: bootstrap[key] for key in ('username', 'password')})
    previous = panel('GET', '/settings')
    backup = ROOT / 'secrets' / 'settings-before-sso.json'
    if not backup.exists():
        private_json(backup, previous)
    binder = sys.argv[2] if len(sys.argv) > 2 else '/var/tmp/cpa-helper-bind-oidc'
    subprocess.run([binder, '-config', str(DOCUMENT)], check=True)
    result = panel('PUT', '/settings', {
        'oidc_enabled': True, 'oidc_issuer': ISSUER,
        'oidc_client_id': config['client_id'], 'oidc_client_secret': config['client_secret'],
        'oidc_public_url': PUBLIC, 'oidc_username_claim': 'preferred_username',
        'oidc_auto_create_users': True, 'sso_default_login': 'oidc',
    })
    if not result.get('oidc_enabled') or result.get('sso_default_login') != 'oidc':
        raise RuntimeError('Default CAS login was not persisted')
    print('Default CAS login enabled; existing administrator, API keys and usage retained.')


if __name__ == '__main__':
    if os.geteuid() != 0:
        raise SystemExit('Run as root')
    os.umask(0o077)
    {'register': register, 'enable': enable}[sys.argv[1]]()
