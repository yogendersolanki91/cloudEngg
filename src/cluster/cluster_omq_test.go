package cluster

import (
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
	//"reflect"
	//"sync/atomic"
	//"go/printer"
	//	"fmt"
	//	"fmt"
	"fmt"
)

type Person struct {
	Name string
	Age  int
}

var send = 0

var rcv = 0
var brdsnd = 0
var totalbrodcastmsg = 100

func TestServerSendRcvBrodcast(t *testing.T) {
	cofigpath, _ := filepath.Abs("/home/blackh/IdeaProjects/untitled/src/raftconfig.json")
	var allserver [11]ServerObj
	for i := 1; i <= 10; i++ {
		allserver[i] = New(i, cofigpath)
		go recvprepar(&allserver[i])
		go sendprepare(&allserver[i])
		go brdcstprepare(&allserver[i])
	}
	allbrodcast := (len(allserver) - 2) * (len(allserver) - 1) * totalbrodcastmsg
	time.Sleep(time.Second * 10)

	if rcv == send+allbrodcast {
		t.Log("recived " + strconv.Itoa(rcv) + " Send " + strconv.Itoa(send+allbrodcast) + "    Both are Equal HAPPPY!!!")
	} else {
		t.Error("Simple Peer Communication Test Fail No of send recive msg are not equal sorry... RCV are equal to:-" + strconv.Itoa(rcv) + " Send equal" + strconv.Itoa(send+allbrodcast))

	}
	if brdsnd != rcv-send {
		t.Error("Broadcast Test Failed--- Recived	" + strconv.Itoa(rcv-send) + "Brodacast	Send" + strconv.Itoa(allbrodcast) + "  Both are not equal sorry...bache ki jaan le li")
	} else {

		t.Log("Broadcast Test pass----- Broadcast recived " + strconv.Itoa(rcv-send) + " Brodacast	Send" + strconv.Itoa(allbrodcast) + "  Both are  EQUAL HAPPY")
	}

}

var brdmutex sync.Mutex

func brdsnplus(amount int) {
	brdmutex.Lock()
	brdsnd += amount
	brdmutex.Unlock()

}

var sendmutex sync.Mutex

func sendplus() {
	sendmutex.Lock()
	send += 1
	sendmutex.Unlock()

}

var rcvmutex sync.Mutex

func rcvnplus() {
	rcvmutex.Lock()
	rcv += 1
	rcvmutex.Unlock()

}
func brdcstprepare(myserver *ServerObj) {

	for l := 1; l <= totalbrodcastmsg; l++ {
		myserver.Outbox() <- &Envelope{Pid: -1, MsgId: l, Msg: "dfd"}
		brdsnplus(len(myserver.Peers_o))
	}

	time.Sleep(time.Millisecond * 100)

}

func sendprepare(myserver *ServerObj) {
	var sendreq VoteReq
	var sendRes VoteRespose
	var sendHB HeartBeat
	sendHB.LeaderId = 34
	sendHB.Term = 9
	sendreq.IdCandidate = 90
	sendreq.Term = 33
	sendRes.VoteResult = true
	sendRes.Term = 33

	for k := 1; k <= 10; k++ {
		if myserver.ID != k {
			for l := 1; l <= 1000; l++ {
				switch l % 5 {
				case 0:
					myserver.Outbox() <- &Envelope{Pid: k, MsgId: l, Msg: "chk msg"}
				case 1:
					myserver.Outbox() <- &Envelope{Pid: k, MsgId: l, Msg: 2905}
				case 2:
					myserver.Outbox() <- &Envelope{Pid: k, MsgId: l, Msg: sendreq}
				case 3:
					myserver.Outbox() <- &Envelope{Pid: k, MsgId: l, Msg: sendRes}
				case 4:
					myserver.Outbox() <- &Envelope{Pid: k, MsgId: l, Msg: sendHB}
				}
				sendplus()
			}
			time.Sleep(time.Millisecond * 100)

		}
	}

}
func recvprepar(myserver *ServerObj) {

	for {
		select {
		case x := <-myserver.Inbox():
			var msg Envelope
			msg = *x

			switch msg.Msg.(type) {
			case VoteReq:
				fmt.Print("Req ")
				fmt.Println(msg)
			case VoteRespose:
				fmt.Print("Respns ")
				fmt.Println(msg)
			case int:
				fmt.Print("intger ")
				fmt.Println(msg)
			case string:
				fmt.Print("string ")
				fmt.Println(msg)
			case HeartBeat:
				fmt.Print("HearBeat ")
				fmt.Println(msg)
			}

			//fmt.Println(msg.Msg);
			rcvnplus()

		case <-time.After(10 * time.Second):
			println("Waited and waited. Ab thak gaya   1\n")
		}
	}
}
