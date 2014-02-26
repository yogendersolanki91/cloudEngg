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
```sh
    rafter = *NewRaft(ID, "PATH TO CONFIG FILE")
```
please take care of the " * " that is use near "NewRaft".

There is four function that is accessible for the library users.
* IsLeader() - This function will return true when a server is in the state of the Leader.
* Term()- This Function will return the current term of the server. To know more about term see Raft paper.

Two function additionally provided for Debugging the code.
* Start()-It will start the server after killing the server with Stop().
* Stop()- It will stop the all execution process of server and save the current term to disk and become a follower.



License
----

MIT