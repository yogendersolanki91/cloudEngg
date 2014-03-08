package raft


import (
	"testing"
	"net/rpc"
	//"os/exec"
	"strconv"
	"time"
	"log"
	"os/exec"
)

func checkleaders(count *int,index *int){
	var res bool;
	for i := 0; i < 10; i++ {
		if !killed[i] {

			if isLeader(dbg[i], &res) {
				//fmt.Println(dbg[i])
				*index=i;
				*count++;
			}
		}
	}
}

const (
	totalClient=10;
)
func isLeader(port int, res *bool) bool {
	client, err := rpc.Dial("tcp", "localhost:"+strconv.Itoa(port))
	var pair bool
	var result bool
	result = true
	if err == nil {
		err2 := client.Call("Comm.IsLeaderRPC", pair, &result)
		if err2 == nil {
			res = &result
		} else {
			log.Panic(err2)
		}
	} else {
		log.Panic(err)
	}
	client.Close()

	return result
}
func Term(port int, res *int) int {
	var pair bool
	*res = 3
	client, err := rpc.Dial("tcp", "localhost:"+strconv.Itoa(port))
	if err != nil {
		log.Panic(err)
	}
	er2 := client.Call("Comm.GetTermRPC", pair, res)
	if er2 != nil {
		log.Panic(er2)
	}
	client.Close()
	log.Println(*res)
	return *res
}
func Kill(port int) {
	client, err := rpc.Dial("tcp", "localhost:"+strconv.Itoa(port))
	pair := true
	var res int
	if err != nil {

		log.Panic(err)
	}
	er2 := client.Call("Comm.KillRPC", pair, &res)
	if er2 != nil {
		log.Panic(er2)
	}
	client.Close()
}
var res bool
var leaderPort int
var leaderCount int
var killed [10]bool
var dbg [10]int
func TestBasicLeader(t *testing.T) {

	baseDbgPort:=19910
	//var dbg [totalClient]int{19910, 19911, 19912, 19913, 19914, 19915, 19916, 19917, 19918, 19919}
	log.Println("Starting total 10 server as given in the config file...")
	for f := 0; f < totalClient; f++ {
		killed[f] = false
		dbg[f]=baseDbgPort+f;
	}
	//var s int;
	res = true
	for key, _ := range dbg {
		cmd := exec.Command("go", "run", "/home/blackh/IdeaProjects/untitled/src/RaftDummy.go", "-id", strconv.Itoa(key+1), "-dbgport", strconv.Itoa(dbg[key]))
		cmd.Start()
	}
	time.Sleep(3 * time.Second)
	//Basic Test-Alsmost One leader at a time

	log.Println("All server started.Running Basic Leader Election Test")
	checkleaders(&leaderCount,&leaderPort);
	log.Println("All server started.Running Basic Leader Election Test")
	if leaderCount==1 {
		log.Println("Leader Basic test completed sucsefully")
	}else{
		if leaderCount>1{
		t.Error("Basic leader election test failed: Due to more then one leader founfd")}else{
			t.Error("Basic leader election test failed: Due to no leader found after enought amount of time")
		}
	}
	log.Println("Running minority failure test")
	MinorityFailure(t);
	log.Println("Running Majority failure test")
	MajorityFailure(t);


}
func MinorityFailure(tester *testing.T){
	for j := 0; j < totalClient/2-2; {
		if !killed[j] {
			if !isLeader(dbg[j], &res) {
				killed[j] = true
				Kill(dbg[j])
				j++
			}
		}
	}
	leaderCount = 0
	Kill(dbg[leaderPort])
	killed[leaderPort] = true
	log.Println("Killed "+strconv.Itoa(totalClient/2-1)+" server including the leader..")
	time.Sleep(1 * time.Second)
	checkleaders(&leaderCount,&leaderPort);
	if !killed[leaderPort] && leaderCount == 1 {
		log.Println("Minority fail test is OK")
	}else{
		tester.Error("Minority test failed")
	}



}
func MajorityFailure(tester *testing.T){
	log.Println("Killing leader..")
	Kill(dbg[leaderPort])
	killed[leaderPort] = true
	leaderCount=0;
	checkleaders(&leaderCount,&leaderPort);

	if leaderCount==0 {
		log.Println("Majority test pass no leader choosen")
	}else{
			tester.Error("Majority Leader Fail")
	}
	log.Println("Killing all proceess")
	for j := 0; j < 10;j++ {
		if !killed[j] {
			Kill(dbg[j])
		}
	}
}
