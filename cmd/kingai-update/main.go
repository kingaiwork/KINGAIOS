package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/deviceidentity"
	"github.com/kingaiwork/KINGAIOS/internal/devicepack"
	"github.com/kingaiwork/KINGAIOS/internal/tufclient"
	kingupdate "github.com/kingaiwork/KINGAIOS/internal/update"
)

var version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 { usage() }
	switch os.Args[1] {
	case "verify": verify(os.Args[2:])
	case "edge-verify": edgeVerify(os.Args[2:])
	case "tuf-fetch": tufFetch(os.Args[2:])
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

func edgeVerify(args []string) {
	fs:=flag.NewFlagSet("edge-verify",flag.ExitOnError)
	envelopePath:=fs.String("envelope","","signed KINGAI update envelope JSON")
	publicKeyPath:=fs.String("public-key","","KINGAI update Ed25519 public key")
	artifactPath:=fs.String("artifact","","downloaded update artifact")
	identityPath:=fs.String("identity","/etc/kingai/device.json","trusted Edge device identity")
	packDir:=fs.String("device-pack-dir","/etc/kingai/device-packs","verified Device Pack manifest directory")
	artifactRoot:=fs.String("device-artifact-root","/usr/lib/kingai/device-packs","verified Device Pack artifact root")
	packTrust:=fs.String("device-pack-trust","/etc/kingai/trust/device-pack-keys","Device Pack public-key directory")
	handlerRoot:=fs.String("handler-root","/run/kingai-device","protected handler socket root")
	profile:=fs.String("profile","iot","installed KINGAI profile")
	_ = fs.Parse(args)
	if *envelopePath=="" || *publicKeyPath=="" || *artifactPath=="" { fail(fmt.Errorf("--envelope, --public-key and --artifact are required")) }

	b,err:=os.ReadFile(*envelopePath);if err!=nil{fail(err)}
	var e kingupdate.Envelope
	if err:=json.Unmarshal(b,&e);err!=nil{fail(err)}
	pub,err:=kingupdate.LoadPublicKey(*publicKeyPath);if err!=nil{fail(err)}
	if err:=kingupdate.VerifyEnvelope(e,pub);err!=nil{fail(err)}
	if err:=kingupdate.VerifyArtifact(*artifactPath,e.Manifest);err!=nil{fail(err)}
	identity,err:=deviceidentity.LoadTrusted(*identityPath);if err!=nil{fail(err)}
	packs,err:=devicepack.LoadVerifiedRuntime(*packDir,*artifactRoot,*packTrust,*handlerRoot,identity.BoardID,time.Second);if err!=nil{fail(err)}
	ctx:=kingupdate.TargetContext{
		Profile:*profile, Arch:runtime.GOARCH, Channel:identity.UpdateChannel,
		BoardID:identity.BoardID, DeviceClass:identity.Class, DevicePackIDs:packs.PackIDs(),
		AttestationMode:identity.Attestation,
	}
	if err:=kingupdate.CheckTargetCompatibility(e.Manifest,ctx);err!=nil{fail(err)}
	enc:=json.NewEncoder(os.Stdout);enc.SetIndent("","  ")
	_ = enc.Encode(map[string]any{
		"verified":true,"product":e.Manifest.Product,"version":e.Manifest.Version,
		"artifact":e.Manifest.Artifact,"device_id":identity.DeviceID,"board_id":identity.BoardID,
		"device_class":identity.Class,"channel":identity.UpdateChannel,"arch":runtime.GOARCH,
		"device_packs":packs.PackIDs(),"attestation":identity.Attestation,
	})
}

func tufFetch(args []string) {
	fs:=flag.NewFlagSet("tuf-fetch",flag.ExitOnError)
	metadataURL:=fs.String("metadata-url","","TUF metadata HTTPS base URL")
	targetsURL:=fs.String("targets-url","","TUF targets HTTPS base URL")
	root:=fs.String("trusted-root","/usr/share/kingai/trust/tuf/root.json","out-of-band trusted TUF root.json")
	state:=fs.String("state-dir","/var/lib/kingai-state/tuf","local trusted TUF state directory")
	target:=fs.String("target","","trusted target name")
	_ = fs.Parse(args)
	path,err:=tufclient.Fetch(tufclient.Config{MetadataURL:*metadataURL,TargetsURL:*targetsURL,TrustedRootPath:*root,StateDir:*state},*target)
	if err!=nil{fail(err)}
	enc:=json.NewEncoder(os.Stdout);enc.SetIndent("","  ");_ = enc.Encode(map[string]string{"target":*target,"path":path,"trust":"TUF-v2-pinned-root"})
}

func stage(args []string) {
	fs:=flag.NewFlagSet("stage",flag.ExitOnError)
	target:=fs.String("target-disk","","exact installed KINGAI target disk")
	source:=fs.String("source-root","","verified target KINGAI root filesystem")
	stateKey:=fs.String("state-key","","LUKS STATE key file")
	targetVersion:=fs.String("target-version","","target VERSION_ID")
	confirm:=fs.String("confirm","","must equal UPDATE:<exact target disk>")
	_ = fs.Parse(args)
	jsonOut:=os.Stdout;os.Stdout=os.Stderr
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

func usage(){fmt.Fprintln(os.Stderr,"usage: kingai-update verify <envelope.json> <public-key.b64> <artifact> | edge-verify --envelope <envelope.json> --public-key <key.b64> --artifact <artifact> [--identity /etc/kingai/device.json] | tuf-fetch --metadata-url https://... --targets-url https://... --trusted-root <root.json> --target <name> | stage --target-disk <disk> --source-root <rootfs> --state-key <file> --target-version <version> --confirm UPDATE:<disk> | boot-health [--state <slots.json>]");os.Exit(2)}
func fail(err error){ fmt.Fprintln(os.Stderr,"update failed:",err); os.Exit(1) }
