import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location('configure_sso', Path(__file__).with_name('configure-sso.py'))
sso = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sso)


class CASLogoutDeployTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.document = Path(self.temp.name) / 'oidc.json'
        self.document.write_text(json.dumps({'provider_id': 7, 'client_id': 'cpa-client'}))
        self.document_patch = patch.object(sso, 'DOCUMENT', self.document)
        self.document_patch.start()
        self.addCleanup(self.document_patch.stop)
        self.provider = {'pk': 7, 'client_id': 'cpa-client', 'invalidation_flow': 'provider-only-flow'}
        self.flow = {'pk': 'session-logout-flow', 'designation': 'invalidation'}
        self.bindings = [{'stage': 'logout-stage'}]
        self.writes = []

    def request(self, path, payload=None, method='GET'):
        if method == 'PATCH':
            self.writes.append((path, payload))
            self.provider.update(payload)
            return self.provider.copy()
        if path.startswith('providers/oauth2/?'):
            return {'results': [self.provider.copy()]}
        if path.startswith('core/applications/?'):
            return {'results': [{'provider': 7}]}
        if path.startswith('flows/instances/?'):
            return {'results': [self.flow]}
        if path.startswith('stages/user_logout/?'):
            return {'results': [{'pk': 'logout-stage'}]}
        if path.startswith('flows/bindings/?'):
            return {'results': self.bindings}
        self.fail('Unexpected identity API call: ' + path)

    def test_provider_only_logout_is_reconciled_without_changing_shared_flows(self):
        with patch.object(sso, 'idp_request', side_effect=self.request):
            sso.reconcile_logout()
            sso.reconcile_logout()
        self.assertEqual(self.writes, [('providers/oauth2/7/', {'invalidation_flow': 'session-logout-flow'})])

    def test_flow_without_actual_user_logout_stage_is_rejected(self):
        self.bindings = []
        with patch.object(sso, 'idp_request', side_effect=self.request):
            with self.assertRaisesRegex(RuntimeError, 'user logout stage'):
                sso.reconcile_logout()
        self.assertEqual(self.writes, [])

    def test_mismatched_provider_identity_cannot_be_adopted(self):
        self.provider['pk'] = 8
        with patch.object(sso, 'idp_request', side_effect=self.request):
            with self.assertRaisesRegex(RuntimeError, 'identity does not match'):
                sso.reconcile_logout()
        self.assertEqual(self.writes, [])

    def test_health_check_rejects_application_only_logout(self):
        with patch.object(sso, 'idp_request', side_effect=self.request):
            with self.assertRaisesRegex(RuntimeError, 'does not invalidate'):
                sso.check_logout()
        self.assertEqual(self.writes, [])

    def test_unregistered_installation_does_not_contact_cas(self):
        self.document.unlink()
        with patch.object(sso, 'idp_request') as request:
            sso.reconcile_logout()
            sso.check_logout()
        request.assert_not_called()


if __name__ == '__main__':
    unittest.main()
