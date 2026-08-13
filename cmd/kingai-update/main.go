package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	kingupdate "github.com/kingaiwork/KINGAIOS/internal/update"
)

var version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 { usage() }
	switch os.Args[1] {
	case "verify": verify(os.Args[2:])
	case "stage": stage(os.Args[2:])
	case "boot-health": bootHealth(os.Args[2:])
	default: usage()
	}
}

func verify(args []string) {
	if len(args) != 3 { usage() }
	b, err := os.ReadFile(args[0]); if err != nil { fail(err) }
	var e kingupdate.Envelope
	if err := json.Unmarshal(b,&e); err != nil { fail(err) }
	pub, err := kingupdate.LoadPublicKey(args[1]); if err != nil { fail(err) }
	if err := kingupdate.VerifyEnvelope(e,pub); err != nil { fail(err) }
	if err := kingupdate.VerifyArtifact(args[2],e.Manifest); err != nil { fail(err) }
	fmt.Printf("KINGAI update verified: version=%s artifact=%s verifier=%s\n",e.Manifest.Version,e.Manifest.Artifact,version)
}

func stage(args []string) {
	fs:=flag.NewFlagSet("stage",flag.ExitOnError)
	target:=fs.String("target-disk","","exact installed KINGAI target disk")
	source:=fs.String("source-root","","verified target KINGAI root filesystem")
	stateKey:=fs.String("state-key","","LUKS STATE key file")
	targetVersion:=fs.String("target-version","","target VERSION_ID")
	confirm:=fs.String("confirm","","must equal UPDATE:<exact target disk>")
	_ = fs.Parse(args)
	jsonOut:=os.Stdout
	os.Stdout=os.Stderr
	res,err:=kingupdate.ExecuteStage(kingupdate.ExecuteOptions{TargetDisk:*target,SourceRoot:*source,StateKey:*stateKey,TargetVersion:*targetVersion,Confirmation:*confirm})
	os.Stdout=jsonOut
	if err!=nil{fail(err)}
	enc:=json.NewEncoder(os.Stdout);enc.SetIndent("","  ");_ = enc.Encode(res)
}

func bootHealth(args []string) {
	fs:=flag.NewFlagSet("boot-health",flag.ExitOnError)
	state:=fs.String("state",kingupdate.DefaultStatePath,"slot state JSON on encrypted STATE")
	_ = fs.Parse(args)
	result,err:=kingupdate.ReconcileHealthyBoot(*state);if err!=nil{fail(err)}
	fmt.Println("KINGAI A/B health:",result)
}

func usage(){fmt.Fprintln(os.Stderr,"usage: kingai-update verify <envelope.json> <public-key.b64> <artifact> | stage --target-disk <disk> --source-root <rootfs> --state-key <file> --target-version <version> --confirm UPDATE:<disk> | boot-health [--state <slots.json>]");os.Exit(2)}
func fail(err error){ fmt.Fprintln(os.Stderr,"update failed:",err); os.Exit(1) }
