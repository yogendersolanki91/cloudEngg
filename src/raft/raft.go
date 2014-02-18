//To understand the raft reference of official GO-RAFT has been taken but no part of the code has taken.
//Link to the reference https://github.com/goraft/
//Currently The Term is not stored in persistent memory..it will be done soon
package raft

import (
	"cluster"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"time"
	//	"image/gif"
	//"os/exec"
	//	"path/filepath"
	"fmt"
	"strconv"
)

//Defining All Stat of the Raft
const (
	Leader    = 1
	Candidate = 2
	Follower  = 3
)

//Main Raft Class to manage all properties of a raft instanse
type rafTclass struct {
	currentTerm            int
	votedFor               int
	totalvote              int
	myID                   int
	LastComm               time.Time
	currentState           int
	currentLeader          int
	HeartBeatTime          int
	ElectionTimeout        int
	electionTimeoutchannel chan bool
	started                bool
	servreEntity           cluster.ServerObj
	LogFilePATH            string
}

type timeouts struct {
	ID             int
	ElctionTimeout int
}

//to get raftConfig from the Disk
type raftConfig struct {
	Servers  []cluster.ServerConf `json:"Servers"`
	Timeouts []timeouts           `json:"Timeouts"`
	LogFile  string               `json:"LogFile"`
}

//Main Raft Interface that is accessible out side the package
type Raft interface {
	Term() int
	IsLeader() bool
}

//Miscellaneous Function of Raft
func (r *rafTclass) Stop() {
	r.started = false
	fmt.Println("Stoped")
}

func (r rafTclass) Term() int {
	return r.currentTerm
}
func (r rafTclass) IsLeader() bool {
	if r.currentState == Leader {
		return true
	} else {
		return false
	}
}

func (r *rafTclass) Start() {
	r.started = true
}

//RAFT initializer
func NewRaft(id int, path string) *Raft {
	//Intialising Object Properties
	var RaftObj rafTclass
	cofigpath, _ := filepath.Abs(path)
	alldetails := getALLCOnfig(cofigpath)
	RaftObj.HeartBeatTime = 100
	RaftObj.currentState = Follower
	RaftObj.currentTerm = 1
	RaftObj.myID = id
	RaftObj.votedFor = 0
	RaftObj.Stop()
	RaftObj.electionTimeoutchannel = make(chan bool)
	RaftObj.servreEntity = cluster.New(id, path)
	//Getting Election Time Out From COnfig Object
	for timeOu := range alldetails.Timeouts {
		if alldetails.Timeouts[timeOu].ID == id {
			RaftObj.ElectionTimeout = alldetails.Timeouts[timeOu].ElctionTimeout
		}
	}
	//Main Task Manager based in the state task will be chosen here
	go func() {
		RaftObj.votedFor = 0
		RaftObj.Start()
		RaftObj.LastComm = time.Now()
		//go RaftObj.timeout();
		fmt.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Start -> Follower")
		for {
			if RaftObj.started {
				switch RaftObj.currentState {

				case Follower: //If it is a Follower then perform this
					RaftObj.performasFollower()
				case Candidate: //If it is a candidate then perform this
					RaftObj.perfomeasCandidate()
				case Leader: //If it is a Leader then perform this
					RaftObj.perfomeasLeader()
				}
			} else {
				RaftObj.currentState = Follower
			}

		}
	}()
	var sender Raft
	sender = RaftObj
	return &sender
}

//Evaluate that is it ok to send vote if it is ok then return TRUE
func (RaftObj *rafTclass) sendVote(req cluster.VoteReq) bool {
	//If older request then current term then REFUSE
	if req.Term <= RaftObj.currentTerm {
		return false
	}
	//if already voted then no need to vote
	if RaftObj.votedFor != req.IdCandidate && RaftObj.votedFor != 0 {
		//fmt.Println(strconv.Itoa(RaftObj.myID) + ":Already Voted")
		return false
	}
	//else perform the operation like term change and vote for change and Accept
	RaftObj.currentTerm = req.Term
	RaftObj.votedFor = req.IdCandidate
	RaftObj.LastComm = time.Now()
	return true
}

