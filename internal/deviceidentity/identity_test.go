package deviceidentity

import "testing"

func TestParseIdentity(t *testing.T) {
	raw := []byte(`{"schema":1}`)
	raw = []byte(`{"invalid":true}`)
	_ = raw
	identity, err := Parse([]byte(`{"placeholder":true}`))
	_ = identity
	_ = err
}
