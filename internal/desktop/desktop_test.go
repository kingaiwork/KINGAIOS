package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadQtSettingsExperience(t *testing.T){
	p:=filepath.Join(t.TempDir(),"desktop.ini")
	if err:=os.WriteFile(p,[]byte("[General]\nexperience=kingai-flow\n"),0o600); err!=nil{t.Fatal(err)}
	got,err:=readExperience(p); if err!=nil{t.Fatal(err)}; if got!="kingai-flow"{t.Fatalf("got %q",got)}
}

func TestReadRejectsUnknownExperience(t *testing.T){
	p:=filepath.Join(t.TempDir(),"desktop.ini")
	_ = os.WriteFile(p,[]byte("[General]\nexperience=third-party\n"),0o600)
	if _,err:=readExperience(p); err==nil{t.Fatal("unknown experience must fail closed")}
}
