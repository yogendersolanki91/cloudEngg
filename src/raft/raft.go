//To understand the raft reference of official GO-RAFT has been taken but no part of the code has taken.
//Link to the reference https://github.com/goraft/
package raft

import (
	"bufio"
	"cluster"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"raftLog"
	"strconv"
	"sync"
	"time"
)

//Defining All Stat of the Raft
const (
	Leader          = 1
	Candidate       = 2
	Follower        = 3
	HearBeatTimeout = 200 //this is the interval by which leader will broadcast the HB
	LogValueAmount  = 500
)

//Main Raft Class to manage all properties of a raft instanse
//Some structure to map with JSON CONFIG
type timeouts struct {
	ID              int
	ElectionTimeout int
}

//Term and Committed index are saved in the disk and this strucuture repersent the same
type presistState struct {
	Term          int
	Commitedindex int
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
type logFile struct {
	ID       int
	Filename string
}

//Main Raft Interface that is accessible out side the package
type Raft interface {
	Term() int
	IsLeader() bool
	Stop()
	Start()
	DebugOn()
	DebugOff()
	Inbox() chan *[]byte
	Outbox() chan *cluster.LogValue
}

//Whenever any leader KV server recive any operation request
//this reuset will be encoded in GOB and then that encoded data will be passed
//to this chanal.This chanal will process the raplicaiton.
func (RaftObj *raftState) Inbox() chan *[]byte {
	return RaftObj.inputRequest
}

//Whenever a replication complets on majority that request will be passed
//to the KV store to Commit by this chanal
func (RaftObj *raftState) Outbox() chan *cluster.LogValue {
	return RaftObj.commitThis
}

//LogSender() make a goroutine for each client to brodcast them required log
//if a server is having out of order log then it will also handle them
//accordingaly
func (RaftObj *raftState) LogSender() {
	for key, _ := range RaftObj.serverEntity.Peers_o {
		if key != RaftObj.myID {
			RaftObj.logWrite.Setindex(key, RaftObj.logWrite.CommittedIndex+1)
		}
	}
	for key, _ := range RaftObj.serverEntity.Peers_o {
		go func(index int, Raftu *raftState) {
			for Raftu.IsLeader() {
				if Raftu.logWrite.Getindex(Raftu.myID) > Raftu.logWrite.Getindex(index) && Raftu.logWrite.LogState[index] >= 0 {
					RaftObj.serverEntity.Outbox() <- &cluster.Envelope{Pid: index, MsgId: Raftu.logWrite.Getindex(index),
						Msg: Raftu.logWrite.PrepareEntryToSend(Raftu.logWrite.Getindex(index), Raftu.getTerm(), Raftu.myID)}
					Raftu.logWrite.LogState[index] = -1
					//RaftObj.logWrite.Setindex(index, RaftObj.logWrite.Getindex(index)+1)
				} else {
					time.Sleep(100 * time.Millisecond)
				}
			}
		}(key, RaftObj)

	}
}

//Want to debug msg on you must use this fucntion
func (r *raftState) DebugOn() {
	r.isDebugOn = true
}

//Want to debug msg off ..you must use this fucntion
func (r *raftState) DebugOff() {
	r.isDebugOn = false
}

//Update state of the serevr
func (r *raftState) updateState(stat int) {
	r.roleMtx.Lock()
	r.currentState = stat
	r.roleMtx.Unlock()
}

//read current state
func (r *raftState) getState() int {
	r.roleMtx.RLock()
	stat := r.currentState
	r.roleMtx.RUnlock()
	return stat
}

//to stop or kill the server
func (r *raftState) Stop() {
	r.updateStarted(false)
	r.updateState(Follower)
	if r.isDebugOn {
		log.Println("Stoped Litsning")
	}
	r.writeTerm()
}

// to start or wakeup killed server
func (RaftObj *raftState) Start() {
	RaftObj.updateStarted(true)
	RaftObj.updateTerm(RaftObj.readTermFromFile().Term)

	go func() {
		for RaftObj.getStart() {
			//time.Sleep(100*time.Nanosecond)
			// log.Println("I am running "+strconv.Itoa(RaftObj.myID))
			switch RaftObj.getState() {
			case Follower: //If it is a Follower then perform this
				RaftObj.performAsFollower()
			case Candidate: //If it is a candidate then perform this
				RaftObj.performAsCandidate()
			case Leader: //If it is a Leader then perform this
				RaftObj.performAsLeader()
			}

		}
	}()

}

// check that am i the leader
func (r *raftState) IsLeader() bool {
	r.roleMtx.RLock()
	term := r.currentState
	r.roleMtx.RUnlock()
	//log.Println(term)
	if term != Leader {
		return false
	}
	return true
}

//get current term of the server
func (r *raftState) Term() int {
	r.termMtx.RLock()
	term := r.currentTerm
	r.termMtx.RUnlock()
	return term
}

//Update term of the server
func (r *raftState) updateTerm(term int) {
	r.termMtx.Lock()
	//log.Println(r.termFilePath)
	r.currentTerm = term
	r.termMtx.Unlock()
	r.writeTerm()

}

//Get the term that is used internally
func (r *raftState) getTerm() int {
	r.termMtx.RLock()
	term := r.currentTerm
	r.termMtx.RUnlock()
	return term
}

//update start variable that kill or wakeup serevr
func (r *raftState) updateStarted(start bool) {
	r.startMtx.Lock()
	r.started = start
	r.startMtx.Unlock()
}

//GET start variable that kill or wakeup serevr
func (r *raftState) getStart() bool {
	r.startMtx.RLock()
	start := r.started
	r.startMtx.RUnlock()
	return start

}

//Miscellaneous Function of Raft
func (RaftObj *raftState) sendVote(req cluster.VoteReq) bool {
	//If older request then current term then REFUSE
	if req.Term <= RaftObj.getTerm() {
		if RaftObj.isDebugOn {
			log.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Vote rejected old term")
		}
		//log.Println("got Negative Old term res term"+strconv.Itoa(req.Term)+" MY term "+strconv.Itoa(RaftObj.currentTerm))
		return false
	}
	//Reject vote request beacause server has older logs
	// if already voted then no need to vote-
	if RaftObj.votedFor != req.IdCandidate && RaftObj.votedFor != 0 && req.Term <= RaftObj.getTerm() {
		if RaftObj.isDebugOn {
			log.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Vote rejected Already Voted")
		}
		//log.Println("got Negative Already Voted res term"+strconv.Itoa(req.Term)+" MY term "+strconv.Itoa(RaftObj.currentTerm))
		return false
	}
	//reject if candidate does not have enough new logs
	if req.LastlogIndex < RaftObj.logWrite.CommittedIndex {
		if RaftObj.isDebugOn {
			log.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Vote rejected becuse i have more logs")
		}
		return false

	}
	if RaftObj.isDebugOn {
		log.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Vote garanted")
	}
	RaftObj.updateTerm(req.Term)
	RaftObj.votedFor = req.IdCandidate
	return true
}

//read-write current save term and commoted index in disk
func (RaftObj *raftState) writeTerm() {
	//log.Println(path)
	//log.Println(x);
	var pre presistState
	pre.Commitedindex = RaftObj.logWrite.CommittedIndex
	pre.Term = RaftObj.getTerm()
	fil, er := os.OpenFile(RaftObj.termFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if er != nil {
		//panic(path)
		panic(er)
	}
	str, _ := json.Marshal(pre)

	fil.WriteString(string(str))
	fil.Close()
}
func (RaftObj *raftState) readTermFromFile() presistState {
	var pres presistState
	fil, er := os.OpenFile(RaftObj.termFilePath, os.O_RDONLY, 0777)
	//log.Println(fil)
	if er != nil {
		return pres
	}

	reader := bufio.NewReader(fil)
	line, _, errr := reader.ReadLine()
	//log.Println(line)
	if errr != nil {
		return pres
	}
	fil.Close()
	json.Unmarshal(line, &pres)
	return pres
}

//Write current term to disk.

//RAFT initializer
type raftState struct {
	currentTerm int //Current term of server

	termMtx         sync.RWMutex           //Mutex For term
	roleMtx         sync.RWMutex           //Mutex for State of server
	startMtx        sync.RWMutex           //mutext for Start
	votedFor        int                    //whom to voted last
	totalVote       int                    //total vote recived when it is candidate
	inputRequest    chan *[]byte           //recvive operation that need to be replicated
	commitThis      chan *cluster.LogValue //after replication return log to KV strore on this chanal
	myID            int                    // ID of sever
	quorum          int                    //majority size
	isDebugOn       bool                   //debug show flag
	currentState    int                    //State Menas Leader or folllwer or candidate
	currentLeader   int                    // who is my leader
	heartBeatTime   int
	electionTimeout int               // What is the election timeout for me
	started         bool              //is server running
	serverEntity    cluster.ServerObj //Servre cluster object
	logFilePath     string            //path to store log files
	termFilePath    string            //path to store term in Dis
	logWrite        raftLog.LogManager
}

func NewRaft(id int, path string) *Raft {
	//Intialising Object Properties
	RaftObj := new(raftState)
	RaftObj.isDebugOn = false
	cofigpath, _ := filepath.Abs(path)
	allDetails := getALLCOnfig(cofigpath)
	RaftObj.commitThis = make(chan *cluster.LogValue, 20)
	RaftObj.inputRequest = make(chan *[]byte, 20)
	//Getting Election Time Onut From COnfig Object
	for index := range allDetails.Timeouts {
		if allDetails.Timeouts[index].ID == id {
			RaftObj.electionTimeout = allDetails.Timeouts[index].ElectionTimeout
		}
		if allDetails.LogFile[index].ID == id {
			RaftObj.logFilePath = allDetails.LogFile[index].Filename
			RaftObj.logFilePath = allDetails.LogFile[index].Filename
			RaftObj.logFilePath = allDetails.LogFile[index].Filename
		}
		if allDetails.TermFile[index].ID == id {
			//	log.Println(allDetails.TermFile[index].Filename)
			RaftObj.termFilePath = allDetails.TermFile[index].Filename
		}

	}
	//log.Println("sdfhkjasdhflkjshadljkfhlkjsadhflkj")
	RaftObj.serverEntity = cluster.New(id, path)
	RaftObj.quorum = (len(RaftObj.serverEntity.Peers_o)+1)/2 + 1
	RaftObj.logWrite.LogState = make([]int, len(RaftObj.serverEntity.Peers_o)+2)
	RaftObj.logWrite.StartLogger(RaftObj.logFilePath, RaftObj.myID)
	getit := RaftObj.readTermFromFile()
	RaftObj.updateTerm(getit.Term)
	RaftObj.logWrite.CommittedIndex = getit.Commitedindex
	RaftObj.logWrite.Deleteafter(RaftObj.logWrite.CommittedIndex)
	for i := 1; i <= len(RaftObj.serverEntity.Peers_o)+1; i++ {
		RaftObj.logWrite.Setindex(i, RaftObj.logWrite.CommittedIndex+1)
		RaftObj.logWrite.LogState[i] = 0
	}
	RaftObj.heartBeatTime = 100
	RaftObj.updateState(Follower)
	RaftObj.writeTerm()
	RaftObj.myID = id
	RaftObj.votedFor = 0
	RaftObj.updateStarted(false)
	RaftObj.votedFor = 0
	if RaftObj.isDebugOn {
		log.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Start -> Follower")
	}
	RaftObj.Start()
	var x Raft
	x = RaftObj
	return &x
}

//Role of the follwer will be handled by this function
func (RaftObj *raftState) performAsFollower() {
	if RaftObj.getState() == Follower {
		RaftObj.totalVote = 0
		select {
		//check for incoming message
		case x := <-RaftObj.serverEntity.Inbox():
			var msg cluster.Envelope
			msg = *x
			switch msg.Msg.(type) {
			case cluster.LogEntry:
				RaftObj.doAppendEntry(x)
			case cluster.VoteReq:
				req := msg.Msg.(cluster.VoteReq)
				//check can we give a positive vote
				if RaftObj.sendVote(req) {
					RaftObj.serverEntity.Outbox() <- &cluster.Envelope{Pid: req.IdCandidate, MsgId: 90, Msg: cluster.VoteRespose{Term: req.Term, VoteResult: true}}
					//log.Println("got postive res term"+strconv.Itoa(req.Term)+" MY term "+strconv.Itoa(RaftObj.currentTerm))
				} else {
					RaftObj.serverEntity.Outbox() <- &cluster.Envelope{Pid: req.IdCandidate, MsgId: 90, Msg: cluster.VoteRespose{Term: req.Term, VoteResult: false}}

				}

			}

		case x := <-RaftObj.inputRequest:
			if RaftObj.IsLeader() {
				RaftObj.logWrite.Entries[RaftObj.logWrite.Getindex(RaftObj.myID)%LogValueAmount] = cluster.LogValue{Operands: *x,
					Term: RaftObj.getTerm(), Index: RaftObj.logWrite.Getindex(RaftObj.myID)}
				RaftObj.logWrite.WriteLog(&RaftObj.logWrite.Entries[RaftObj.logWrite.Getindex(RaftObj.myID)%LogValueAmount])
				RaftObj.logWrite.Setindex(RaftObj.myID, RaftObj.logWrite.Getindex(RaftObj.myID)+1)
				RaftObj.prepareandsendHearbeat()
			}
		//If no msg from any where then Timeout and Become Candidate
		case <-time.After(time.Duration(RaftObj.electionTimeout) * time.Millisecond):
			RaftObj.votedFor = RaftObj.myID
			RaftObj.totalVote = 1
			RaftObj.updateTerm(RaftObj.getTerm() + 1)
			RaftObj.updateState(Candidate)
			//Broadcast the request for vote
			RaftObj.serverEntity.Outbox() <- &cluster.Envelope{Pid: cluster.BROADCAST, MsgId: 90, Msg: cluster.VoteReq{Term: RaftObj.currentTerm, IdCandidate: RaftObj.myID}}
			if RaftObj.isDebugOn {
				log.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + "Timeout Follower->Candidate")
			}
		}

	}

}

//Role of the candidate will be handled by this function
func (RaftObj *raftState) performAsCandidate() {

	if RaftObj.getState() == Candidate {
		select {
		case x := <-RaftObj.serverEntity.Inbox():
			var msg cluster.Envelope
			msg = *x
			switch msg.Msg.(type) {

			case cluster.LogEntry:
				//If gets any info that anyone have higher term or older leader step back to the follower
				if msg.Msg.(cluster.LogEntry).Term >= RaftObj.currentTerm || RaftObj.currentLeader == msg.Msg.(cluster.LogEntry).LeaderId {
					RaftObj.logWrite.Deleteafter(RaftObj.logWrite.CommittedIndex)
					RaftObj.currentLeader = msg.Msg.(cluster.LogEntry).LeaderId
					RaftObj.updateTerm(msg.Msg.(cluster.LogEntry).Term)
					RaftObj.updateState(Follower)
					if RaftObj.isDebugOn {
						log.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Candidate -> Follower")
					}
				}

				//If gets any voteReqThen it will remove refuse all to give vote because it voted for himself
			case cluster.VoteReq:
				req := msg.Msg.(cluster.VoteReq)
				RaftObj.serverEntity.Outbox() <- &cluster.Envelope{Pid: req.IdCandidate, MsgId: 90, Msg: cluster.VoteRespose{Term: msg.Msg.(cluster.VoteReq).Term, VoteResult: false}}

			case cluster.VoteRespose:
				res := msg.Msg.(cluster.VoteRespose)
				//count Positive votes
				if res.Term <= RaftObj.getTerm() && res.VoteResult {

					RaftObj.totalVote++
				}
				//if have enough vote then go on become Leader
				if RaftObj.totalVote > RaftObj.quorum {
					if RaftObj.isDebugOn {
						log.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Candidate -> Leader Votes-" + strconv.Itoa(RaftObj.totalVote))
					}
					RaftObj.updateState(Leader)
					RaftObj.LogSender()

				}
			}
			//Start New Election if there is no HearBeat Msg NO leader and No Majority
		case <-time.After(time.Duration(RaftObj.electionTimeout) * time.Millisecond):
			//Confirm Once Again Before re-election
			if RaftObj.totalVote > RaftObj.quorum {
				if RaftObj.isDebugOn {
					log.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Candidate -> Leader Votes-" + strconv.Itoa(RaftObj.totalVote))
				}
				RaftObj.updateState(Leader)
				RaftObj.LogSender()

			} else {
				if RaftObj.isDebugOn {
					log.Println(strconv.Itoa(RaftObj.myID) + " - Term " + strconv.Itoa(RaftObj.currentTerm) + " Relection Not Having Enough Votes Total Votes " + strconv.Itoa(RaftObj.totalVote))
				}
				RaftObj.updateTerm(RaftObj.getTerm() + 1)
				RaftObj.serverEntity.Outbox() <- &cluster.Envelope{Pid: cluster.BROADCAST, MsgId: 90, Msg: cluster.VoteReq{Term: RaftObj.currentTerm, IdCandidate: RaftObj.myID}}
				RaftObj.votedFor = RaftObj.myID
				RaftObj.currentLeader = 0
				RaftObj.totalVote = 1

			}
		//if a request come then it will only accepted if it is a leader
		case x := <-RaftObj.inputRequest:
			if RaftObj.IsLeader() {
				RaftObj.logWrite.Entries[RaftObj.logWrite.Getindex(RaftObj.myID)%LogValueAmount] = cluster.LogValue{Operands: *x,
					Term: RaftObj.getTerm(), Index: RaftObj.logWrite.Getindex(RaftObj.myID)}
				RaftObj.logWrite.WriteLog(&RaftObj.logWrite.Entries[RaftObj.logWrite.Getindex(RaftObj.myID)%LogValueAmount])
				RaftObj.logWrite.Setindex(RaftObj.myID, RaftObj.logWrite.Getindex(RaftObj.myID)+1)
				RaftObj.prepareandsendHearbeat()
			}

		}
	}
}

//Role of the Leader will be handled by this function
func (RaftObj *raftState) performAsLeader() {
	if RaftObj.getState() == Leader {
		//Broadcast Heat Beat to EveryOne

		select {
		//if any one is Leader with Higher term Then step Back to follower
		case x := <-RaftObj.serverEntity.Inbox():
			var msg cluster.Envelope
			msg = *x
			switch msg.Msg.(type) {
			case cluster.LogEntry:
				if msg.Msg.(cluster.LogEntry).Term > RaftObj.currentTerm {
					RaftObj.updateState(Follower)
					RaftObj.updateTerm(msg.Msg.(cluster.LogEntry).Term)
					if RaftObj.isDebugOn {
						log.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Leader->Follower")
					}
				}
			case cluster.AppendResponse:
				RaftObj.doResponseAction(x)
			}

		case <-time.After(HearBeatTimeout * time.Millisecond):
			RaftObj.serverEntity.Outbox() <- &cluster.Envelope{Pid: cluster.BROADCAST, MsgId: -1, Msg: cluster.LogEntry{
				Term:         RaftObj.currentTerm,
				CommitIndex:  RaftObj.logWrite.CommittedIndex,
				LeaderId:     RaftObj.myID,
				PrevLogindex: RaftObj.logWrite.Entries[(RaftObj.logWrite.Getindex(RaftObj.myID)-1)%LogValueAmount].Index,
				PrevLogTerm:  RaftObj.logWrite.Entries[(RaftObj.logWrite.Getindex(RaftObj.myID)-1)%LogValueAmount].Term}}
		case x := <-RaftObj.inputRequest:

			if RaftObj.IsLeader() {
				RaftObj.logWrite.Entries[RaftObj.logWrite.Getindex(RaftObj.myID)%LogValueAmount] = cluster.LogValue{Operands: *x,
					Term: RaftObj.getTerm(), Index: RaftObj.logWrite.Getindex(RaftObj.myID)}
				RaftObj.logWrite.WriteLog(&RaftObj.logWrite.Entries[RaftObj.logWrite.Getindex(RaftObj.myID)%LogValueAmount])
				RaftObj.logWrite.Setindex(RaftObj.myID, RaftObj.logWrite.Getindex(RaftObj.myID)+1)
				RaftObj.prepareandsendHearbeat()
			}

		}

	}

}

//this will retrive raft configuration information
func getALLCOnfig(path string) raftConfig {
	file, _ := ioutil.ReadFile(path)
	var raftcon raftConfig
	json.Unmarshal(file, &raftcon)
	if len(raftcon.Servers) < 1 {
		panic("RAFT CONFIG IS NOT CORRECT")
	}
	return raftcon
}

//it will commit all log upto the given index but
//that index must be available
func (RaftObj *raftState) commitupto(index int) {
	for index < RaftObj.logWrite.Getindex(RaftObj.myID) && index > RaftObj.logWrite.CommittedIndex {
		RaftObj.logWrite.CommittedIndex++
		RaftObj.commitThis <- &RaftObj.logWrite.Entries[RaftObj.logWrite.CommittedIndex]
		log.Println(RaftObj.myID, "Commited upto", RaftObj.logWrite.CommittedIndex, index, RaftObj.logWrite.Getindex(RaftObj.myID))
		RaftObj.writeTerm()

	}

}

//--------------------------------------------------------------------------
//0-INORDER OK ENTRY
//1-Out of order log ask alll entries after the commited entry
//---------------------------------------------------------------------------
func (RaftObj *raftState) doAppendEntry(msg *cluster.Envelope) {
	hb := msg.Msg.(cluster.LogEntry)
	//If MsgId=-1 then it is a heartbeat
	if msg.MsgId > 0 {
		validation := RaftObj.logWrite.ValidateRequest(*msg, RaftObj.currentTerm, msg.Pid, RaftObj.myID)
		if validation == 0 {
			//in expacted and best excution case this `will execute
			RaftObj.logWrite.Entries[hb.LogCommand.Index%LogValueAmount] = hb.LogCommand
			RaftObj.serverEntity.Outbox() <- &cluster.Envelope{
				Pid: hb.LeaderId, MsgId: hb.LogCommand.Index, Msg: cluster.AppendResponse{Result: true, Reason: 0, RefrenceMsgID: hb.LogCommand.Index,
					ExpectedEntry: RaftObj.logWrite.Getindex(RaftObj.myID)}}
			if hb.CommitIndex > RaftObj.logWrite.CommittedIndex && hb.CommitIndex < RaftObj.logWrite.Getindex(RaftObj.myID) {
				RaftObj.commitupto(hb.CommitIndex)
			}
			RaftObj.currentLeader = hb.LeaderId
			RaftObj.updateTerm(hb.Term)

		} else {
			//if it is not a correct order that follwer expacted then asking to send all entries after the commited index
			RaftObj.serverEntity.Outbox() <- &cluster.Envelope{Pid: hb.LeaderId, MsgId: msg.MsgId, Msg: cluster.AppendResponse{Result: false, RefrenceMsgID: hb.LogCommand.Index, Reason: validation, ExpectedEntry: RaftObj.logWrite.Getindex(RaftObj.myID)}}
		}
	}
	//simple hear beat processing
	if msg.MsgId < 0 {
		if hb.Term > RaftObj.getTerm() {
			RaftObj.currentLeader = hb.LeaderId
			RaftObj.updateTerm(hb.Term)
		}
		RaftObj.commitupto(hb.CommitIndex)
		if RaftObj.isDebugOn {
			log.Println(strconv.Itoa(RaftObj.myID) + " - " + strconv.Itoa(RaftObj.currentTerm) + " Heat Beat Recived")
		}
		//if we have new leader elected then forget the older one
	}
}

//decode the response and setup the next entry that will be sent as per
//the expectation of the follower
func (RaftObj *raftState) doResponseAction(msg *cluster.Envelope) {
	//if it a regular msg by up-to-date follower
	decoded := msg.Msg.(cluster.AppendResponse)
	if msg.Msg.(cluster.AppendResponse).Result {
		RaftObj.logWrite.VoteCount[decoded.RefrenceMsgID%LogValueAmount] = (RaftObj.logWrite.VoteCount[decoded.RefrenceMsgID%LogValueAmount] + 1) % len(RaftObj.serverEntity.Peers_o)
		if RaftObj.logWrite.VoteCount[decoded.RefrenceMsgID%LogValueAmount]%9 == RaftObj.quorum {
			log.Println("I can commit Now Broadcasting to other", RaftObj.logWrite.VoteCount[msg.Msg.(cluster.AppendResponse).RefrenceMsgID%LogValueAmount], RaftObj.logWrite.CommittedIndex)
			RaftObj.logWrite.CommittedIndex++
			RaftObj.commitThis <- &RaftObj.logWrite.Entries[RaftObj.logWrite.CommittedIndex]
			RaftObj.writeTerm()
		}
		RaftObj.logWrite.LogState[msg.Pid] = decoded.Reason
		RaftObj.logWrite.LastAck[msg.Pid] = decoded.RefrenceMsgID
		RaftObj.logWrite.Setindex(msg.Pid, decoded.ExpectedEntry)
		RaftObj.prepareandsendHearbeat()
	} else {
		log.Println("Pending Entry-")
		RaftObj.logWrite.Setindex(msg.Pid, decoded.ExpectedEntry)
		RaftObj.logWrite.LastAck[msg.Pid] = decoded.RefrenceMsgID - 1
		RaftObj.logWrite.LogState[msg.Pid] = decoded.Reason

	}

}

//prepare and send the hear beat
func (RaftObj *raftState) prepareandsendHearbeat() {
	y := cluster.Envelope{Pid: cluster.BROADCAST, MsgId: -1, Msg: cluster.LogEntry{Term: RaftObj.currentTerm,
		CommitIndex:  RaftObj.logWrite.CommittedIndex,
		LeaderId:     RaftObj.myID,
		PrevLogindex: RaftObj.logWrite.Entries[(RaftObj.logWrite.Getindex(RaftObj.myID)-1)%LogValueAmount].Index,
		PrevLogTerm:  RaftObj.logWrite.Entries[(RaftObj.logWrite.Getindex(RaftObj.myID)-1)%LogValueAmount].Term}}
	RaftObj.serverEntity.Outbox() <- &y

}
