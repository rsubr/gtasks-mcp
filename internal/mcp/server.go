package mcp

import (
	"encoding/json"
	"net/http"

	"gtasks-mcp/internal/tasks"
)

type Server struct{ tasks *tasks.Service }

func NewServer(tasks *tasks.Service)*Server{ return &Server{tasks:tasks} }

func (s *Server) Start(addr string) error {
	http.HandleFunc("/rpc", s.handleRPC)
	http.HandleFunc("/events", s.handleSSE)
	http.HandleFunc("/manifest", s.handleManifest)
	return http.ListenAndServe(addr,nil)
}

func (s *Server) handleSSE(w http.ResponseWriter,r *http.Request){
	w.Header().Set("Content-Type","text/event-stream")
	flusher,_:=w.(http.Flusher)

	ch:=make(chan string)
	clients.Lock(); clients.m[ch]=true; clients.Unlock()
	defer func(){clients.Lock();delete(clients.m,ch);clients.Unlock()}()

	for{ msg:=<-ch; w.Write([]byte("data: "+msg+"\n\n")); flusher.Flush() }
}

type Req struct{
	JSONRPC string `json:"jsonrpc"`
	ID interface{} `json:"id"`
	Method string `json:"method"`
	Params json.RawMessage `json:"params"`
}

type Res struct{
	JSONRPC string `json:"jsonrpc"`
	ID interface{} `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error interface{} `json:"error,omitempty"`
}

func errResp(id interface{},code int,msg string)Res{
	return Res{"2.0",id,nil,map[string]interface{}{"code":code,"message":msg}}
}

func (s *Server) handleRPC(w http.ResponseWriter,r *http.Request){
	var req Req; json.NewDecoder(r.Body).Decode(&req)

	switch req.Method{

	case "initialize":
		json.NewEncoder(w).Encode(Res{"2.0",req.ID,map[string]interface{}{"protocolVersion":"2025-06-18","capabilities":map[string]interface{}{"tools":true,"resources":true,"streaming":true}},nil})

	case "tools/list":
		json.NewEncoder(w).Encode(Res{"2.0",req.ID,map[string]interface{}{"tools":ToolSchemas()},nil})

	case "tools/call":
		var p struct{Name string;Input json.RawMessage}
		json.Unmarshal(req.Params,&p)

		switch p.Name{

		case "list":
			res,_:=s.tasks.List()
			json.NewEncoder(w).Encode(Res{"2.0",req.ID,res,nil})

		case "search":
			var in struct{Query string}; json.Unmarshal(p.Input,&in)
			res,_:=s.tasks.Search(in.Query)
			json.NewEncoder(w).Encode(Res{"2.0",req.ID,res,nil})

		case "create":
			var in struct{Title,Notes,Due string}; json.Unmarshal(p.Input,&in)
			res,_:=s.tasks.Create(in.Title,in.Notes,in.Due)
			broadcast("task_created")
			json.NewEncoder(w).Encode(Res{"2.0",req.ID,res,nil})
		}
	}

}

func (s *Server) handleManifest(w http.ResponseWriter,r *http.Request){
	json.NewEncoder(w).Encode(map[string]interface{}{"name":"Google Tasks MCP","tools":ToolSchemas()})
}
