package installer

import "testing"

func TestPlanRefusesMountedDisk(t *testing.T){
	devs:=[]Device{{Name:"vda",Path:"/dev/vda",Type:"disk",Size:64*GiB,Children:[]Device{{Name:"vda1",Path:"/dev/vda1",Type:"part",Mountpoints:[]string{"/"}}}}}
	if _,err:=BuildPlan(devs,"/dev/vda","server");err==nil{t.Fatal("mounted target must be rejected")}
}

func TestDesktopABPlan(t *testing.T){
	devs:=[]Device{{Name:"vdb",Path:"/dev/vdb",Type:"disk",Size:64*GiB}}
	p,err:=BuildPlan(devs,"/dev/vdb","desktop");if err!=nil{t.Fatal(err)}
	if p.Executable{t.Fatal("developer planner must never mark plan executable")}
	if len(p.Partitions)!=4{t.Fatalf("expected four partitions, got %d",len(p.Partitions))}
	if p.Partitions[1].SizeBytes!=p.Partitions[2].SizeBytes{t.Fatal("A/B root slots must match")}
}

func TestSmallDiskRejected(t *testing.T){
	devs:=[]Device{{Name:"vdb",Path:"/dev/vdb",Type:"disk",Size:12*GiB}}
	if _,err:=BuildPlan(devs,"/dev/vdb","desktop");err==nil{t.Fatal("undersized disk must be rejected")}
}
