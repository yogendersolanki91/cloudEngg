package main

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
)

type Server struct {
	data map[string]string
}

type Args struct {
	Key   string
	Value string
}

func (Store *Server) Initi() *Server {
	Store.data = make(map[string]string)
	return Store
}
func (Store *Server) AddKey(args Args, reply *string) error {
	if Store.data[args.Key] == "" {
		Store.data[args.Key] = args.Value
	} else {
		*reply = "err"
	}
	return nil
}
func (Store *Server) GetKey(args Args, reply *string) error {
	if Store.data[args.Key] != "" {
		*reply = Store.data[args.Key]
	} else {
		*reply = "err"
	}
	return nil
}
func (Store *Server) Update(args Args, reply *string) error {
	if Store.data[args.Key] != "" {
		Store.data[args.Key] = args.Value
		*reply = "OK"
	} else {
		*reply = "err."
	}
	return nil
}
func (Store *Server) Delete(args Args, reply *string) error {
	if Store.data[args.Key] != "" {
		delete(Store.data, args.Key)
		*reply = "OK"

	} else {
		*reply = "err"
	}
	return nil
}
func connect() net.Listener {
	resp, err := net.Listen("tcp", ":2905")
	if err != nil {
		fmt.Println("error occured while runnig server")
		os.Exit(5)
	}
	return resp
}
func handler(dataPOint *Server, litsn *net.Listener) {
	for {
		conn, err := litsn.Accept()

		if err != nil {
		}
		rpc.ServeConn(conn)
		fmt.Println(dataPOint.data)

	}
}
func main() {
	srver := new(Server).Initi()
	rpc.Register(srver)
	litsner := connect()
	go handler(srver, &litsner)

}
