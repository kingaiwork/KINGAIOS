package devicepackadmin

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kingaiwork/KINGAIOS/internal/devicepack"
)

func adminManifest() devicepack.Manifest{
	return devicepack.Manifest{Schema:devicepack.SchemaVersion,ID:"kingai.test-board",Name:"Test",Version:"1.0.0",Arch:runtime.GOARCH,Vendor:"KINGAI",BoardIDs:[]string{"board-a"},Boot:devicepack.Boot{Method:"uefi"},Artifacts:[]devicepack.Artifact{{Name:"config.bin",Kind:"config",SHA256:"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",SizeBytes:1}},Security:devicepack.Security{SignedManifest:true,RedistributionReviewed:true}}
}

func TestBoardCompatibility(t *testing.T){
	m:=adminManifest()
	if err:=boardCompatible(m,"board-a");err!=nil{t.Fatal(err)}
	if err:=boardCompatible(m,"board-b");err==nil{t.Fatal("wrong board must be rejected")}
	if err:=boardCompatible(m,"");err==nil{t.Fatal("missing board id must be rejected")}
}

func TestCopyAndVerifyArtifact(t *testing.T){
	root:=t.TempDir();source:=filepath.Join(root,"source.bin");dest:=filepath.Join(root,"dest.bin")
	data:=[]byte("edge-artifact");if err:=os.WriteFile(source,data,0o644);err!=nil{t.Fatal(err)}
	sum:=sha256.Sum256(data)
	artifact:=devicepack.Artifact{Name:"config.bin",Kind:"config",SHA256:hex.EncodeToString(sum[:]),SizeBytes:int64(len(data))}
	if err:=copyAndVerifyArtifact(source,dest,artifact);err!=nil{t.Fatal(err)}
	got,err:=os.ReadFile(dest);if err!=nil{t.Fatal(err)};if string(got)!=string(data){t.Fatal("artifact copy changed data")}
}

func TestCopyRejectsArtifactHashMismatch(t *testing.T){
	root:=t.TempDir();source:=filepath.Join(root,"source.bin");dest:=filepath.Join(root,"dest.bin")
	data:=[]byte("edge-artifact");if err:=os.WriteFile(source,data,0o644);err!=nil{t.Fatal(err)}
	artifact:=devicepack.Artifact{Name:"config.bin",Kind:"config",SHA256:"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",SizeBytes:int64(len(data))}
	if err:=copyAndVerifyArtifact(source,dest,artifact);err==nil{t.Fatal("hash mismatch must be rejected")}
}

func TestCleanAbsolute(t *testing.T){
	if err:=cleanAbsolute("/etc/kingai/device-packs");err!=nil{t.Fatal(err)}
	for _,path:=range []string{"relative","/etc/../tmp","/tmp/bad\npath"}{if err:=cleanAbsolute(path);err==nil{t.Fatalf("unsafe path accepted: %q",path)}}
}
