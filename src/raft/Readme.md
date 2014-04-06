Raft
=========

Raft is a famous consensus algorithm and this the implementation of that algorithm. It will be implemented stepwise in current status it is a Leader Election algorithm. It is very easy to use library and it is written with easy to understand code.

What is the roles of server in Raft??
----------------------------------
There is three different role that can be assigned to server

* Leader
* Candidate
* Follower


There can be at most one leader at a time. If there is majority number of server is not available then it does not guarantee of having leader.


How to use
--------------
To use the Raft you have to write just one line of code
* To initialize

```sh
    rafter = *NewRaft(ID, "PATH TO CONFIG FILE")

```
* Whenever a new request came just encode that with GOB and pass encoded byte array to

```sh
    rafter.Inbox()<-[]byte

```
* If rafter.Outbox() receive any entry meant that has been replicated and client must commit operation that is given in the LogValue must be performed
please take care of the " * " that is use near "NewRaft".

There is four function that is accessible for the library users.

* IsLeader() - This function will return true when a server is in the state of the Leader.

* Term()- This Function will return the current term of the server. To know more about term see Raft paper.

Example is given in the file named RaftDummy.go.

Additional Function
---------------------
Function to simulate the partitioned network behaviour.These function will not kill the process but cut off the communication.

* Start()-It will start the server after killing the server with Stop().
* Stop()- It will stop the all execution process of server and save the current term to disk and become a follower.

Function to switch the log messages.

* DebugOn() - It will show detailed log message while running that can be useful for debugging
* DebugOff()- It will switch off log messages.

Testing Instruction (only for test)
--------------------
Some changes that required while running test cases.
* First the path of the config file and the RaftDummy.go file must be given while running test case on your system.RaftDummy.go is a file that start a dummy process and that can communicate with the test process.So the line No. 96 in raft_dummyprocess_test.go should contain path according to your system to run test successfully. 
* Line No 96:raft_dummyProcess_test.go

    ```sh
    cmd := exec.Command("go", "run", "path to RaftDummy.go", "-id", strconv.Itoa(key+1), 
    "-dbgport", strconv.Itoa(dbg[key]))
    ```
* Line No 72:raftDummy.go
    
    ```sh
        raftVal = *raft.NewRaft(*id, "path to config file")
    ```

* To test the package in correct way you must use this command
```sh
    go test -v -run=^TestCrashFollower$ raft //for testing the case when a server crashed and recover
    go test -v -run=^TestBasicLogReplication$ raft //for testing case the when everything running correctly
```
Don't use "go test raft" directly to test all case at once
You must change line no 100,136,146 to make test run your system must replace these line with absolute path executable of RaftDummy

License
----

MIT
