package mcp

import "sync"

var clients = struct{
	sync.Mutex
	m map[chan string]bool
}{m: make(map[chan string]bool)}

func broadcast(msg string){
	clients.Lock(); defer clients.Unlock()
	for ch:=range clients.m{
		select{case ch<-msg:default:{}}
	}
}
