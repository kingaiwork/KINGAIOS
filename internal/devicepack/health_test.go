package devicepack

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRuntimeHealthRequiresDeclaredHandler(t *testing.T) {
	root:=t.TempDir()
	rt,err:=NewRuntime([]Manifest{runtimeManifest()},root,"",time.Second)
	if err!=nil{t.Fatal(err)}
	if err:=rt.Health(context.Background());err==nil{t.Fatal("missing declared handler must make runtime unhealthy")}
}

func TestRuntimeHealthAcceptsHealthyPrivateHandler(t *testing.T) {
	root:=t.TempDir()
	socket:=filepath.Join(root,"gpio-read.sock")
	listener,err:=net.Listen("unix",socket);if err!=nil{t.Fatal(err)}
	defer listener.Close()
	if err:=os.Chmod(socket,0o600);err!=nil{t.Fatal(err)}
	server:=&http.Server{Handler:http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.Method==http.MethodGet&&r.URL.Path=="/healthz"{w.WriteHeader(http.StatusOK);return}
		http.Error(w,"not found",http.StatusNotFound)
	})}
	go func(){_ = server.Serve(listener)}()
	defer server.Shutdown(context.Background())

	rt,err:=NewRuntime([]Manifest{runtimeManifest()},root,"",time.Second);if err!=nil{t.Fatal(err)}
	if err:=rt.Health(context.Background());err!=nil{t.Fatalf("healthy handler should satisfy runtime readiness: %v",err)}
}

func TestRuntimeHealthRejectsHandlerError(t *testing.T) {
	root:=t.TempDir()
	socket:=filepath.Join(root,"gpio-read.sock")
	listener,err:=net.Listen("unix",socket);if err!=nil{t.Fatal(err)}
	defer listener.Close()
	if err:=os.Chmod(socket,0o600);err!=nil{t.Fatal(err)}
	server:=&http.Server{Handler:http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){http.Error(w,"degraded",http.StatusServiceUnavailable)})}
	go func(){_ = server.Serve(listener)}()
	defer server.Shutdown(context.Background())

	rt,err:=NewRuntime([]Manifest{runtimeManifest()},root,"",time.Second);if err!=nil{t.Fatal(err)}
	if err:=rt.Health(context.Background());err==nil{t.Fatal("unhealthy handler must make runtime unhealthy")}
}
