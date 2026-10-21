package couchschedule

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

const (
	maxBuildInputBytes   = 64 << 20
	maxBuildInputFiles   = 10000
	maxBuildCommandBytes = 8 << 20
)

type buildInput struct {
	Module, Path string
	Data         []byte
}

type BuildManifest struct {
	Version                      int
	Key, BinarySHA256, Toolchain string
	LastUsed                     int64 `json:"last_used,omitempty"`
	Inputs                       []BuildInputDigest
}

type BuildInputDigest struct{ Module, Path, SHA256 string }

func digestBuild(raw []byte) string {
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}

func manifestFor(toolchain string, inputs []buildInput) BuildManifest {
	sort.Slice(inputs, func(i, j int) bool {
		if inputs[i].Module != inputs[j].Module {
			return inputs[i].Module < inputs[j].Module
		}
		return inputs[i].Path < inputs[j].Path
	})
	m := BuildManifest{Version: 1, Toolchain: toolchain}
	for _, in := range inputs {
		m.Inputs = append(m.Inputs, BuildInputDigest{in.Module, in.Path, digestBuild(in.Data)})
	}
	raw, _ := json.Marshal(m)
	m.Key = digestBuild(raw)
	return m
}
