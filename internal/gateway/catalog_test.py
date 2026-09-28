import unittest
from catalog import platform_catalog


class CatalogTest(unittest.TestCase):
    def test_metadata_only_no_execution(self):
        source = '''raise RuntimeError("must not execute")
PLATFORMS: object = OrderedDict([
    ("cli", PlatformInfo(label="CLI", default_toolset="hermes-cli")),
    ("telegram", PlatformInfo(label="Telegram", default_toolset="hermes-telegram")),
    ("api_server", PlatformInfo(label="API", default_toolset="hermes-api-server"))])'''
        catalog = platform_catalog(source)
        self.assertEqual(set(catalog), {'cli', 'telegram'})
        self.assertIn('kanban', catalog['telegram']['required'])
        self.assertFalse(catalog['telegram']['default_qualified'])

    def test_unqualified_shape_refused(self):
        with self.assertRaises(ValueError):
            platform_catalog('PLATFORMS = load_credentials()')


if __name__ == '__main__':
    unittest.main()
