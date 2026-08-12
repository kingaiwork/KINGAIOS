package statuspub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicStatusIsSanitizedAndReadable(t *testing.T) {
	p := filepath.Join(t.TempDir(), "status.json")
	s := Snapshot{Product:"KINGAI OS", Version:"0.1", Architecture:"D4", Health:"ok", Policy:"enabled", RegisteredAgents:3, ModelStrategy:"provider-neutral", ModelMode:"hybrid", MemoryMode:"local-first"}
	if err := Write(p, s); err != nil { t.Fatal(err) }
	b, err := os.ReadFile(p); if err != nil { t.Fatal(err) }
	var got Snapshot
	if err := json.Unmarshal(b, &got); err != nil { t.Fatal(err) }
	if got.Product != "KINGAI OS" || got.RegisteredAgents != 3 { t.Fatalf("unexpected status: %#v", got) }
	text := strings.ToLower(string(b))
	for _, forbidden := range []string{"prompt", "token", "secret", "memory_content", "api_key", "password"} {
		if strings.Contains(text, forbidden) { t.Fatalf("public status contains forbidden field %q", forbidden) }
	}
	st, err := os.Stat(p); if err != nil { t.Fatal(err) }
	if st.Mode().Perm() != 0o644 { t.Fatalf("status mode=%o want 644", st.Mode().Perm()) }
}
