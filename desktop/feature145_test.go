package main

import (
	"net/url"
	"testing"
)

func TestCloneURLValuesDoesNotMutateOriginal(t *testing.T) {
	in := url.Values{"where": {"1=1"}}
	out := cloneURLValues(in)
	out.Set("where", "x=1")
	if in.Get("where") != "1=1" {
		t.Fatalf("clone alterou o original: %q", in.Get("where"))
	}
}

func TestIBAMASourceLabels(t *testing.T) {
	if got := ibamaSourceLabel(true); got == "" {
		t.Fatal("fonte IBAMA atual sem rótulo")
	}
	if got := ibamaSourceLabel(false); got == "" {
		t.Fatal("fallback IBAMA legado sem rótulo")
	}
}
