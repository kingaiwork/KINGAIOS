package main

import "testing"

func TestValidReviewRef(t *testing.T){
	for _,value:=range []string{"SEC-2026-0042","licenses/jetson-bsp-review-17","change-1234"}{if !validReviewRef(value){t.Fatalf("expected valid review reference: %q",value)}}
	for _,value:=range []string{"","../escape","bad\nref"}{if validReviewRef(value){t.Fatalf("expected invalid review reference: %q",value)}}
}
