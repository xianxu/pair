package couchschedule

import "testing"

func TestBuildManifestIncludesCapturedBytesAndRecipe(t *testing.T) {
	first := manifestFor("toolchain1", []buildInput{{"root", "b", []byte("second")}, {"root", "a", []byte("first")}})
	reordered := manifestFor("toolchain1", []buildInput{{"root", "a", []byte("first")}, {"root", "b", []byte("second")}})
	if first.Key != reordered.Key {
		t.Fatal("enumeration order changed build key")
	}
	changed := manifestFor("toolchain1", []buildInput{{"root", "a", []byte("edited")}, {"root", "b", []byte("second")}})
	if first.Key == changed.Key {
		t.Fatal("changed source retained build key")
	}
	tool := manifestFor("toolchain2", []buildInput{{"root", "a", []byte("first")}, {"root", "b", []byte("second")}})
	if first.Key == tool.Key {
		t.Fatal("changed toolchain retained build key")
	}
}
