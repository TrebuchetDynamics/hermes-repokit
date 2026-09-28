import importlib.util
import json
import os
from pathlib import Path
import sys
import tempfile
import types
import unittest
sys.modules['yaml']=types.SimpleNamespace(safe_load=json.loads)
spec=importlib.util.spec_from_file_location('memory',Path(__file__).with_name('memory.py'))
memory=importlib.util.module_from_spec(spec)
spec.loader.exec_module(memory)

class MemoryTest(unittest.TestCase):
 def setUp(self):
  self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup)
  self.root=Path(self.tmp.name);self.roles=['default','researcher','planner','executor','reviewer','steward'];self.calls=[]
  link=self.root/'.openviking/ovcli.conf.repository';link.parent.mkdir();link.write_text('{}');link.chmod(0o600);self.link=str(link)
  for name in self.roles:
   home=self.root if name=='default' else self.root/'profiles'/name;home.mkdir(parents=True,exist_ok=True)
   config={'owner':'keep','memory':{'provider':'builtin','owner_setting':17}}
   if name=='default':config['memory'].update(provider='openviking',openviking={'use_ovcli_config':True,'ovcli_config_path':self.link})
   (home/'config.yaml').write_text(json.dumps(config))
 def native(self,*args):
  self.calls.append(args)
  home=self.root if args[1]=='default' else self.root/'profiles'/args[1]
  c=json.loads((home/'config.yaml').read_text());parts=args[4].split('.');target=c
  for part in parts[:-1]:target=target.setdefault(part,{})
  try:value=json.loads(args[5])
  except ValueError:value=args[5]
  target[parts[-1]]=value;(home/'config.yaml').write_text(json.dumps(c))
 def check(self,home,candidate):
  self.assertEqual(candidate['openviking']['ovcli_config_path'],self.link)
 def test_effective_connection_rejects_profile_overrides_and_linked_peers(self):
  shared={'endpoint':'http://openviking:1933','api_key':'fixture','account':'','user':'','agent':''}
  for field,value in [('endpoint','http://owner:1933'),('api_key','other-fixture'),('account','other-account'),('user','other-repo'),('agent','peer')]:
   with self.subTest(field=field):
    effective=dict(shared,**{field:value})
    ov=types.SimpleNamespace(_ovcli_values_for=lambda _:shared,_resolve_connection_settings=lambda _:effective)
    with self.assertRaises(RuntimeError):memory.resolve_connection(ov,{},'repository')
  ov=types.SimpleNamespace(_ovcli_values_for=lambda _:dict(shared,agent='linked-peer'),_resolve_connection_settings=lambda _:shared)
  with self.assertRaises(RuntimeError):memory.resolve_connection(ov,{},'repository')
 def test_effective_connection_accepts_same_native_link(self):
  shared={'endpoint':'http://openviking:1933','api_key':'fixture','account':'','user':'','agent':''}
  ov=types.SimpleNamespace(_ovcli_values_for=lambda _:shared,_resolve_connection_settings=lambda _:shared)
  self.assertEqual(memory.resolve_connection(ov,{},'repository'),shared)
 def test_share_native_link_without_copying_credentials(self):
  memory.configure(self.root,self.roles,self.check,self.native)
  for name in self.roles:
   home=self.root if name=='default' else self.root/'profiles'/name
   c=json.loads((home/'config.yaml').read_text())
   self.assertEqual(c['owner'],'keep');self.assertEqual(c['memory']['owner_setting'],17)
   self.assertEqual(c['memory']['provider'],'openviking')
   self.assertTrue(c['memory']['memory_enabled']);self.assertTrue(c['memory']['user_profile_enabled'])
   self.assertEqual(c['memory']['openviking']['ovcli_config_path'],self.link)
   self.assertFalse((home/'.env').exists())
 def test_conflicting_provider_refuses_before_any_profile_write(self):
  p=self.root/'profiles/reviewer/config.yaml';c=json.loads(p.read_text());c['memory']['provider']='owner-provider';p.write_text(json.dumps(c))
  with self.assertRaises(RuntimeError):memory.configure(self.root,self.roles,self.check,self.native)
  self.assertEqual(self.calls,[])
 def test_failed_effective_connection_refuses_before_any_write(self):
  def rejected(home,c):
   if home.name=='reviewer':raise RuntimeError('private endpoint must not be printed')
  with self.assertRaises(RuntimeError):memory.configure(self.root,self.roles,rejected,self.native)
  self.assertEqual(self.calls,[])
 def test_unshared_or_unsafe_link_refused(self):
  Path(self.link).chmod(0o644)
  with self.assertRaises(RuntimeError):memory.configure(self.root,self.roles,self.check,self.native)
  self.assertEqual(self.calls,[])
 def test_dormant_owner_connection_drift_refuses_before_any_write(self):
  for prior in ({'ovcli_config_path':'/owner/ovcli.conf'}, {'agent':'owner-peer'}, {'endpoint':'http://owner:1933'}, {'use_ovcli_config':True}):
   with self.subTest(prior=prior):
    self.calls=[]
    p=self.root/'profiles/reviewer/config.yaml';c=json.loads(p.read_text());c['memory'].update(provider='builtin',openviking=prior);p.write_text(json.dumps(c))
    with self.assertRaises(RuntimeError):memory.configure(self.root,self.roles,self.check,self.native)
    self.assertEqual(self.calls,[])
if __name__=='__main__':unittest.main()
