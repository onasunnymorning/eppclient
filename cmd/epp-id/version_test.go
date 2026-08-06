package main

import "testing"

func TestFormatBuildInfo(t *testing.T) {
	got := formatBuildInfo("go1.26.5")
	want := `version.BuildInfo{Version:"EPP-ID 20260806", Maintainer:"freddy@pandi.id", GoVersion:"go1.26.5"}`
	if got != want {
		t.Fatalf("formatBuildInfo() = %q, want %q", got, want)
	}
}
