import os
from pathlib import Path
import re
import socket
import subprocess
import tempfile
import threading
import time
import unittest
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
TEMPLATE = Path(__file__).parent / 'nginx' / 'aigateway.nigulas.top.conf'


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, response, code, message, headers, newurl):
        return None


@unittest.skipUnless(os.environ.get('CPA_HELPER_TEST_NGINX'), 'Set CPA_HELPER_TEST_NGINX to run real proxy regression')
class NginxLogoutProxyTests(unittest.TestCase):
    def test_deployed_proxy_respects_logout_bypass_and_keeps_password_api_blocked(self):
        class Backend(BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(200)
                self.send_header('Content-Type', 'text/html')
                self.end_headers()
                self.wfile.write(b'<html>CPA login fixture</html>')

            def do_POST(self):
                self.do_GET()

            def log_message(self, *_):
                pass

        backend = ThreadingHTTPServer(('127.0.0.1', 0), Backend)
        threading.Thread(target=backend.serve_forever, daemon=True).start()
        self.addCleanup(backend.server_close)
        self.addCleanup(backend.shutdown)
        with socket.socket() as listener:
            listener.bind(('127.0.0.1', 0))
            port = listener.getsockname()[1]
        nginx = os.environ['CPA_HELPER_TEST_NGINX']
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'logs').mkdir()
            site = TEMPLATE.read_text(encoding='utf-8').split('\nserver {', 1)[1]
            site = 'server {' + site
            site = site.replace('listen 443 ssl;', 'listen 127.0.0.1:%d;' % port)
            site = re.sub(r'(?m)^\s*ssl_.*;\s*$', '', site)
            site = site.replace('127.0.0.1:18317', '127.0.0.1:%d' % backend.server_port)
            config = root / 'nginx.conf'
            config.write_text('daemon off;\npid logs/nginx.pid;\nerror_log logs/error.log;\n'
                              'events {}\nhttp {\nclient_body_temp_path client_temp;\nproxy_temp_path proxy_temp;\n'
                              'fastcgi_temp_path fastcgi_temp;\nuwsgi_temp_path uwsgi_temp;\nscgi_temp_path scgi_temp;\n'
                              'map $http_upgrade $connection_upgrade { default upgrade; "" close; }\n' + site + '\n}', encoding='utf-8')
            args = [nginx, '-p', str(root) + '/', '-c', str(config)]
            subprocess.run(args + ['-t'], check=True, capture_output=True)
            process = subprocess.Popen(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            try:
                client = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())

                def request(path, method='GET'):
                    req = urllib.request.Request('http://127.0.0.1:%d%s' % (port, path), method=method)
                    try:
                        return client.open(req, timeout=3)
                    except urllib.error.HTTPError as error:
                        return error

                for _ in range(50):
                    try:
                        with request('/api/ready') as response:
                            self.assertEqual(response.status, 200)
                        break
                    except urllib.error.URLError:
                        time.sleep(0.05)
                else:
                    self.fail('nginx did not become ready')
                with request('/login?redirect=%2Faccount%2Fprofile') as response:
                    self.assertEqual(response.status, 302)
                    self.assertIn('/oidc/login?returnTo=%2Faccount%2Fprofile', response.headers['Location'])
                for suffix in ('?skipsso=true', '?skipsso=true&sso_logout=local'):
                    for _ in range(2):  # also reproduce refreshing the post-logout page
                        with request('/login' + suffix) as response:
                            self.assertEqual(response.status, 200)
                            self.assertIsNone(response.headers.get('Location'))
                            self.assertIn('no-store', response.headers['Cache-Control'])
                            self.assertIn(b'CPA login fixture', response.read())
                with request('/api/auth/login', 'POST') as response:
                    self.assertEqual(response.status, 403)
                for path in ('/v0/management/config', '/v8/management/plugins'):
                    with request(path) as response:
                        self.assertEqual(response.status, 404)
            finally:
                process.terminate()
                process.communicate(timeout=5)


if __name__ == '__main__':
    unittest.main()
