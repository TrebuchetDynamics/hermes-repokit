package development

import "embed"

// Frozen generated recipe from e0246ef1d2f30cdb4fa174a0b72ad39e81c99576,
// immediately before the login-shell PATH fix. Do not derive these bytes from
// current assets: upgrades must never silently widen the accepted preimage.
//
//go:embed all:legacy
var legacyAssets embed.FS

func LegacyFingerprint(req Requirements) string {
	if req.Go {
		return "617671ef605ee7a176cf809ae663eec87419d5f094897f9fabcf0120422e3335"
	}
	return "8ec197a291d0200317e88b6d7654fd0cc91ca9dec342ad0b03ba50d070a591d9"
}

func LegacyRecipe(req Requirements) (map[string][]byte, error) {
	files := make(map[string][]byte)
	for _, name := range []string{"Dockerfile", ".dockerignore", "repokit-docker-test", "repokit-openviking", "openviking-run", "openviking-finish", "patch-openviking-entrypoint.py"} {
		source := name
		if name == "Dockerfile" {
			source = "Dockerfile-base"
			if req.Go {
				source = "Dockerfile-go"
			}
		}
		data, err := legacyAssets.ReadFile("legacy/" + source)
		if err != nil {
			return nil, err
		}
		files[name] = data
	}
	return files, nil
}
