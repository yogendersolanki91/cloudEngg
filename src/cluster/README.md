Cluster for CLOUD
=========

This assignment is the component of a cloud

  - It can Communicate with all peer server
  - It can broadcast the message
  - Easy to use simple library

This is the part of the course assignment that will lead to cloud in final stage.It is the cluster library for that cloud.It is build with Go language that is using the ZeroMQ for communicating over the network between the peer server on the same cluster.

This library is depended on the config.json file that is required to initialise cluster nodes,the file must be properly formatted as showed below an example:
* "ID"-integer ID for the server (Must be unique)

* "Host"-String address/IP of the server  

* "Port"-Port on which server will listen

```ssh
{
"Servers":
    [
    {"ID":1,"Host":"localhost","Port":290001},
    ]
}
```

above format must be followed otherwise server will not start at all.  

Testing
--------------
* To test the package u must have config.json file and package folder in the src directory.Other than that ZeroMq package must be available on the src directory.

```ssh
        src/cluster/cluster_0mq.go
        src/cluster/cluster_omq_test.go
        src/config.json
        src/github.com/pebbe/*.*

```
* Use go test -v in cluster folder to view log and test.It will show no of message broadcast and peer to peer message.

Installation and Use
--------------
* To use this library just download the cluster.go and create a folder name "cluster" in the src director and just use

```sh
import "cluster"
```
* To add New server with the id=2 

```sh
newserv=cluster.New(2,"absolute path to config file")
```
* Other Property such as peer servers can be accessed by the function that are provide for a ServerObj.The property peer_o contains info about all the peer server.

* To check inbox you can use access by Inbox() chanel.For example

```ssh
    for {
    	select {
		case envelope := <-myserver.Inbox():
        ///whenerve a msg arrive u can process that msg here
		case <-time.After(10 * time.Second):
			println("Waited and waited. Ab thak gaya   1\n")
		}
	}
```

* Similarly you send msg by chanel outbox() and to broadcast a msg just use pid=-1.For Example

```ssh
    myserver.Outbox() <- &Envelope{Pid: k, MsgId: l, Msg: "chk msg"}
```

Improved
----
* XML marshaling replaced with GOB encoder because of message drops
* Data of any type can be send by the chanel.Bug removed.
* Bug in the broadcast msg .Bug Recovered.
* Bug in the msg Pid has been recovered.


Version
----

1.2


License
----

MIT

    
