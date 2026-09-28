import importlib.machinery
import importlib.util
import os
from pathlib import Path
import subprocess
import tempfile
import json
import sys
from unittest.mock import patch, Mock
import signal
import unittest

HERE = Path(__file__).resolve().parent
loader = importlib.machinery.SourceFileLoader('docker_test_helper', str(HERE / 'repokit-docker-test'))
spec = importlib.util.spec_from_loader(loader.name, loader)
helper = importlib.util.module_from_spec(spec)
loader.exec_module(helper)

class HelperTests(unittest.TestCase):
    def test_snapshot_excludes_private_ignored_backup_and_links(self):
        with tempfile.TemporaryDirectory() as tmp:
            source = Path(tmp) / 'source'; source.mkdir()
            subprocess.run(['git', 'init', '-q', str(source)], check=True)
            for name, data in {'.gitignore': 'ignored\n', 'tracked.go': 'source', 'new.go': 'new', 'ignored/key': 'secret', '.hermes/auth.json': 'secret', 'backups/auth.json': 'secret', 'old.bak': 'secret'}.items():
                p=source/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_text(data)
            subprocess.run(['git','-C',str(source),'add','-f','tracked.go','.hermes/auth.json','ignored/key'],check=True)
            (source/'link').symlink_to(source/'.hermes/auth.json')
            destination=Path(tmp)/'snapshot';destination.mkdir()
            helper.snapshot(source,destination,os.environ.copy())
            self.assertEqual((destination/'tracked.go').read_text(),'source')
            self.assertEqual((destination/'new.go').read_text(),'new')
            for name in ['.hermes','.git','ignored','backups','old.bak','link']:
                self.assertFalse((destination/name).exists(),name)

    def test_private_environment_drops_credentials_and_redirects_docker(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            env=helper.fixture_environment(root,{'PATH':'/unsafe','DOCKER_HOST':'tcp://host:2375','DOCKER_CONTEXT':'host','DOCKER_CONFIG':'/private','DOCKER_TLS_VERIFY':'1','AWS_SECRET_ACCESS_KEY':'secret','OPENAI_API_KEY':'secret','GIT_DIR':'/private','SSH_AUTH_SOCK':'/private'})
            self.assertEqual(env['DOCKER_HOST'],helper.SOCKET)
            self.assertEqual(env['DOCKER_CONTEXT'],'default')
            self.assertEqual(env['REPOKIT_DOCKER_CONTEXT'],'default')
            self.assertEqual(env['TMPDIR'],str(root/'tmp'))
            self.assertEqual(env['HOME'],str(root/'home'))
            self.assertEqual(env['REPOKIT_DOCKER_TESTS'],'1')
            for name in ['AWS_SECRET_ACCESS_KEY','OPENAI_API_KEY','DOCKER_TLS_VERIFY','GIT_DIR','SSH_AUTH_SOCK']:
                self.assertNotIn(name,env)
            self.assertNotIn('/unsafe',env['PATH'])

    def test_docker_global_options_cannot_redirect_daemon(self):
        self.assertEqual(helper.safe_docker_args(['--context','default','ps']),['ps'])
        self.assertEqual(helper.safe_docker_args(['--context=default','context','inspect','default']),['context','inspect','default'])
        self.assertEqual(helper.safe_docker_args(['run','--env','DOCKER_HOST=other','image']),['run','--env','DOCKER_HOST=other','image'])
        for args in [['--context','host','ps'],['--host=unix:///var/run/docker.sock','ps'],['-H','tcp://host','ps'],['--config','/private','ps'],['ps','--config=/private'],['--tlsverify','ps'],['--context','default','--host','x','ps']]:
            with self.assertRaises(ValueError,msg=str(args)):helper.safe_docker_args(args)


    def test_runs_in_private_snapshot_and_removes_only_own_fixture(self):
        with tempfile.TemporaryDirectory() as tmp:
            base=Path(tmp);source=base/'source';source.mkdir()
            subprocess.run(['git','init','-q',str(source)],check=True)
            (source/'dirty.txt').write_text('current edits')
            (source/'.hermes').mkdir();(source/'.hermes/auth.json').write_text('secret')
            scratch=base/'scratch';scratch.mkdir(mode=0o700)
            unrelated=scratch/'other-fixture';unrelated.mkdir();(unrelated/'keep').write_text('keep')
            (base/'docker.sock').touch()
            docker=base/'docker';docker.write_text('#!/usr/bin/python3\nprint(["io.repokit.docker-test=1"].__str__().replace(chr(39),chr(34)))\n');docker.chmod(0o700)
            report=base/'report.json'
            command=[sys.executable,'-c',
                'import os,json,pathlib; p=pathlib.Path.cwd(); '
                'pathlib.Path('+repr(str(report))+').write_text(json.dumps({"cwd":str(p),"tmp":os.environ["TMPDIR"],"host":os.environ["DOCKER_HOST"],"secret":os.environ.get("OPENAI_API_KEY"),"data":(p/"dirty.txt").read_text(),"private":(p/".hermes").exists(),"git":(p/".git").is_dir()}))']
            with patch.object(helper,'SOURCE',source),patch.object(helper,'SCRATCH',scratch),patch.object(helper,'SOCKET','unix://'+str(base/'docker.sock')),patch.object(helper,'DOCKER',str(docker)),patch.dict(os.environ,{'OPENAI_API_KEY':'secret'}),patch.object(helper.stat,'S_ISSOCK',return_value=True):
                self.assertEqual(helper.run_fixture(command,10),0)
            data=json.loads(report.read_text())
            self.assertTrue(data['cwd'].startswith(str(scratch)))
            self.assertTrue(data['tmp'].startswith(str(scratch)))
            self.assertEqual(data['data'],'current edits')
            self.assertIsNone(data['secret']);self.assertFalse(data['private']);self.assertTrue(data['git'])
            self.assertEqual(list(scratch.iterdir()),[unrelated])
            self.assertEqual((source/'.hermes/auth.json').read_text(),'secret')


    def test_only_new_fixture_resources_are_labeled_for_cleanup(self):
        self.assertEqual(helper.label_owned_resources(['run','--rm','image'], 'fixture-abc'),['run','--label','io.repokit.docker-test.fixture=fixture-abc','--rm','image'])
        self.assertEqual(helper.label_owned_resources(['volume','create','volume'], 'fixture-abc'),['volume','create','--label','io.repokit.docker-test.fixture=fixture-abc','volume'])
        self.assertEqual(helper.label_owned_resources(['compose','up','-d'], 'fixture-abc'),['compose','up','-d'])


    def test_cleanup_is_bounded_by_exact_fixture_label(self):
        calls=[]
        def run(args,**kwargs):
            calls.append(args)
            output=b'abc123\n' if 'ls' in args and 'container' in args else b''
            return subprocess.CompletedProcess(args,0,output,b'')
        with patch.object(helper.subprocess,'run',side_effect=run):
            helper.cleanup_resources(Path('/docker-tests/fixture-owned'),{})
        self.assertEqual(len(calls),4)
        for args in calls:
            self.assertNotIn('prune',args)
            self.assertIn(helper.SOCKET,args)
            if 'ls' in args:self.assertIn('label=io.repokit.docker-test.fixture=fixture-owned',args)
            else:self.assertEqual(args[-3:],['--force','--','abc123'])


    def test_cleanup_kills_descendants_after_parent_has_exited(self):
        child=Mock(pid=12345)
        child.wait.return_value=0
        signals=[]
        def killpg(pid, sig):signals.append((pid,sig))
        with patch.object(helper.os,'killpg',side_effect=killpg),patch.object(helper.time,'monotonic',side_effect=[0.0,0.0,0.1,0.2,0.4]),patch.object(helper.time,'sleep'):
            helper.terminate_group(child,grace=0.3)
        self.assertEqual(signals[0],(12345,signal.SIGTERM))
        self.assertIn((12345,signal.SIGKILL),signals)
        child.wait.assert_called_once_with(timeout=3)

    def test_cleanup_does_not_signal_a_gone_process_group_again(self):
        child=Mock(pid=12345)
        with patch.object(helper.os,'killpg',side_effect=ProcessLookupError) as killpg:
            helper.terminate_group(child,grace=0.1)
        killpg.assert_called_once_with(12345,signal.SIGTERM)
        child.wait.assert_called_once_with(timeout=3)


    def test_snapshot_excludes_known_credentials_even_when_force_tracked(self):
        with tempfile.TemporaryDirectory() as tmp:
            source=Path(tmp)/'source';source.mkdir()
            subprocess.run(['git','init','-q',str(source)],check=True)
            denied=['.env','.env.local','profiles/default/.env.production','auth.json',
                    'profiles/default/auth.json','credentials.json','nested/credentials.json',
                    '__pycache__/module.pyc','nested/__pycache__/module.pyc','module.pyc']
            allowed=['app.py','.env.example','nested/.env.sample']
            for name in denied+allowed:
                path=source/name;path.parent.mkdir(parents=True,exist_ok=True)
                path.write_text('synthetic fixture data')
            subprocess.run(['git','-C',str(source),'add','-f','--']+denied+allowed,check=True)
            destination=Path(tmp)/'snapshot';destination.mkdir()
            helper.snapshot(source,destination,os.environ.copy())
            for name in denied:self.assertFalse((destination/name).exists(),name)
            for name in allowed:self.assertEqual((destination/name).read_text(),'synthetic fixture data')

if __name__ == '__main__':unittest.main()
