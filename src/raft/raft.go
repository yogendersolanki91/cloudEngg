package main

import (
	"cluster"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

type Raft struct {
	currentTerm int;
	votedFor int

}

type timeouts struct {
	ID             int `json:"ID"`
	ElctionTimeout int `json:"ElctionTimeout"`
	HeartBeat      int `json:"HeartBeat"`
}

type RaftConfig struct {
	Servers  []cluster.ServerConf `json:"Servers"`
	Timeouts []timeouts           `json:"Timeouts"`
	LogFile  string               `json:"LogFile"`
}

func main() {
	cofigpath, _ := filepath.Abs("/home/blackh/IdeaProjects/untitled/src/raftconfig.json")
	println(cofigpath)
	file, _ := ioutil.ReadFile(cofigpath)
	//fmt.Println(file)
	var raftcon RaftConfig
	json.Unmarshal(file, &raftcon)
	fmt.Println(raftcon.Timeouts)

}
