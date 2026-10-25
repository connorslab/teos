import importlib.util
import json
from pathlib import Path
import tempfile
import tomllib
import unittest

spec = importlib.util.spec_from_file_location("configure", Path(__file__).with_name("configure.py"))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class ConfigurationTests(unittest.TestCase):
    def config(self):
        return {"knots": {"host": "node.example", "port": 18443, "cookie": "/run/knots/.cookie"}, "network": "regtest", "lnd": {"enabled": True}}

    def test_lnd_is_optional(self):
        cfg = self.config()
        cfg.pop("lnd")
        self.assertEqual(set(module.render(cfg)), {"teos/teos.toml"})
        cfg["lnd"] = {"enabled": "true"}
        with self.assertRaises(ValueError):
            module.render(cfg)

    def test_shared_backend_and_network(self):
        result = module.render(self.config())
        teos = tomllib.loads(result["teos/teos.toml"])
        lnd = json.loads(result["lnd/tower.json"])
        self.assertEqual(teos["btc_rpc_connect"], "node.example")
        self.assertEqual(teos["btc_rpc_port"], 18443)
        self.assertEqual(teos["btc_network"], "regtest")
        self.assertEqual(lnd["network"], "regtest")
        self.assertEqual(lnd["rpc_url"], "http://node.example:18443")
        self.assertEqual(lnd["cookie"], teos["btc_rpc_cookie"])

    def test_remote_listener_does_not_expose_admin(self):
        cfg = self.config()
        cfg["listeners"] = {"bind": "0.0.0.0"}
        result = module.render(cfg)
        self.assertEqual(tomllib.loads(result["teos/teos.toml"])["rpc_bind"], "127.0.0.1")
        self.assertNotIn("rpc_listen", json.loads(result["lnd/tower.json"]))
        self.assertEqual(json.loads(result["lnd/tower.json"])["listen"], "0.0.0.0:9911")

    def test_rejects_unsafe_or_ambiguous_input(self):
        for extra in [
            {"network":"testnet"}, {"unknown":1}, {"teos":{"api_prt":123}},
            {"lnd":{"tower_port":True}}, {"lnd":{"enabled":True, "tower_port":9814}},
            {"teos":{"renewal_slot_topups":"false"}},
            {"knots":{"cookie":"/tmp/key\n[Bitcoin]\nbitcoin.testnet=1"}},
            {"knots":{"cookie":"relative/cookie"}},
            {"knots":{"cookie":"/tmp/cookie", "host":"node:8332"}},
        ]:
            with self.subTest(extra=extra):
                cfg = self.config()
                cfg.update(extra)
                with self.assertRaises(ValueError):
                    module.render(cfg)

    def test_existing_unmanaged_config_preserved(self):
        with tempfile.TemporaryDirectory() as tmp:
            target = Path(tmp)/"lnd/tower.json"
            target.parent.mkdir()
            target.write_text("operator configuration")
            with self.assertRaises(ValueError):
                module.write_configs(tmp, module.render(self.config()), True)
            self.assertEqual(target.read_text(), "operator configuration")
            self.assertFalse((Path(tmp)/"teos/teos.toml").exists())

    def test_managed_update_and_permissions(self):
        with tempfile.TemporaryDirectory() as tmp:
            result = module.render(self.config())
            module.write_configs(tmp, result)
            with self.assertRaises(ValueError):
                module.write_configs(tmp, result)
            module.write_configs(tmp, result, True)
            for rel in result:
                self.assertEqual((Path(tmp)/rel).read_text(), result[rel])
                if __import__("os").name == "posix":
                    self.assertEqual((Path(tmp)/rel).stat().st_mode & 0o777, 0o600)

    @unittest.skipUnless(__import__("os").name == "posix", "POSIX symlink semantics")
    def test_symlink_output_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            real = Path(tmp)/"real"
            real.mkdir()
            link = Path(tmp)/"link"
            link.symlink_to(real, target_is_directory=True)
            with self.assertRaises(ValueError):
                module.write_configs(link, module.render(self.config()))

if __name__ == "__main__":
    unittest.main()