//this is the work that Follower will perform
func (RaftObj *rafTclass) performasFollower() {
	if RaftObj.currentState == Follower {
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
				if hb.Term >= RaftObj.currentTerm {
					RaftObj.currentLeader = hb.LeaderId
					RaftObj.currentTerm = hb.Term
					RaftObj.LastComm = time.Now()
				}
				RaftObj.LastComm = time.Now()

			case cluster.VoteReq:
				req := msg.Msg.(cluster.VoteReq)
				//check can we give a positive vote
				if RaftObj.sendVote(req) {
					RaftObj.servreEntity.Outbox() <- &cluster.Envelope{Pid: req.IdCandidate, MsgId: 90, Msg: cluster.VoteRespose{Term: req.Term, VoteResult: true}}
				} else {
					RaftObj.servreEntity.Outbox() <- &cluster.Envelope{Pid: req.IdCandidate, MsgId: 90, Msg: cluster.VoteRespose{Term: req.Term, VoteResult: false}}
				}
				RaftObj.LastComm = time.Now()

			}
		//If no msg from any where then Timeout and Become Candidate
		case <-time.After(time.Duration(RaftObj.ElectionTimeout) * time.Millisecond):
			RaftObj.votedFor = RaftObj.myID
			RaftObj.totalvote = 1
			RaftObj.currentState = Candidate
			//Broadcast the request for vote
			RaftObj.servreEntity.Outbox() <- &cluster.Envelope{Pid: cluster.BRODCAST, MsgId: 90, Msg: cluster.VoteReq{Term: RaftObj.currentTerm, IdCandidate: RaftObj.myID}}
			fmt.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + "Timeout Follower->Candidate")
		}

	}

}
func (RaftObj *rafTclass) perfomeasCandidate() {

	if RaftObj.currentState == Candidate {
		select {
		case x := <-RaftObj.servreEntity.Inbox():
			var msg cluster.Envelope
			msg = *x
			switch msg.Msg.(type) {

			case cluster.HeartBeat:
				//If gets any info that anyone have higher term or older leader step back to the follower
				if msg.Msg.(cluster.HeartBeat).Term > RaftObj.currentTerm || RaftObj.currentLeader == msg.Msg.(cluster.HeartBeat).LeaderId {
					RaftObj.currentLeader = msg.Msg.(cluster.HeartBeat).LeaderId
					RaftObj.currentTerm = msg.Msg.(cluster.HeartBeat).Term
					RaftObj.currentState = Follower
					fmt.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Candidate -> Follower")
				}
				RaftObj.LastComm = time.Now()
				//If gets any voteReqThen it will remove refuse all to give vote because it voted for himself
			case cluster.VoteReq:
				req := msg.Msg.(cluster.VoteReq)
				RaftObj.servreEntity.Outbox() <- &cluster.Envelope{Pid: req.IdCandidate, MsgId: 90, Msg: cluster.VoteRespose{Term: msg.Msg.(cluster.VoteReq).Term, VoteResult: false}}

			case cluster.VoteRespose:
				res := msg.Msg.(cluster.VoteRespose)
				//count Positive votes
				if res.Term <= RaftObj.currentTerm && res.VoteResult {

					RaftObj.totalvote++
				}
				//if have enough vote then go on become Leader
				if RaftObj.totalvote > 5 {
					fmt.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Candidate -> Leader Votes-" + strconv.Itoa(RaftObj.totalvote))
					RaftObj.currentState = Leader

				}
			}
		//Start New Election if there is no HearBeat Msg NO leader and No Majority
		case <-time.After(time.Duration(RaftObj.ElectionTimeout) * time.Millisecond):
			//Confirm Once Again  Before re-election
			if RaftObj.totalvote > 5 {
				fmt.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Candidate -> Leader Votes-" + strconv.Itoa(RaftObj.totalvote))
				RaftObj.currentState = Leader

			} else {
				RaftObj.currentTerm += 1
				RaftObj.servreEntity.Outbox() <- &cluster.Envelope{Pid: cluster.BRODCAST, MsgId: 90, Msg: cluster.VoteReq{Term: RaftObj.currentTerm, IdCandidate: RaftObj.myID}}
				RaftObj.votedFor = RaftObj.myID
				RaftObj.currentLeader = 0
				RaftObj.totalvote = 1

			}

		}
	}
}
func (RaftObj *rafTclass) perfomeasLeader() {

	if RaftObj.currentState == Leader {
		RaftObj.LastComm = time.Now()
		time.Sleep(80 * time.Millisecond)
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
					RaftObj.currentState = Follower
					RaftObj.currentTerm = msg.Msg.(cluster.HeartBeat).Term
					fmt.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Leader->Follower")
				}
				RaftObj.LastComm = time.Now()
			}
		default:

		}

	}

}
func getALLCOnfig(path string) raftConfig {
	file, _ := ioutil.ReadFile(path)
	var raftcon raftConfig
	json.Unmarshal(file, &raftcon)
	return raftcon
}
