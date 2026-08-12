package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kingaiwork/KINGAIOS/internal/installer"
)

var version="0.1.0-dev"

func main(){
	if len(os.Args)<2{usage();os.Exit(2)}
	switch os.Args[1] {
	case "list": list()
	case "plan": plan(os.Args[2:])
	case "version": fmt.Printf("KINGAI Installer Planner %s\n",version)
	default: usage();os.Exit(2)
	}
}

func usage(){fmt.Fprintln(os.Stderr,"usage: kingai-installer <list|plan|version>\n  kingai-installer plan --target /dev/DEVICE --profile server|desktop|iot")}

func list(){devs,err:=installer.Discover();if err!=nil{fail(err)};enc:=json.NewEncoder(os.Stdout);enc.SetIndent("","  ");_ = enc.Encode(installer.CandidateDisks(devs))}

func plan(args []string){fs:=flag.NewFlagSet("plan",flag.ExitOnError);target:=fs.String("target","","target disk");profile:=fs.String("profile","server","distribution profile");_ = fs.Parse(args);devs,err:=installer.Discover();if err!=nil{fail(err)};p,err:=installer.BuildPlan(devs,*target,*profile);if err!=nil{fail(err)};enc:=json.NewEncoder(os.Stdout);enc.SetIndent("","  ");_ = enc.Encode(p)}

func fail(err error){fmt.Fprintln(os.Stderr,"installer preflight failed:",err);os.Exit(1)}
