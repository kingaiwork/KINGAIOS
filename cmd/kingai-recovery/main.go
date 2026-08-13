package main

import (
	"flag"
	"fmt"
	"os"

	kingrecovery "github.com/kingaiwork/KINGAIOS/internal/recovery"
)

func main(){
	if len(os.Args)<2{usage()}
	switch os.Args[1]{
	case "inspect": runInspect(os.Args[2:])
	case "rollback": runRollback(os.Args[2:])
	case "repair-boot": runRepair(os.Args[2:])
	default: usage()
	}
}
func common(name string,args []string)(kingrecovery.Options,*flag.FlagSet){fs:=flag.NewFlagSet(name,flag.ExitOnError);target:=fs.String("target-disk","","exact installed KINGAI disk");key:=fs.String("state-key","","STATE LUKS key");confirm:=fs.String("confirm","","exact destructive confirmation");_ = fs.Parse(args);return kingrecovery.Options{TargetDisk:*target,StateKey:*key,Confirmation:*confirm},fs}
func runInspect(args []string){o,_:=common("inspect",args);r,e:=kingrecovery.Inspect(o);if e!=nil{fail(e)};fmt.Println(kingrecovery.Marshal(r))}
func runRollback(args []string){o,_:=common("rollback",args);r,e:=kingrecovery.Rollback(o);if e!=nil{fail(e)};fmt.Println(kingrecovery.Marshal(r))}
func runRepair(args []string){o,_:=common("repair-boot",args);r,e:=kingrecovery.RepairBoot(o);if e!=nil{fail(e)};fmt.Println(kingrecovery.Marshal(r))}
func usage(){fmt.Fprintln(os.Stderr,"usage: kingai-recovery inspect --target-disk <disk> --state-key <key> | rollback ... --confirm ROLLBACK:<disk> | repair-boot ... --confirm REPAIR:<disk>");os.Exit(2)}
func fail(e error){fmt.Fprintln(os.Stderr,"recovery failed:",e);os.Exit(1)}
