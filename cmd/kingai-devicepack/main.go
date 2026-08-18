package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kingaiwork/KINGAIOS/internal/deviceidentity"
	"github.com/kingaiwork/KINGAIOS/internal/devicepackadmin"
)

var version = "0.1.0-dev"

func main(){
	if len(os.Args)<2{usage()}
	switch os.Args[1]{
	case "install":install(os.Args[2:])
	case "list":list(os.Args[2:])
	case "deactivate":deactivate(os.Args[2:])
	case "version":fmt.Println("KINGAI Device Pack Admin",version)
	default:usage()
	}
}

func commonPaths(fs *flag.FlagSet)(*string,*string,*string,*string,*string){
	manifestDir:=fs.String("manifest-dir","/etc/kingai/device-packs","active Device Pack manifest directory")
	artifactRoot:=fs.String("artifact-root","/usr/lib/kingai/device-packs","active Device Pack artifact root")
	trustDir:=fs.String("trust-dir","/etc/kingai/trust/device-pack-keys","root-provisioned Device Pack public-key directory")
	lockPath:=fs.String("lock","/run/lock/kingai-device-pack.lock","lifecycle lock file")
	adminAudit:=fs.String("admin-audit","/var/log/kingai/device-pack-admin.jsonl","root administrative audit log")
	return manifestDir,artifactRoot,trustDir,lockPath,adminAudit
}

func install(args []string){
	fs:=flag.NewFlagSet("install",flag.ExitOnError)
	manifest:=fs.String("manifest","","signed release manifest")
	signature:=fs.String("signature","","detached manifest signature; defaults to <manifest>.sig")
	artifacts:=fs.String("artifacts","","directory containing exact artifact basenames")
	identityPath:=fs.String("identity","/etc/kingai/device.json","trusted Edge device identity")
	confirm:=fs.String("confirm","","must equal INSTALL:<pack-id>")
	manifestDir,artifactRoot,trustDir,lockPath,adminAudit:=commonPaths(fs)
	_ = fs.Parse(args)
	if *manifest==""||*artifacts==""||*confirm==""{fail(fmt.Errorf("--manifest, --artifacts and --confirm are required"))}
	if *signature==""{*signature=*manifest+".sig"}
	identity,err:=deviceidentity.LoadTrusted(*identityPath);if err!=nil{fail(fmt.Errorf("trusted device identity is required before Device Pack installation: %w",err))}
	paths:=devicepackadmin.Paths{ManifestDir:*manifestDir,ArtifactRoot:*artifactRoot,TrustDir:*trustDir,LockPath:*lockPath,AdminAudit:*adminAudit}
	result,err:=devicepackadmin.Install(paths,devicepackadmin.InstallOptions{ManifestPath:*manifest,SignaturePath:*signature,ArtifactDir:*artifacts,BoardID:identity.BoardID,Confirmation:*confirm});if err!=nil{fail(err)}
	printJSON(result)
}

func list(args []string){
	fs:=flag.NewFlagSet("list",flag.ExitOnError)
	manifestDir,artifactRoot,trustDir,lockPath,adminAudit:=commonPaths(fs)
	_ = fs.Parse(args)
	paths:=devicepackadmin.Paths{ManifestDir:*manifestDir,ArtifactRoot:*artifactRoot,TrustDir:*trustDir,LockPath:*lockPath,AdminAudit:*adminAudit}
	items,err:=devicepackadmin.List(paths);if err!=nil{fail(err)}
	printJSON(map[string]any{"device_packs":items,"count":len(items)})
}

func deactivate(args []string){
	fs:=flag.NewFlagSet("deactivate",flag.ExitOnError)
	packID:=fs.String("pack","","canonical Device Pack id")
	confirm:=fs.String("confirm","","must equal DEACTIVATE:<pack-id>")
	manifestDir,artifactRoot,trustDir,lockPath,adminAudit:=commonPaths(fs)
	_ = fs.Parse(args)
	if *packID==""||*confirm==""{fail(fmt.Errorf("--pack and --confirm are required"))}
	paths:=devicepackadmin.Paths{ManifestDir:*manifestDir,ArtifactRoot:*artifactRoot,TrustDir:*trustDir,LockPath:*lockPath,AdminAudit:*adminAudit}
	result,err:=devicepackadmin.Deactivate(paths,*packID,*confirm);if err!=nil{fail(err)}
	printJSON(result)
}

func printJSON(value any){b,err:=devicepackadmin.MarshalIndented(value);if err!=nil{fail(err)};_,_ = os.Stdout.Write(b)}
func fail(err error){fmt.Fprintln(os.Stderr,"Device Pack administration failed:",err);os.Exit(1)}
func usage(){fmt.Fprintln(os.Stderr,"usage: kingai-devicepack install --manifest <pack.json> --artifacts <dir> --confirm INSTALL:<pack-id> | list | deactivate --pack <pack-id> --confirm DEACTIVATE:<pack-id> | version");os.Exit(2)}
