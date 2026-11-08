import importlib.util
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import urllib.error

spec = importlib.util.spec_from_file_location('group_scheduler', Path(__file__).with_name('group_scheduler.py'))
scheduler = importlib.util.module_from_spec(spec)
spec.loader.exec_module(scheduler)


class GroupSchedulerDeployTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.deployment = Path(self.temp.name)
        (self.deployment / 'group-scheduler.env').write_text('CPA_GROUP_SCHEDULER_STRATEGY=round-robin\n')

    def inventory(self):
        return {'plugins': [{'id': scheduler.PLUGIN, 'registered': True,
                             'effective_enabled': True, 'metadata': {'version': '1.0.0'}}]}

    def test_enable_preserves_etag_and_writes_only_owned_plugin(self):
        calls = []
        def request(key, method, path, payload=None, revision=None):
            calls.append((method, path, payload, revision))
            if path == '/config/plugins':
                return {'enabled': False, 'configs': {'other': {'enabled': False}}}, '"snapshot-1"'
            if path == '/config/routing':
                return {'strategy': 'round-robin'}, '"snapshot-1"'
            if path == '/plugins':
                return self.inventory(), ''
            return {'status': 'ok'}, ''
        with patch.object(scheduler, 'core_request', side_effect=request):
            scheduler.configure_scheduler('private-key', self.deployment)
        writes = [call for call in calls if call[0] == 'PATCH']
        self.assertEqual(len(writes), 1)
        self.assertEqual(writes[0][3], '"snapshot-1"')
        self.assertEqual(writes[0][2], {'plugins': {'enabled': True, 'dir': 'plugins', 'configs': {
            scheduler.PLUGIN: {'enabled': True, 'priority': 1}}}})

    def test_conflict_is_not_retried_or_overwritten(self):
        writes = []
        def request(key, method, path, payload=None, revision=None):
            if method == 'PATCH':
                writes.append(payload)
                raise urllib.error.HTTPError('http://localhost/config', 412, 'conflict', {}, None)
            return ({'enabled': False} if path.endswith('/plugins') else {'strategy': 'round-robin'}), '"old"'
        with patch.object(scheduler, 'core_request', side_effect=request):
            with self.assertRaises(urllib.error.HTTPError):
                scheduler.configure_scheduler('private-key', self.deployment)
        self.assertEqual(len(writes), 1)

    def test_other_plugin_is_not_implicitly_activated_or_replaced(self):
        with patch.object(scheduler, 'core_request', return_value=(
                {'enabled': False, 'configs': {'other': {'enabled': True}}}, '"revision"')) as request:
            with self.assertRaises(RuntimeError):
                scheduler.configure_scheduler('private-key', self.deployment)
        self.assertEqual(request.call_count, 1)

    def test_registered_but_inactive_plugin_fails_health_check(self):
        inventory = self.inventory()
        inventory['plugins'][0]['effective_enabled'] = False
        with patch.object(scheduler, 'core_request', return_value=(inventory, '')):
            with self.assertRaises(RuntimeError):
                scheduler.check_scheduler('private-key')

    def test_unsupported_strategy_fails_instead_of_changing_routing(self):
        with patch.object(scheduler, 'core_request', return_value=({'strategy': 'weighted-round-robin'}, '')):
            with self.assertRaises(RuntimeError):
                scheduler.routing_strategy('private-key')

    def test_unpinned_image_is_rejected_before_any_docker_operation(self):
        deployment = self.deployment / 'deploy'
        deployment.mkdir()
        (self.deployment / '.env').write_text('CPA_HELPER_IMAGE=untrusted:latest\n')
        with patch.object(scheduler.subprocess, 'check_output') as command:
            with self.assertRaises(RuntimeError):
                scheduler.prepare_scheduler(self.deployment, deployment, 'private-key')
        command.assert_not_called()


if __name__ == '__main__':
    unittest.main()
