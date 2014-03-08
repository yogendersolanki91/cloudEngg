package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"raft"
	"strconv"
	"time"
)

type Comm struct {
	Term       int
	LeaderShip bool
}

func (c *Comm) IsLeaderRPC(none bool, answer *bool) error {
	*answer = raftVal.IsLeader()
	return nil
}
func (c *Comm) GetTermRPC(none bool, answer *int) error {
	*answer = raftVal.Term()
	return nil
}
func (c *Comm) KillRPC(none bool, answer *int) error {
	time.AfterFunc(300*time.Millisecond, func() { os.Exit(2) })
	return nil
}

var raftVal raft.Raft
var Sender Comm
var id = flag.Int("id", 0, "Id of the the raft instance or server (required and Non Zero)")
var dbgport = flag.Int("dbgport", 0, "Port for the debugging process used by test case (required and Non Zero)")
var dbgmsg = flag.Int("dbgmsg", 0, "Whether to show or not the log/debug message (it is optional)")

func connect() net.Listener {
	resp, err := net.Listen("tcp", ":"+strconv.Itoa(*dbgport))
	if err != nil {
		fmt.Println("error occured while runnig server"+ err.Error())
		f.WriteString("error in Connect Line 41 -"+ err.Error())
		os.Exit(5)
	}
	return resp
}
func handler(litsn *net.Listener) {
	for {
		conn, err := (*litsn).Accept()
		if err != nil {
				f.WriteString("error in handler Line 49 "+ err.Error())

		}
		rpc.ServeConn(conn)
	}
}
var f *os.File;
func main() {


	flag.Parse()
	if *id == 0 || *dbgport == 0 {
		fmt.Println("Can't run because proper arguments not given.Use -h or --help to get info about arguments")
		os.Exit(2)
	}

	f, _ = os.OpenFile("testlogfile"+strconv.Itoa(*id), os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)

	rpc.Register(&Sender)
	litsner := connect()
	go handler(&litsner)
	raftVal = *raft.NewRaft(*id, "/home/blackh/IdeaProjects/untitled/src/raftconfig.json")
	if *dbgmsg>0 {
		raftVal.DebugOn()
	}

	time.Sleep(time.Hour)
}
