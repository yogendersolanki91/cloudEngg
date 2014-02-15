package cluster

import (
	"encoding/json"
	"fmt"
	zmq4 "github.com/pebbe/zmq4"
	"io/ioutil"
	"os"
	"strconv"

//	"io"

)

//to store info written in JSon config File
type ServerConf struct {
	ID   int    `json:"ID"`
	Host string `json:"Host"`
	Port int    `json:"Port"`
}

//main server object that represent a server
type ServerObj struct {
	ID        int    //id of server
	Host      string //host address
	Port      int
	Peers_o   map[int]ServerConf //info of all other host
	In_chnl   chan *Envelope
	Out_chnl  chan *Envelope
	myconn    *zmq4.Socket         //socket to litsen other
	peer_conn map[int]*zmq4.Socket //socket that connect this server to all other server maped by id of that servre
}

//to maintain global information about all servers
type Allserver struct {
	Servers []ServerConf
}

const (
	BRODCAST = -1
)

//Structure of the envelope used to send msg
type Envelope struct {
	Pid   int
	MsgId int
	Msg   interface{}
}

func (b Envelope) String() string {

	return fmt.Sprintf("%b", b)
}
func (b Envelope) OtherString() string {

	return fmt.Sprintf("%b", b.Msg)
}

//basic Function that will be accessed by the server object
type Server interface {
	Pid() int
	Peers() []int
	Outbox() chan *Envelope
	Inbox() chan *Envelope
}

//will return the id of the server
func (current ServerObj) Pid() int {
	return current.ID
}

//will return the all peer servers in the cluster
func (current ServerObj) Peers() map[int]ServerConf {
	return current.Peers_o
}

//chandal to pass msg that has been received
func (current ServerObj) Inbox() chan *Envelope {

	return current.In_chnl

}

//chandal will be used to send the msg
func (current ServerObj) Outbox() chan *Envelope {
	return current.Out_chnl
}

//will be used internally to get all info from file
func getAllserver(cofg string) Allserver {

	file, _ := ioutil.ReadFile(cofg)
	var jsontype Allserver
	json.Unmarshal(file, &jsontype)
	if len(jsontype.Servers) < 1 {
		//exiting after showing msg if there is no server or there is empty or corrupted file
		panic("Either Something wrong with config file or file has not valid info or check the config file path")
		os.Exit(29)
	}
	//returning all server
	return jsontype
}

// to send the envelope over the network we need string this will provide formatted string fro json encoding ans decoding of the message
func wrapMsg(msg Envelope) string {
	var send string
	switch msg.Msg.(type) {
	case string:
		send = "{\"Pid\":" + strconv.Itoa(msg.Pid) + ",\"MsgId\":" + strconv.Itoa(msg.MsgId) + ",\"Msg\":\"" + msg.Msg.(string) + "\"}"
	case int:
		send = "{\"Pid\":" + strconv.Itoa(msg.Pid) + ",\"MsgId\":" + strconv.Itoa(msg.MsgId) + ",\"Msg\":\"" + strconv.Itoa(msg.Msg.(int)) + "\"}"
	default:
		send = "{\"Pid\":" + strconv.Itoa(msg.Pid) + ",\"MsgId\":" + strconv.Itoa(msg.MsgId) + ",\"Msg\":\"" + msg.OtherString() + "\"}"
	}
	return send
}

// this will decode msg back to Envelope structure from the json encode msg
func unwrapMs(msg string) Envelope {
	var toretun Envelope
	json.Unmarshal([]byte(msg), &toretun)
	return toretun
}

//here is the all stuff that required to create a server object
func New(id int, cofg string) ServerObj {
	//getting info abut all server that is available
	global_object := getAllserver(cofg)
	var newServer ServerObj
	mypeer := make(map[int]ServerConf)
	// first create all peer map object and initialize the server object
	for i := range global_object.Servers {

		if global_object.Servers[i].ID != id {

			//add to peers list if not server itself
			mypeer[global_object.Servers[i].ID] = global_object.Servers[i]
		} else {
			//add info of self if not peer
			newServer.ID = global_object.Servers[i].ID
			newServer.Host = global_object.Servers[i].Host
			newServer.Port = global_object.Servers[i].Port

		}

	}

	//assign everything that is related to this server
	newServer.Peers_o = mypeer

	// initialising outbox inbox chalnal
	newServer.In_chnl = make(chan *Envelope)
	newServer.Out_chnl = make(chan *Envelope)

	//server itself start listnig at port defined in config file
	bindConn := "tcp://*:" + strconv.Itoa(newServer.Port)
	myconn, _ := zmq4.NewSocket(zmq4.PULL)
	myconn.Bind(bindConn)
	newServer.myconn = myconn

	// initialising socket for the peer server
	newServer.peer_conn = make(map[int]*zmq4.Socket)
	for key, srvr := range newServer.Peers_o {
		conect := "tcp://" + srvr.Host + ":" + strconv.Itoa(srvr.Port)
		conn, er01 := zmq4.NewSocket(zmq4.PUSH)
		if er01 != nil {
			panic(er01)
		}
		conn.Connect(conect)
		newServer.peer_conn[key] = conn

	}

	//go routing to use receive msg from peer
	go func() {
		for {

			rcvmsg, er := myconn.Recv(0)

			if er != nil {
				panic(er)
			}

			var msg Envelope
			msg = unwrapMs(rcvmsg)
			newServer.Inbox() <- &msg

		}
	}()

	//go routine to send data that is coming over out chanel
	go func() {
		for {
			select {
			case x := <-newServer.Out_chnl:
				var msg Envelope
				msg = *x
				if msg.Pid != BRODCAST {
					newServer.peer_conn[msg.Pid].Send(wrapMsg(Envelope{Pid: newServer.Pid(), MsgId: msg.MsgId, Msg: msg.Msg}), 0)
				} else {
					for _, sockpeer := range newServer.peer_conn {
						sockpeer.Send(wrapMsg(Envelope{Pid: newServer.Pid(), MsgId: msg.MsgId, Msg: msg.Msg}), 0)
					}

				}

			}

		}
	}()
	//now returning this Serverobj
	return newServer
}
