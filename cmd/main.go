package main

import (
	"fmt"

	"github.com/suddutt1/eventlistener/pkg/fabric/util"
)

func main() {
	fmt.Println("Starting fabric-eventlistener tool...")
	defer fmt.Println("fabric-eventlistener. Done...")
	util.NewFabricUtil("", true)
}
