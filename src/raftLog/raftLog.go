package raftLog

import (
	"os"
	"cluster"
	"encoding/json"
	"bufio"
	"sync"
	"log"
)
const (
	LogValueAmount = 500
)
type LogManager struct {
	File          *os.File  //File handle to the log file
	Path          string //path to log file
	indexMtx       sync.RWMutex //mutex to lock index of all
	Entries       []cluster.LogValue //in memory log entry to reduce disk access
	VoteCount     []int //vote counter for each log
	CommittedIndex int //current committed index
	LogState      []int //Log is ok or not
	LastAck			map[int]int // last msh that is recived by the follower
	nexttosend    map[int]int // expacted entry of each follower
}
//get an entry from disk if it is not available in memory
func (r *LogManager)getEntry(index int)cluster.LogValue{
	var readEntry cluster.LogValue
	inputFile, err := os.Open(r.Path)
	if err != nil {
		return readEntry
	}
	defer inputFile.Close()
	reader := bufio.NewReader(inputFile)
	for {
		temp, _, errr := reader.ReadLine()
		if errr!=nil && index==0{
			var readEntry2 cluster.LogValue
			return readEntry2
		}
		json.Unmarshal(temp,&readEntry)
		if readEntry.Index==index{
			return readEntry
		}
	}
	log.Println("Inconsistent log requested entry do not exist",index)
	panic("incososoososos")
}
//write entry to the disk
func (r *LogManager) WriteLog(m *cluster.LogValue) {
	text, _ := json.Marshal(m)
	f, err := os.OpenFile(r.Path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0660)
	if err != nil {
		panic(err)
	}

	if _, err = f.WriteString(string(text) + "\n"); err != nil {
		panic(err)
	}
	f.Close()
}
//intialinze the log manager
func (r *LogManager) StartLogger(path string,index int) {
	r.Entries = make([]cluster.LogValue, LogValueAmount)
	r.VoteCount = make([]int,LogValueAmount)
	r.nexttosend=make(map[int]int)
	r.LastAck=make(map[int]int)
	r.Path = path

	r.IntializeInMemoryEntry(index);
}
//delete all entry that is exist after given index
func(LogObj *LogManager) Deleteafter(index int){

	tFile, _ := os.OpenFile(LogObj.Path+"_temp.txt",os.O_CREATE|os.O_WRONLY,660)
	inputFile, _ := os.OpenFile(LogObj.Path,os.O_RDONLY,660)
	i:=1
	reader := bufio.NewReader(inputFile)
		for i <= index {
			line, _, errr := reader.ReadLine()
			if errr != nil {
				log.Println(errr)
			}

			tFile.WriteString(string(line) + "\r\n")
			i++
		}
		inputFile.Close()
		tFile.Close()
		os.Remove(LogObj.Path)
		os.Rename(LogObj.Path+"_temp.txt", LogObj.Path)

}
//Get first entry stored in the log file
func (r *LogManager) GetOldestEntry() (int, int) {
	var readEntry cluster.LogValue
	inputFile, err := os.Open(r.Path)
	if err != nil {
		return 0, 0
	}
	defer inputFile.Close()
	reader := bufio.NewReader(inputFile)
	line, _, errr := reader.ReadLine()
	if errr != nil {
		return 0, 0
	} else {
		errrr := json.Unmarshal(line, &readEntry)
		if errrr != nil {
			panic(errrr)
		} else {
			return readEntry.Index, readEntry.Term
		}
	}
	return 0, 0
}
//When Raft start put all entry in memory
func (r *LogManager) IntializeInMemoryEntry(index int){
	var readEntry cluster.LogValue
	isEntry:=true
	inputFile, err := os.Open(r.Path)
	i:=1;
	if err != nil {
		isEntry=false;
	}
	defer inputFile.Close()
	reader := bufio.NewReader(inputFile)
	for isEntry {
		temp, _, errr := reader.ReadLine()
		if errr != nil {
			r.Setindex(index,i)
			break;
		} else {
			json.Unmarshal(temp,&readEntry)
			r.Entries[readEntry.Index%LogValueAmount]=readEntry;
			i++;
		}
	}
}
//update index of a follower
func(LogObj *LogManager)Setindex(key int,index int){
	LogObj.indexMtx.Lock();
	LogObj.nexttosend[key]=index
	LogObj.indexMtx.Unlock()

}
//get index of a follower
func (LogObj *LogManager)Getindex(key int)int{
	LogObj.indexMtx.Lock();
	defer LogObj.indexMtx.Unlock()
	return LogObj.nexttosend[key]
}
//get entry by index if it is not correct in memory then retrive from disk
//it make sure that it will never send a wrong entry
func (LogObj *LogManager)PrepareEntryToSend(index int,leaderterm int,leaderid int)cluster.LogEntry{
	var logtosend cluster.LogEntry
	logtosend.CommitIndex = LogObj.CommittedIndex
	logtosend.Term = leaderterm
	logtosend.LeaderId = leaderid
	logtosend.PrevLogindex = LogObj.Entries[(index-1)%LogValueAmount].Index
	logtosend.PrevLogTerm = LogObj.Entries[(index-1)%LogValueAmount].Term
	logtosend.LogCommand = LogObj.Entries[index%LogValueAmount]
	if logtosend.LogCommand.Index!=index{
		var preentry cluster.LogValue;
		logtosend.LogCommand=LogObj.getEntry(index)
		preentry=LogObj.getEntry(index-1);
		logtosend.PrevLogindex=preentry.Index;
		logtosend.PrevLogTerm=preentry.Term
		//LogObj.Entries[index%LogValueAmount]=logtosend.LogCommand;
	}
	return logtosend
}
//match the entry acceptance criteria
func(LogObj *LogManager)ValidateRequest(msg cluster.Envelope,term int,leaderid int,id int) int {
	currentindex:=(LogObj.Getindex(id))
	hb := msg.Msg.(cluster.LogEntry)
	if hb.LogCommand.Index == currentindex && hb.Term >= term &&
		hb.PrevLogindex == LogObj.Entries[(currentindex-1)%LogValueAmount].Index &&
		hb.PrevLogTerm == LogObj.Entries[(currentindex-1)%LogValueAmount].Term {
		LogObj.WriteLog(&hb.LogCommand)

		log.Println(id,"Accpeted ",hb.LogCommand.Index)
		LogObj.Setindex(id,hb.LogCommand.Index+1);
		return 0;
	}else{
		//RaftObj.allIndex[RaftObj.myID]=RaftObj.allIndex[RaftObj.myID];
		log.Println(id,": 1 Failed",hb.LogCommand.Index ," != " ,currentindex)
		LogObj.Setindex(id,LogObj.CommittedIndex+1);
		return 1

	}
	return 0
}
