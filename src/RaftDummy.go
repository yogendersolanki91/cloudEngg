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
	"math/rand"
	leveldb "github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"encoding/gob"
	"bytes"
	"cluster"
	"log"
)

type Comm struct {
	Term       int
	LeaderShip bool
	isStart   bool
}
type get struct {
	Key []byte
	Ro  *opt.ReadOptions
}
type put struct {
	Key []byte
	Value []byte
	Wo  *opt.WriteOptions
}
type delete struct {
	Key []byte
	Wo  *opt.WriteOptions
}
func store(data interface{})[]byte {
	gob.Register(get{})
	gob.Register(put{})
	gob.Register(delete{})
	m := new(bytes.Buffer)
	enc := gob.NewEncoder(m)
	err := enc.Encode(&data)
	if err != nil { panic(err) }
	return m.Bytes()
}
func load(byts []byte)interface {} {
	var e interface{}
	gob.Register(get{})
	gob.Register(put{})
	gob.Register(delete{})
	p := new(bytes.Buffer)
	p.Write(byts)
	dec := gob.NewDecoder(p)
	err := dec.Decode(&e)
	if err != nil {
		panic(err) }else{
	}
	return e
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
func (c *Comm)DoOperation(whattodo bool,non *int)error{
		c.isStart=whattodo
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
		fmt.Println("error occured while runnig server" + err.Error())
		panic(err)
		os.Exit(5)
	}
	return resp
}
func handler(litsn *net.Listener) {
	for {
		conn, err := (*litsn).Accept()
		if err != nil {
			log.Println(err)
			panic(err)

		}
		rpc.ServeConn(conn)
	}
}


func random(max int) []byte {
	x:= rand.Intn(max)
	return []byte(strconv.Itoa(x))
}

func main() {
	flag.Parse()
	if *id == 0 || *dbgport == 0 {
		fmt.Println("Can't run because proper arguments not given.Use -h or --help to get info about arguments")
		os.Exit(2)
	}

	rpc.Register(&Sender)
	litsner := connect()
	go handler(&litsner)
	raftVal = *raft.NewRaft(*id, "C:\\CloudSrc\\src\\raftconfig.json")
	db,_:=leveldb.OpenFile("db_"+strconv.Itoa(*id)+".db",nil)

	if *dbgmsg > 0 {
		raftVal.DebugOn()
	}

	go func(){
		i:=0
		for i<100 {

			if raftVal.IsLeader(){
				var op []byte
				switch i%2{
				case 0:
					op=store(put{Key:random(40),Value:random(40),Wo:nil})
					log.Println("Doing PUT",i)
				case 1:
					op=store(delete{Key:random(40),Wo:nil})
					log.Println("Doing DEL",i)
				}
				i++

				raftVal.Inbox()<-&op
			}
			time.Sleep(time.Millisecond*500)
		}
	}()
	go func(){
			for {

					select {
					case y := <-raftVal.Outbox():
						var command cluster.LogValue
						command = *y;
						var operation interface{}
						operation = load(command.Operands)
						switch operation.(type){
						case put:
							db.Put(operation.(put).Key, operation.(put).Value, operation.(put).Wo)
							log.Println("PUT",string(operation.(put).Key),operation.(put).Value)
						case delete:
							db.Delete(operation.(delete).Key, operation.(delete).Wo)
							log.Println("delete",operation.(delete).Key)

						}





				}
			}
	}()

	time.Sleep(time.Hour)
}
