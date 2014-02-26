package raft

import (
	"log"
	"testing"
	"time"
	//"strconv"
	//"fmt"
)

func TestLEADERELCETION(t *testing.T) {
	all := make(map[int]Raft)
	//Initialize all of the Raft Objects Creating total 10 objects
	for i := 1; i < 11; i++ {
		all[i] = *NewRaft(i, "../raftconfig.json")
	}
	time.Sleep(1 * time.Second)
	//Counting total leaders
	leaderCount := 0
	leaderindex := 0
	for i := 1; i < 11; i++ {
		if all[i].IsLeader() {
			leaderCount += 1
			leaderindex = i
		}

	}
	//if total leader is one then it is ok becasuse there can be atmost one leader
	if leaderCount == 1 {
		log.Println("Only One Leader Found Its ")
	} else {
		log.Panic("No Lader Found or moer then one leader found")
	}
	//Kill 3 Servers that is not leader
	kill := 0
	//Three Follower has been stopped
	for i := 1; i < 11; i++ {
		if !all[i].IsLeader() {
			all[i].Stop()
			kill++
		}
		if kill >= 3 {
			break
		}
	}
	//and also kill one leader and now total running servers are 6
	all[leaderindex].Stop()
	time.Sleep(2 * time.Second)
	//Count Again How many leaders are there
	leaderCount = 0
	for i := 1; i < 11; i++ {
		if all[i].IsLeader() {
			leaderCount += 1
			log.Println(i)
			leaderindex = i
		}

	}
	//if there is only one leader then its ok
	if leaderCount == 1 {
		log.Println("Only One Leader Found Minority Failure  ")
	} else {
		log.Panic("Leader Found More then One or None Fail")
	}
	//Now stop leader also that will lead to 5 server cluster and that is Majority Failure
	all[leaderindex].Stop()
	time.Sleep(2 * time.Second)
	//Now count all leader
	leaderCount = 0
	for i := 1; i < 11; i++ {
		if all[i].IsLeader() {
			leaderCount += 1
			log.Println(i)
			leaderindex = i
		}

	}
	//Here we should not have any leader because minority cant choose leader
	if leaderCount == 0 {
		log.Println("No Lader Found in Majority Failure as it should not be")
	} else {
		log.Panic("Something is wrong how can we get a leder in majority failure")
	}

}
