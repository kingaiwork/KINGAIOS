package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kingaiwork/KINGAIOS/internal/devicepack"
)

var keyIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,63}$`)

func main(){
	fs:=flag.NewFlagSet("device-pack-release",flag.ExitOnError)
	template:=fs.String("template","","integration template JSON")
	artifactDir:=fs.String("artifact-dir","","directory containing exact artifact basenames")
	version:=fs.String("version","","release version")
	keyID:=fs.String("key-id","","Device Pack release key id")
	privateKey:=fs.String("private-key","","base64 Ed25519 private key file")
	manifestOut:=fs.String("manifest-out","","final manifest path; .sig is written beside it")
	reviewed:=fs.Bool("redistribution-reviewed",false,"confirm that firmware/driver redistribution review is complete")
	reviewRef:=fs.String("review-ref","","immutable ticket, change or review reference documenting redistribution approval")
	notes:=fs.String("notes","KINGAI Device Pack release manifest.","release security notes")
	_ = fs.Parse(os.Args[1:])
	if *template==""||*artifactDir==""||*version==""||*keyID==""||*privateKey==""||*manifestOut==""{fail(errors.New("--template, --artifact-dir, --version, --key-id, --private-key and --manifest-out are required"))}
	if !*reviewed || !validReviewRef(*reviewRef){fail(errors.New("release requires --redistribution-reviewed and a valid --review-ref"))}
	if !keyIDPattern.MatchString(*keyID)||strings.Contains(*keyID,".."){fail(errors.New("invalid key id"))}

	manifest,err:=devicepack.LoadIntegrationTemplate(*template);if err!=nil{fail(err)}
	manifest.Version=strings.TrimSpace(*version)
	manifest.Security.SignedManifest=true
	manifest.Security.RedistributionReviewed=true
	manifest.Security.Notes=strings.TrimSpace(*notes)+" redistribution_review_ref="+strings.TrimSpace(*reviewRef)
	for i:=range manifest.Artifacts{
		artifact:=&manifest.Artifacts[i]
		if strings.ContainsAny(artifact.Name,"/\\")||artifact.Name=="."||artifact.Name==".."{fail(fmt.Errorf("unsafe artifact name %q",artifact.Name))}
		if artifact.Kind=="firmware"||artifact.Kind=="bootloader"||artifact.Kind=="driver"{
			upper:=strings.ToUpper(artifact.License+" "+artifact.Source)
			if artifact.License==""||artifact.Source==""||strings.Contains(upper,"REVIEW_REQUIRED")||strings.Contains(upper,"REPLACE_"){
				fail(fmt.Errorf("artifact %q still has unreviewed license/source metadata",artifact.Name))
			}
		}
		path:=filepath.Join(*artifactDir,artifact.Name)
		if filepath.Dir(path)!=filepath.Clean(*artifactDir){fail(fmt.Errorf("artifact %q escaped artifact directory",artifact.Name))}
		size,digest,err:=hashArtifact(path);if err!=nil{fail(err)}
		artifact.SizeBytes=size;artifact.SHA256=digest
	}
	if err:=manifest.Validate();err!=nil{fail(err)}

	key,err:=loadPrivateKey(*privateKey);if err!=nil{fail(err)}
	manifestBytes,err:=json.MarshalIndent(manifest,"","  ");if err!=nil{fail(err)}
	manifestBytes=append(manifestBytes,'\n')
	sig:=ed25519.Sign(key,manifestBytes)
	envelope:=devicepack.DetachedSignature{Schema:1,KeyID:*keyID,Signature:base64.StdEncoding.EncodeToString(sig)}
	sigBytes,err:=json.MarshalIndent(envelope,"","  ");if err!=nil{fail(err)}
	sigBytes=append(sigBytes,'\n')

	if err:=writeExclusive(*manifestOut,manifestBytes,0o644);err!=nil{fail(err)}
	sigPath:=*manifestOut+".sig"
	if err:=writeExclusive(sigPath,sigBytes,0o644);err!=nil{_ = os.Remove(*manifestOut);fail(err)}
	sum:=sha256.Sum256(manifestBytes)
	fmt.Printf("released Device Pack %s version=%s manifest_sha256=%s signature=%s review_ref=%s\n",manifest.ID,manifest.Version,hex.EncodeToString(sum[:]),sigPath,strings.TrimSpace(*reviewRef))
}

func validReviewRef(value string)bool{
	value=strings.TrimSpace(value)
	if value==""||len(value)>256{return false}
	for _,r:=range value{if r<0x20||r==0x7f{return false}}
	return !strings.Contains(value,"..")
}

func hashArtifact(path string)(int64,string,error){
	info,err:=os.Lstat(path);if err!=nil{return 0,"",err}
	if info.Mode()&os.ModeSymlink!=0||!info.Mode().IsRegular(){return 0,"",fmt.Errorf("artifact %s must be a regular non-symlink file",filepath.Base(path))}
	f,err:=os.Open(path);if err!=nil{return 0,"",err};defer f.Close()
	h:=sha256.New();n,err:=io.Copy(h,f);if err!=nil{return 0,"",err}
	if n!=info.Size(){return 0,"",errors.New("artifact size changed while hashing")}
	return n,hex.EncodeToString(h.Sum(nil)),nil
}

func loadPrivateKey(path string)(ed25519.PrivateKey,error){
	info,err:=os.Lstat(path);if err!=nil{return nil,err}
	if info.Mode()&os.ModeSymlink!=0||!info.Mode().IsRegular(){return nil,errors.New("private key must be a regular non-symlink file")}
	if info.Mode().Perm()&0o077!=0{return nil,errors.New("private key must be mode 0600 or stricter")}
	raw,err:=os.ReadFile(path);if err!=nil{return nil,err}
	decoded,err:=base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)));if err!=nil{return nil,err}
	if len(decoded)!=ed25519.PrivateKeySize{return nil,errors.New("invalid Ed25519 private key size")}
	return ed25519.PrivateKey(decoded),nil
}

func writeExclusive(path string,data []byte,mode os.FileMode)error{
	if !filepath.IsAbs(path){return errors.New("output path must be absolute")}
	f,err:=os.OpenFile(path,os.O_WRONLY|os.O_CREATE|os.O_EXCL,mode);if err!=nil{return err}
	ok:=false
	defer func(){if !ok{_ = os.Remove(path)}}()
	if _,err:=f.Write(data);err!=nil{_ = f.Close();return err}
	if err:=f.Sync();err!=nil{_ = f.Close();return err}
	if err:=f.Close();err!=nil{return err}
	ok=true
	return nil
}

func fail(err error){fmt.Fprintln(os.Stderr,"device-pack release failed:",err);os.Exit(1)}
