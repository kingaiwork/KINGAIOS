package main

import (
	"encoding/json"
	"fmt"
	"os"

	kingupdate "github.com/kingaiwork/KINGAIOS/internal/update"
)

var version = "0.1.0-dev"

func main() {
	if len(os.Args) != 5 || os.Args[1] != "verify" {
		fmt.Fprintln(os.Stderr, "usage: kingai-update verify <envelope.json> <public-key.b64> <artifact>")
		os.Exit(2)
	}
	b, err := os.ReadFile(os.Args[2]); if err != nil { fail(err) }
	var e kingupdate.Envelope
	if err := json.Unmarshal(b,&e); err != nil { fail(err) }
	pub, err := kingupdate.LoadPublicKey(os.Args[3]); if err != nil { fail(err) }
	if err := kingupdate.VerifyEnvelope(e,pub); err != nil { fail(err) }
	if err := kingupdate.VerifyArtifact(os.Args[4],e.Manifest); err != nil { fail(err) }
	fmt.Printf("KINGAI update verified: version=%s artifact=%s verifier=%s\n",e.Manifest.Version,e.Manifest.Artifact,version)
}
func fail(err error){ fmt.Fprintln(os.Stderr,"verification failed:",err); os.Exit(1) }
