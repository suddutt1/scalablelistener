package tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/suddutt1/eventlistener/pkg/fabric/util"
)

func Test_Blockinjector(t *testing.T) {
	t.Skip()
	//This function creates a set of transaction in a test network
	fabricUtl := util.NewFabricUtil("./fixture/connection-profile-buyer.yaml", true)
	if fabricUtl == nil {
		t.Logf("Error in intialization")
		t.FailNow()
	}
	if !fabricUtl.RegisterUser("suddutt6", "myS!perSecret", nil) {
		t.Logf("Failed to register")
		t.FailNow()
	}
	if !fabricUtl.EnrollUser("suddutt6", "myS!perSecret") {
		t.Logf("Failed to enroll")
		t.FailNow()
	}
	key := fmt.Sprintf("Key_%d", time.Now().Unix()%512)
	value := fmt.Sprintf("%d", time.Now().Unix())
	args := [][]byte{
		[]byte(key), []byte(value),
	}
	code, _, err := fabricUtl.Execute("sample", "samplecc", "suddutt6", "SaveKV", args, nil)
	if err != nil || code != 200 {
		t.FailNow()
	}
	//t.Logf("Response text %s", string(payload))
	code, payload, err := fabricUtl.Query("sample", "samplecc", "suddutt6", "GetKV", args)
	if err != nil || code != 200 {
		t.FailNow()
	}

	t.Logf("Response text %s", string(payload))
}
func Test_BlockRead(t *testing.T) {
	//t.Skip()
	//This function creates a set of transaction in a test network
	fabricUtl := util.NewFabricUtil("./fixture/connection-profile-buyer.yaml", true)
	if fabricUtl == nil {
		t.Logf("Error in intialization")
		t.FailNow()
	}
	if !fabricUtl.RegisterUser("suddutt6", "myS!perSecret", nil) {
		t.Logf("Failed to register")
		t.FailNow()
	}
	blkDetails, err := fabricUtl.GetBlockDetails("sample", uint64(23))
	if err != nil {
		t.Logf("Failed to get the block details")
		t.FailNow()
	}
	jb, _ := json.MarshalIndent(blkDetails, "", " ")
	t.Logf("BlockDetails: \n%s", string(jb))
	//t.Logf("BlockDetails: \n%s",blk)

}
