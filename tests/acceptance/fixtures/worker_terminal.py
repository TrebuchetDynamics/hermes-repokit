"""Exercise pinned Hermes terminal snapshots without models or host mounts."""
import os
from pathlib import Path
import tempfile

from tools.environments.local import LocalEnvironment

with tempfile.TemporaryDirectory(prefix='repokit-worker-') as tmp:
    root = Path(tmp)
    for profile in ('default', 'executor', 'reviewer', 'steward'):
        home = root / profile
        home.mkdir()
        os.environ['HOME'] = str(home)
        os.environ['HERMES_HOME'] = str(home)
        os.environ['HERMES_PROFILE'] = profile
        env = LocalEnvironment(cwd='/workspace', timeout=90)
        try:
            # Both initial and reused snapshots must retain the image commands.
            for _ in range(2):
                result = env.execute('command -v hermes && hermes --version && command -v go && go version && command -v gofmt')
                assert result['returncode'] == 0, (profile, result)
                assert 'go1.26.6' in result['output'], (profile, result)
            work = home / 'project'
            work.mkdir()
            (work / 'go.mod').write_text('module fixture\n\ngo 1.26.0\n')
            (work / 'fixture_test.go').write_text('package fixture\nimport "testing"\nfunc TestWorker(t *testing.T) { if 2+2 != 4 { t.Fatal("compiler") } }\n')
            if profile == 'executor':
                result = env.execute(f'cd {work} && GOTOOLCHAIN=local GOCACHE={home}/build GOMODCACHE={home}/mod go test -race ./...')
                assert result['returncode'] == 0, result
        finally:
            env.cleanup()
print('PASS: native terminal lookup, repeated snapshots, and executor compilation')
