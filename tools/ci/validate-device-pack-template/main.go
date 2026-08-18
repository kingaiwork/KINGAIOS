package main

import (
	"fmt"
	"os"

	"github.com/kingaiwork/KINGAIOS/internal/devicepack"
)

func main(){
	if len(os.Args)<2{fmt.Fprintln(os.Stderr,"usage: validate-device-pack-template <template.json> [...]");os.Exit(2)}
	failed:=false
	for _,path:=range os.Args[1:]{
		m,err:=devicepack.LoadIntegrationTemplate(path)
		if err!=nil{fmt.Fprintf(os.Stderr,"%s: %v\n",path,err);failed=true;continue}
		fmt.Printf("valid integration template: %s (%s/%s) capabilities=%d\n",m.ID,m.Arch,m.Boot.Method,len(m.Capabilities))
	}
	if failed{os.Exit(1)}
}
