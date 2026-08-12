package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/desktop"
	"github.com/kingaiwork/KINGAIOS/internal/policy"
)

var version="0.1.0-dev"

func usage(){fmt.Print(`KINGAI OS CLI

Usage:
  kingai version
  kingai status [--json]
  kingai doctor
  kingai policy check <capability> [target]
  kingai desktop list
  kingai desktop show
  kingai desktop set <kingai-intelligence|kingai-flow|kingai-classic>
  kingai desktop apply
`)}

func main(){if len(os.Args)<2{usage();return};switch os.Args[1]{case"version":fmt.Printf("KINGAI OS %s\n",version);case"status":status(os.Args[2:]);case"doctor":doctor();case"policy":policyCmd(os.Args[2:]);case"desktop":desktopCmd(os.Args[2:]);default:usage();os.Exit(2)}}

func status(args []string){var remote map[string]any;if err:=daemonJSON(http.MethodGet,"/v1/status",nil,&remote);err==nil{if len(args)>0&&strings.EqualFold(args[0],"--json"){_=json.NewEncoder(os.Stdout).Encode(remote);return};fmt.Printf("System:       %v\nVersion:      %v\nArchitecture: %v\nPolicy:       %v\n",remote["name"],remote["version"],remote["architecture"],remote["policy"]);return};fallback:=map[string]any{"name":"KINGAI OS","version":version,"channel":"dev","platform":runtime.GOOS+"/"+runtime.GOARCH,"daemon":"offline"};if len(args)>0&&strings.EqualFold(args[0],"--json"){_=json.NewEncoder(os.Stdout).Encode(fallback);return};fmt.Printf("System:  KINGAI OS\nVersion: %s\nPlatform: %s/%s\nDaemon:  offline\n",version,runtime.GOOS,runtime.GOARCH)}

func doctor(){fmt.Println("KINGAI Doctor");if err:=daemonJSON(http.MethodGet,"/healthz",nil,&map[string]any{});err!=nil{fmt.Printf("[warn] kingaid: %v\n",err)}else{fmt.Println("[ok] kingaid")};checks:=[]struct{name,path string}{{"policy","/etc/kingai/policy.json"},{"system config","/etc/kingai/system.json"},{"agent config","/etc/kingai/agents.json"},{"model config","/etc/kingai/models.json"}};for _,c:=range checks{if _,err:=os.Stat(c.path);err==nil{fmt.Printf("[ok] %s\n",c.name)}else{fmt.Printf("[info] %s not installed at %s\n",c.name,c.path)}}}

func policyCmd(args []string){if len(args)<2||args[0]!="check"{usage();os.Exit(2)};req:=policy.Request{Agent:"main",Capability:args[1]};if len(args)>2{req.Target=args[2]};var out policy.Result;if err:=daemonJSON(http.MethodPost,"/v1/policy/evaluate",req,&out);err!=nil{out=policy.Default().Evaluate(req)};_=json.NewEncoder(os.Stdout).Encode(out);if !out.Allowed{os.Exit(3)}}

func desktopCmd(args []string){if len(args)<1{usage();os.Exit(2)};switch args[0]{case"list":for _,e:=range desktop.List(){fmt.Printf("%s\t%s\n",e.ID,e.Name)};case"show":v,err:=desktop.Current();if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)};if v==""{fmt.Println("unselected")}else{fmt.Println(v)};case"set":if len(args)!=2{usage();os.Exit(2)};if err:=desktop.Set(args[1],true);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)};fmt.Printf("desktop experience set to %s\n",args[1]);case"apply":if err:=desktop.ApplyCurrent();err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)};fmt.Println("desktop experience applied");default:usage();os.Exit(2)}}

func daemonJSON(method,path string,body any,out any)error{ctx,cancel:=context.WithTimeout(context.Background(),2*time.Second);defer cancel();tr:=&http.Transport{DialContext:func(ctx context.Context,_,_ string)(net.Conn,error){return(&net.Dialer{}).DialContext(ctx,"unix","/run/kingai/kingaid.sock")}};defer tr.CloseIdleConnections();client:=&http.Client{Transport:tr,Timeout:2*time.Second};var reader io.Reader;if body!=nil{b,err:=json.Marshal(body);if err!=nil{return err};reader=bytes.NewReader(b)};req,err:=http.NewRequestWithContext(ctx,method,"http://kingai"+path,reader);if err!=nil{return err};if body!=nil{req.Header.Set("Content-Type","application/json")};resp,err:=client.Do(req);if err!=nil{return err};defer resp.Body.Close();if resp.StatusCode>=300{return fmt.Errorf("daemon returned %s",resp.Status)};return json.NewDecoder(resp.Body).Decode(out)}
