package model

import "testing"

func TestPrivateSelectsLocal(t *testing.T) {
	c := []Candidate{
		{ID:"cloud", Available:true, Capabilities:[]string{"chat"}, Priority:10},
		{ID:"local", Local:true, Available:true, Capabilities:[]string{"chat"}, Priority:1},
	}
	got, err := Select(Request{Capability:"chat", Private:true}, c)
	if err != nil || got.ID != "local" { t.Fatalf("expected local, got %#v err=%v", got, err) }
}

func TestOfflineRejectsCloudOnly(t *testing.T) {
	_, err := Select(Request{Offline:true}, []Candidate{{ID:"cloud", Available:true, Capabilities:[]string{"*"}}})
	if err == nil { t.Fatal("offline mode must reject cloud-only candidates") }
}
