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

func TestDefinitionsAreDefensiveAndDeduplicateCapabilities(t *testing.T) {
	r := New([]Definition{{ID:"main",Role:"assistant",Capabilities:[]string{"filesystem.read","filesystem.read","network.read"}}})
	defs := r.Definitions()
	if len(defs) != 1 || defs[0].ID != "main" || defs[0].Role != "assistant" { t.Fatalf("unexpected definitions: %#v", defs) }
	if len(defs[0].Capabilities) != 2 { t.Fatalf("capabilities were not deduplicated: %#v", defs[0].Capabilities) }
	defs[0].ID = "tampered"
	defs[0].Capabilities[0] = "security.modify"
	again := r.Definitions()
	if again[0].ID != "main" || again[0].Capabilities[0] != "filesystem.read" { t.Fatalf("Definitions leaked mutable registry state: %#v", again) }
	if r.Allows("main", "security.modify") { t.Fatal("mutating returned definitions must not alter authorization") }
}

func TestDuplicateDefinitionLastOneWinsConsistently(t *testing.T) {
	r := New([]Definition{
		{ID:"main",Role:"old",Capabilities:[]string{"filesystem.read"}},
		{ID:"main",Role:"new",Capabilities:[]string{"network.read"}},
	})
	defs := r.Definitions()
	if len(defs) != 1 || defs[0].Role != "new" { t.Fatalf("unexpected definitions: %#v", defs) }
	if r.Allows("main", "filesystem.read") { t.Fatal("duplicate replacement must not retain stale capability") }
	if !r.Allows("main", "network.read") { t.Fatal("duplicate replacement must keep latest capability set") }
}
