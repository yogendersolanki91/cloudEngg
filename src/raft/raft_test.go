package raft

import (
	"net/rpc"
	"testing"
	//"os/exec"
	"log"
	"os/exec"
	"strconv"
	"time"
	"reflect"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

func checkleaders(count *int, index *int) {
	var res bool
	for i := 0; i < totalClient; i++ {
		if !killed[i] {

			if isLeader(dbg[i], &res) {
				//fmt.Println(dbg[i])
				*index = i
				*count++
			}
		}
	}
}

const (
	totalClient = 5
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
var killed [5]bool
var dbg [5]int

func TestBasicLogReplication(t *testing.T) {
	baseDbgPort := 1910
	//var dbg [totalClient]int{19910, 19911, 19912, 19913, 19914, 19915, 19916, 19917, 19918, 19919}
	log.Println("Starting total 5 server as given in the config file...")
	for f := 0; f < totalClient; f++ {
		killed[f] = false
		dbg[f] = baseDbgPort + f
	}
	//var s int;
	res = true
	for key, _ := range dbg {

		cmd := exec.Command("C:\\CloudSrc\\src\\RaftDummy.exe", "-id", strconv.Itoa(key+1), "-dbgport", strconv.Itoa(dbg[key]))
		cmd.Start()
		log.Println("Server ",key,"Started")

	}
	time.Sleep(15*time.Second)
	//Basic Test-Alsmost One leader at a time

	log.Println("All server started.Running Basic Replication")
	for _, value := range dbg {
		log.Println("Killed",value)
		Kill(value)
	}
	time.Sleep(10*time.Second)
	if !check(totalClient){
		t.Error("not passed unmached entry found")
	}
	time.Sleep(5*time.Second)
	//log.Println("Running minority failure test")
	//MinorityFailure(t)
	//log.Println("Running Majority failure test")
	//MajorityFailure(t)

}
func TestCrashFollower(t *testing.T){
	baseDbgPort := 1910
	//var dbg [totalClient]int{19910, 19911, 19912, 19913, 19914, 19915, 19916, 19917, 19918, 19919}
	log.Println("Starting total 5 server as given in the config file...")
	for f := 0; f < totalClient; f++ {
		killed[f] = false
		dbg[f] = baseDbgPort + f
	}
	//var s int;
	res = true
	for key, _ := range dbg {
		cmd := exec.Command("C:\\CloudSrc\\src\\RaftDummy.exe", "-id", strconv.Itoa(key+1), "-dbgport", strconv.Itoa(dbg[key]))
		cmd.Start()
		log.Println("Server ",key,"Started")

	}
	time.Sleep(5*time.Second)
	log.Println("Kiling 4")
 	Kill(dbg[3])
	time.Sleep(time.Second*5)
	log.Println("Now legged server started..")
	cmd := exec.Command("C:\\CloudSrc\\src\\RaftDummy.exe", "-id", strconv.Itoa(4), "-dbgport", strconv.Itoa(dbg[3]))
	cmd.Start()
	time.Sleep(25*time.Second)
	for	_, value := range dbg {
		log.Println("Killed",value)
		Kill(value)
	}
	time.Sleep(5*time.Second)
	if !check(totalClient){
		t.Error("not passed")
	}
}

func check (count int)bool{
	var allDB [6]*leveldb.DB
	var allDBMAP [6]map[string]string
	var allIterator [6]iterator.Iterator
	for i:=1;i<=count;i++{
		allDBMAP[i]=make(map[string]string)
		allD,err:=leveldb.OpenFile("db_"+strconv.Itoa(i)+".db",nil)
		if err!=nil{
			panic(err)
		}else{
			allDB[i]=allD
			allIterator[i]=allD.NewIterator(nil,nil)
		}
	}



	for k:=1;k<=count ;k++{
		for allIterator[k].Next() {

			allDBMAP[k][string(allIterator[k].Key())]=string(allIterator[k].Value())

		}

	}
	for k:=1;k<count;k++{
		if reflect.DeepEqual(allDBMAP[k],allDBMAP[k+1]){
			log.Println(reflect.DeepEqual(allDBMAP[k],allDBMAP[k+1]),k)
		}else{
			return false
		}
	}
	return true
	


}
