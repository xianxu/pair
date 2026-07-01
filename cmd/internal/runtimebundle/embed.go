package runtimebundle

import (
	"embed"
	"encoding/json"
)

//go:embed assets/runtime/manifest.json assets/runtime/files
var embedded embed.FS

func EmbeddedManifest() RuntimeManifest {
	data, err := embedded.ReadFile("assets/runtime/manifest.json")
	if err != nil {
		panic(err)
	}
	var manifest RuntimeManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		panic(err)
	}
	return manifest
}

func EmbeddedAsset(path string) ([]byte, error) {
	return embedded.ReadFile("assets/runtime/files/" + path)
}
