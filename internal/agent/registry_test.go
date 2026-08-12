package agent

import "testing"

func TestRegistryLeastPrivilege(t *testing.T) {
	r := New([]Definition{
		{ID:"main",Capabilities:[]string{"filesystem.read"}},
		{ID:"system-ops",Capabilities:[]string{"filesystem.read","service.restart"}},
	})
	if !r.Allows("main","filesystem.read") { t.Fatal("declared capability should be allowed") }
	if r.Allows("main","service.restart") { t.Fatal("main must not inherit system-ops capability") }
	if r.Allows("unknown","filesystem.read") { t.Fatal("unknown agent must be denied") }
}

func TestFallbackIsReadOnly(t *testing.T) {
	r := Default()
	if !r.Allows("main","filesystem.read") { t.Fatal("fallback main should read") }
	if r.Allows("main","process.execute") { t.Fatal("fallback registry must not execute") }
}
