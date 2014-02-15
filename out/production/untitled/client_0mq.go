package main
import (
	zmq "github.com/pebbe/zmq4"
	"fmt"
	"os"
	"time"
)

func main(){
 if len(os.Args)>1 {
	 var msg
	 fmt.Scanln(&msg)
	 fmt.Println("sending msg " , msg," at port",os.Args[1])


 }


}


