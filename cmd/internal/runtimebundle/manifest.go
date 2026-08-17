package runtimebundle

import "github.com/xianxu/pair/cmd/internal/runtimebundle/manifestmodel"

type RuntimeAsset = manifestmodel.RuntimeAsset
type RuntimeManifest = manifestmodel.RuntimeManifest

func digestFor(s string) string {
	return manifestmodel.DigestFor(s)
}
