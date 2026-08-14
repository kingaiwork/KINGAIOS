package edgehandler

import (
	"encoding/json"
	"testing"
)

func TestAuthorizeExactCapabilityAndResource(t *testing.T) {
	caps:=map[string][]string{"device.gpio.read":{"gpio:17"}}
	if err:=authorize(caps,Request{Agent:"system-ops",Capability:"device.gpio.read",Target:"gpio:17",Arguments:json.RawMessage(`{"sample":true}`)});err!=nil{t.Fatal(err)}
	if err:=authorize(caps,Request{Agent:"system-ops",Capability:"device.gpio.read",Target:"gpio:18"});err==nil{t.Fatal("undeclared resource must be rejected")}
	if err:=authorize(caps,Request{Agent:"system-ops",Capability:"device.power.control",Target:"power:system"});err==nil{t.Fatal("undeclared capability must be rejected")}
}

func TestAuthorizeRejectsUnsafeRequests(t *testing.T) {
	caps:=map[string][]string{"device.gpio.read":{"gpio:17"}}
	tests:=[]Request{
		{Agent:"",Capability:"device.gpio.read",Target:"gpio:17"},
		{Agent:"system-ops",Capability:"device.*",Target:"gpio:17"},
		{Agent:"system-ops",Capability:"device.gpio.read",Target:"gpio:17\n"},
		{Agent:"system-ops",Capability:"device.gpio.read",Target:"gpio:17",Arguments:json.RawMessage(`{`)},
	}
	for _,req:=range tests{if err:=authorize(caps,req);err==nil{t.Fatalf("unsafe request accepted: %+v",req)}}
}

func TestValidateConfig(t *testing.T) {
	cfg:=Config{SocketPath:"/run/kingai-device/gpio-read.sock",SocketOwner:1000,Capabilities:map[string][]string{"device.gpio.read":{"gpio:17"}}}
	if err:=validateConfig(cfg);err!=nil{t.Fatal(err)}
	cfg.SocketPath="/tmp/gpio.sock"
	if err:=validateConfig(cfg);err==nil{t.Fatal("socket outside handler namespace must be rejected")}
}
