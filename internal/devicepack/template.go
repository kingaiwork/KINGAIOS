package devicepack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// LoadIntegrationTemplate validates a board integration template without
// allowing it to masquerade as a release Device Pack. Templates must remain
// explicitly unsigned and redistribution-unreviewed, with placeholder artifact
// hashes/sizes. A release pack must instead pass Load + detached verification.
func LoadIntegrationTemplate(path string) (Manifest,error) {
	b,err:=os.ReadFile(path);if err!=nil{return Manifest{},err}
	if len(b)==0 || len(b)>maxManifestBytes{return Manifest{},errors.New("invalid template size")}
	dec:=json.NewDecoder(strings.NewReader(string(b)));dec.DisallowUnknownFields()
	var m Manifest
	if err:=dec.Decode(&m);err!=nil{return Manifest{},fmt.Errorf("decode Device Pack template: %w",err)}
	var trailing any
	if err:=dec.Decode(&trailing);!errors.Is(err,io.EOF){return Manifest{},errors.New("Device Pack template must contain exactly one JSON object")}
	if m.Security.SignedManifest || m.Security.RedistributionReviewed{return Manifest{},errors.New("integration template must not claim release signing or redistribution review")}
	if !strings.Contains(strings.ToLower(m.Security.Notes),"template"){return Manifest{},errors.New("integration template security notes must explicitly identify template status")}
	for _,artifact:=range m.Artifacts{
		if artifact.SizeBytes!=0 || artifact.SHA256!="0000000000000000000000000000000000000000000000000000000000000000"{
			return Manifest{},fmt.Errorf("template artifact %q must use explicit zero-size/zero-hash placeholder",artifact.Name)
		}
	}
	candidate:=m
	candidate.Security.SignedManifest=true
	candidate.Security.RedistributionReviewed=true
	if err:=candidate.Validate();err!=nil{return Manifest{},err}
	return m,nil
}
