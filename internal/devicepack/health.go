package devicepack

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Health verifies that every handler referenced by an installed Device Pack is
// both a trusted private Unix socket and a live service. With no installed
// capabilities the generic Edge runtime is healthy and can remain local-only.
func (r *Runtime) Health(ctx context.Context) error {
	if r == nil { return errors.New("device runtime is unavailable") }
	handlers := make(map[string]struct{})
	for _, capability := range r.capabilities { handlers[capability.Handler] = struct{}{} }
	if len(handlers) == 0 { return nil }

	ids := make([]string,0,len(handlers))
	for id := range handlers { ids=append(ids,id) }
	sort.Strings(ids)
	probeTimeout := r.timeout
	if probeTimeout <= 0 || probeTimeout > 3*time.Second { probeTimeout=3*time.Second }
	probeCtx,cancel := context.WithTimeout(ctx,probeTimeout)
	defer cancel()

	errCh:=make(chan error,len(ids))
	var wg sync.WaitGroup
	for _,id:=range ids {
		id:=id
		wg.Add(1)
		go func(){ defer wg.Done(); if err:=r.probeHandler(probeCtx,id);err!=nil{errCh<-err} }()
	}
	wg.Wait();close(errCh)
	for err:=range errCh { if err!=nil{return err} }
	return nil
}

func (r *Runtime) probeHandler(ctx context.Context,handler string) error {
	socketPath:=filepath.Join(r.socketRoot,handler+".sock")
	if filepath.Dir(socketPath)!=r.socketRoot{return fmt.Errorf("%w: handler path escaped socket root",ErrHandlerUnavailable)}
	if err:=validateHandlerSocket(socketPath);err!=nil{return err}
	transport:=&http.Transport{DialContext:func(ctx context.Context,_,_ string)(net.Conn,error){return (&net.Dialer{}).DialContext(ctx,"unix",socketPath)}}
	defer transport.CloseIdleConnections()
	client:=&http.Client{Transport:transport}
	req,err:=http.NewRequestWithContext(ctx,http.MethodGet,"http://kingai-device/healthz",nil);if err!=nil{return err}
	resp,err:=client.Do(req);if err!=nil{return fmt.Errorf("%w: handler %s health request failed: %v",ErrHandlerUnavailable,handler,err)}
	defer resp.Body.Close()
	if resp.StatusCode>=300{return fmt.Errorf("%w: handler %s health returned %s",ErrHandlerUnavailable,handler,resp.Status)}
	return nil
}
