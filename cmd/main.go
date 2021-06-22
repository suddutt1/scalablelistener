package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/suddutt1/eventlistener/pkg/fabric/util"
)

func main() {
	fmt.Println("Starting fabric-eventlistener tool...")
	defer fmt.Println("fabric-eventlistener. Done...")
	conProfile := flag.String("cp", "./connection-profile.yaml", "Connection profile file path")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		fmt.Println("Invalid number of arguments")
		os.Exit(1)
	}
	action := args[0]
	switch action {
	case "listener":
		if len(args) < 2 {
			flag.Usage()
			fmt.Println("Invalid number of arguments for eventlister action")
			os.Exit(1)
		}
		channelID := args[1]
		listenEventsAndEmit(*conProfile, channelID, *verbose)
	}

}
func listenEventsAndEmit(cpPath, channelID string, verbose bool) {
	fabricClient := util.NewFabricUtil(cpPath, true)
	if fabricClient == nil {
		fmt.Println("Error in initializing the event listerner")
		os.Exit(2)
	}
	//Register a new user
	fabricClient.RegisterUser("blockListener", "Secret!2343", nil)
	fabricClient.EnrollUser("blockListener", "Secret!2343")
	blockDetailsChan := make(chan *util.BlockDetails)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := fabricClient.RegisterBlockLister(channelID, "blockListener", blockDetailsChan)
		if err != nil {
			fmt.Println("Error in creating block listerner")
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case blockDetails := <-blockDetailsChan:
				//Put in queque
				jb, _ := json.MarshalIndent(blockDetails, "", " ")
				fmt.Printf("Block received %s\n", string(jb))
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	time.Sleep(10 * time.Millisecond)
	//Launch the consumer loop

	wg.Wait()
	fmt.Println("Finished execution")
}
