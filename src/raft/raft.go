//To understand the raft reference of official GO-RAFT has been taken but no part of the code has taken.
//Link to the reference https://github.com/goraft/
//Currently The Term is not stored in persistent memory..it will be done soon
package raft

import (
	"cluster"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

//Defining All Stat of the Raft
const (
	Leader          = 1
	Candidate       = 2
	Follower        = 3
	HearBeatTimeout = 50 //this is the interval by which leader will broadcast the HB
)

//Main Raft Class to manage all properties of a raft instanse
type rafTclass struct {
	currentTerm            int          //Current term of server
	termMtx                sync.RWMutex //Mutex For term
	RoleMtx                sync.RWMutex //Mutex for State of server
	startMtx               sync.RWMutex //mutext for Start
	votedFor               int          //whom to voted last
	totalvote              int          //total vote recived when it is candidate
	myID                   int          // ID of sever
	LastComm               time.Time
	currentState           int //State Menas Leader or folllwer or candidate
	currentLeader          int // who is my leader
	HeartBeatTime          int
	ElectionTimeout        int // What is the election timeout for me
	electionTimeoutchannel chan bool
	started                bool              //is server running
	servreEntity           cluster.ServerObj //Servre cluster object
	LogFilePATH            string            //path to store log files
	TermFilePath           string            //path to store term in Disk
}

//Some structure to map with JSON CONFIG
type timeouts struct {
	ID             int
	ElctionTimeout int
}
type logFile struct {
	ID       int
	Filename string
}
type termFile struct {
	ID       int
	Filename string
}

//to get raftConfig from the Disk
type raftConfig struct {
	Servers  []cluster.ServerConf `json:"Servers"`
	Timeouts []timeouts           `json:"Timeouts"`
	LogFile  []logFile            `json:"LogFile"`
	TermFile []termFile           `json:"TermFile"`
}

//Main Raft Interface that is accessible out side the package
type Raft interface {
	Term() int
	IsLeader() bool
	Stop()
	Start()
}

//Update state of the serevr
func (r *rafTclass) updateState(stat int) {
	r.RoleMtx.Lock()
	r.currentState = stat
	r.RoleMtx.Unlock()
}

//read current state
func (r *rafTclass) getState() int {
	r.RoleMtx.RLock()
	stat := r.currentState
	r.RoleMtx.RUnlock()
	return stat
}

//to stop or kill the serevr
func (r *rafTclass) Stop() {
	r.updateStarted(false)
	r.updateState(Follower)
	writeTermFromFile(r.TermFilePath, r.getTerm())
}

// to start or wakeup killed server
func (RaftObj *rafTclass) Start() {
	RaftObj.updateStarted(true)
	RaftObj.updateTerm(readTermFromFile(RaftObj.TermFilePath))

	go func() {
		for RaftObj.getStart() {
			//time.Sleep(100*time.Nanosecond)
			//	fmt.Println("I am running "+strconv.Itoa(RaftObj.myID))
			switch RaftObj.getState() {
			case Follower: //If it is a Follower then perform this
				RaftObj.performasFollower()
			case Candidate: //If it is a candidate then perform this
				RaftObj.perfomeasCandidate()
			case Leader: //If it is a Leader then perform this
				RaftObj.perfomeasLeader()
			}

		}
	}()

}

// check that am i the leader
func (r *rafTclass) IsLeader() bool {
	r.RoleMtx.RLock()
	term := r.currentState
	r.RoleMtx.RUnlock()
	//fmt.Println(term)
	if term != Leader {
		return false
	}
	return true
}

//get current term of the server
func (r *rafTclass) Term() int {
	r.termMtx.RLock()
	term := r.currentTerm
	r.termMtx.RUnlock()
	return term
}

//Update term of the server
func (r *rafTclass) updateTerm(term int) {
	r.termMtx.Lock()
	//fmt.Println(r.TermFilePath)
	writeTermFromFile(r.TermFilePath, term)
	r.currentTerm = term
	r.termMtx.Unlock()
}

//Get the term that is used internally
func (r *rafTclass) getTerm() int {
	r.termMtx.RLock()
	term := r.currentTerm
	r.termMtx.RUnlock()
	return term
}

//update start variable that kill or wakeup serevr
func (r *rafTclass) updateStarted(start bool) {
	r.startMtx.Lock()
	r.started = start
	r.startMtx.Unlock()
}

//GET start variable that kill or wakeup serevr
func (r *rafTclass) getStart() bool {
	r.startMtx.RLock()
	start := r.started
	r.startMtx.RUnlock()
	return start

}

//Miscellaneous Function of Raft

//RAFT initializer
func NewRaft(id int, path string) *Raft {
	//Intialising Object Properties
	RaftObj := new(rafTclass)
	cofigpath, _ := filepath.Abs(path)
	alldetails := getALLCOnfig(cofigpath)
	//Getting Election Time Out From COnfig Object
	for timeOu := range alldetails.Timeouts {
		if alldetails.Timeouts[timeOu].ID == id {
			RaftObj.ElectionTimeout = alldetails.Timeouts[timeOu].ElctionTimeout
		}
		if alldetails.LogFile[timeOu].ID == id {
			RaftObj.LogFilePATH = alldetails.LogFile[timeOu].Filename
		}
		if alldetails.TermFile[timeOu].ID == id {
			RaftObj.TermFilePath = alldetails.TermFile[timeOu].Filename
		}
	}
	RaftObj.HeartBeatTime = 100
	RaftObj.updateState(Follower)
	RaftObj.updateTerm(1)
	RaftObj.myID = id
	RaftObj.votedFor = 0
	RaftObj.updateStarted(false)
	//RaftObj.Stop()
	RaftObj.electionTimeoutchannel = make(chan bool)
	RaftObj.servreEntity = cluster.New(id, path)

	//fmt.Println(RaftObj.TermFilePath)
	//Main Task Manager based in the state task will be chosen here
	RaftObj.votedFor = 0
	RaftObj.LastComm = time.Now()
	//go RaftObj.timeout();
	fmt.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Start -> Follower")
	RaftObj.Start()
	var x Raft
	x = &*RaftObj
	return &x
}

//Evaluate that is it ok to send vote if it is ok then return TRUE
func (RaftObj *rafTclass) sendVote(req cluster.VoteReq) bool {
	//If older request then current term then REFUSE
	if req.Term <= RaftObj.getTerm() {
		//fmt.Println("got Negative Old term res term"+strconv.Itoa(req.Term)+" MY term "+strconv.Itoa(RaftObj.currentTerm))
		return false
	}
	//if already voted then no need to vote-
	if RaftObj.votedFor != req.IdCandidate && RaftObj.votedFor != 0 && req.Term <= RaftObj.getTerm() {
		//fmt.Println("got Negative Already Voted res term"+strconv.Itoa(req.Term)+" MY term "+strconv.Itoa(RaftObj.currentTerm))
		return false
	}
	//else perform the operation like term change and vote for change and Accept
	RaftObj.updateTerm(req.Term)
	RaftObj.votedFor = req.IdCandidate
	RaftObj.LastComm = time.Now()
	return true
}

//this is the work that Follower will perform
func (RaftObj *rafTclass) performasFollower() {
	if RaftObj.getState() == Follower {
		RaftObj.totalvote = 0
		select {
		//check for incoming message
		case x := <-RaftObj.servreEntity.Inbox():
			var msg cluster.Envelope
			msg = *x
			switch msg.Msg.(type) {
			case cluster.HeartBeat:
				hb := msg.Msg.(cluster.HeartBeat)
				//if we have new leader elected then forget the older one
				if hb.Term >= RaftObj.getTerm() {
					RaftObj.currentLeader = hb.LeaderId
					RaftObj.updateTerm(hb.Term)
				}
			case cluster.VoteReq:
				req := msg.Msg.(cluster.VoteReq)
				//check can we give a positive vote
				if RaftObj.sendVote(req) {
					RaftObj.servreEntity.Outbox() <- &cluster.Envelope{Pid: req.IdCandidate, MsgId: 90, Msg: cluster.VoteRespose{Term: req.Term, VoteResult: true}}
					//fmt.Println("got postive res term"+strconv.Itoa(req.Term)+" MY term "+strconv.Itoa(RaftObj.currentTerm))
				} else {
					RaftObj.servreEntity.Outbox() <- &cluster.Envelope{Pid: req.IdCandidate, MsgId: 90, Msg: cluster.VoteRespose{Term: req.Term, VoteResult: false}}

				}

			}
			//If no msg from any where then Timeout and Become Candidate
		case <-time.After(time.Duration(RaftObj.ElectionTimeout) * time.Millisecond):
			RaftObj.votedFor = RaftObj.myID
			RaftObj.totalvote = 1
			RaftObj.updateTerm(RaftObj.getTerm() + 1)
			RaftObj.updateState(Candidate)
			//Broadcast the request for vote
			RaftObj.servreEntity.Outbox() <- &cluster.Envelope{Pid: cluster.BRODCAST, MsgId: 90, Msg: cluster.VoteReq{Term: RaftObj.currentTerm, IdCandidate: RaftObj.myID}}
			fmt.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + "Timeout Follower->Candidate")
		}

	}

}
func (RaftObj *rafTclass) perfomeasCandidate() {

	if RaftObj.getState() == Candidate {
		select {
		case x := <-RaftObj.servreEntity.Inbox():
			var msg cluster.Envelope
			msg = *x
			switch msg.Msg.(type) {

			case cluster.HeartBeat:
				//If gets any info that anyone have higher term or older leader step back to the follower
				if msg.Msg.(cluster.HeartBeat).Term >= RaftObj.currentTerm || RaftObj.currentLeader == msg.Msg.(cluster.HeartBeat).LeaderId {
					RaftObj.currentLeader = msg.Msg.(cluster.HeartBeat).LeaderId
					RaftObj.updateTerm(msg.Msg.(cluster.HeartBeat).Term)
					RaftObj.updateState(Follower)
					fmt.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Candidate -> Follower")
				}

				//If gets any voteReqThen it will remove refuse all to give vote because it voted for himself
			case cluster.VoteReq:
				req := msg.Msg.(cluster.VoteReq)
				RaftObj.servreEntity.Outbox() <- &cluster.Envelope{Pid: req.IdCandidate, MsgId: 90, Msg: cluster.VoteRespose{Term: msg.Msg.(cluster.VoteReq).Term, VoteResult: false}}

			case cluster.VoteRespose:
				res := msg.Msg.(cluster.VoteRespose)
				//count Positive votes
				if res.Term <= RaftObj.getTerm() && res.VoteResult {

					RaftObj.totalvote++
				}
				//if have enough vote then go on become Leader
				if RaftObj.totalvote > 5 {
					fmt.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Candidate -> Leader Votes-" + strconv.Itoa(RaftObj.totalvote))
					RaftObj.updateState(Leader)

				}
			}
			//Start New Election if there is no HearBeat Msg NO leader and No Majority
		case <-time.After(time.Duration(RaftObj.ElectionTimeout) * time.Millisecond):
			//Confirm Once Again Before re-election
			if RaftObj.totalvote > 5 {
				fmt.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Candidate -> Leader Votes-" + strconv.Itoa(RaftObj.totalvote))
				RaftObj.updateState(Leader)

			} else {
				fmt.Println(strconv.Itoa(RaftObj.myID) + " -  Term " + strconv.Itoa(RaftObj.currentTerm) + " Relection Not Having Enough Votes Total Votes " + strconv.Itoa(RaftObj.totalvote))
				RaftObj.updateTerm(RaftObj.getTerm() + 1)
				RaftObj.servreEntity.Outbox() <- &cluster.Envelope{Pid: cluster.BRODCAST, MsgId: 90, Msg: cluster.VoteReq{Term: RaftObj.currentTerm, IdCandidate: RaftObj.myID}}
				RaftObj.votedFor = RaftObj.myID
				RaftObj.currentLeader = 0
				RaftObj.totalvote = 1

			}

		}
	}
}
func (RaftObj *rafTclass) perfomeasLeader() {

	if RaftObj.getState() == Leader {

		time.Sleep(HearBeatTimeout * time.Millisecond)
		//Broadcast Heat Beat to EveryOne
		RaftObj.servreEntity.Outbox() <- &cluster.Envelope{Pid: cluster.BRODCAST, MsgId: 90, Msg: cluster.HeartBeat{Term: RaftObj.currentTerm, LeaderId: RaftObj.myID}}
		select {
		//if any one is Leader with Higher term Then step Back to follower
		case x := <-RaftObj.servreEntity.Inbox():
			var msg cluster.Envelope
			msg = *x
			switch msg.Msg.(type) {
			case cluster.HeartBeat:
				if msg.Msg.(cluster.HeartBeat).Term > RaftObj.currentTerm {
					RaftObj.updateState(Follower)
					RaftObj.updateTerm(msg.Msg.(cluster.HeartBeat).Term)
					fmt.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Leader->Follower")
				}
			}
		default:

		}

	}

}
func getALLCOnfig(path string) raftConfig {
	file, _ := ioutil.ReadFile(path)
	var raftcon raftConfig
	json.Unmarshal(file, &raftcon)
	//	fmt.Println(raftcon);
	return raftcon
}

//read current save term in disk
func readTermFromFile(path string) int {
	fil, er := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0660)
	if er != nil {
		return 1

	}
	var b []byte
	fil.Read(b)
	x, _ := strconv.Atoi(string(b))
	fil.Close()
	return x
}

//Write current term to disk.
func writeTermFromFile(path string, term int) {
	//fmt.Println(path)
	x, _ := filepath.Abs(".")
	x = x + "/" + path
	//fmt.Println(x);
	fil, er := os.OpenFile(x, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if er != nil {
		//panic(path)
		panic(er)
	}
	fil.Write([]byte(strconv.Itoa(term)))
	fil.Close()
}
