package util

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric/protoutil"

	"github.com/hyperledger/fabric-protos-go/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

type FabricUtil struct {
	fabsdk                   *fabsdk.FabricSDK
	org, caID, caName        string
	caAdminID, caAdminSecret string
	ctxProvider              context.ClientProvider
	verbose                  bool
	mspClient                *msp.Client
	channelClientMap         map[string]*channel.Client
	ledgerClientMap          map[string]*ledger.Client
}

//Returns new instance of fabric util
func NewFabricUtil(ccPath string, verbose bool) *FabricUtil {
	fu := new(FabricUtil)
	if err := fu.Init(ccPath, verbose); err != nil {
		logger.Errorf("Unable to initialize fabric util %v", err)
		return nil
	}
	return fu
}

func (fu *FabricUtil) Init(ccPath string, verbose bool) error {
	fu.verbose = verbose
	if fu.verbose {
		logger.SetLevel(logrus.DebugLevel)
	}
	configProviders := config.FromFile(ccPath)
	sdk, err := fabsdk.New(configProviders)
	if err != nil {
		logger.Errorf("Failed to create SDK instance: %v", err)
		return err
	}
	fu.fabsdk = sdk
	fu.ctxProvider = fu.fabsdk.Context()
	mspClient, err := msp.New(fu.ctxProvider)
	if err != nil {
		logger.Errorf("MSP client initialization failed %v", err)
		return err
	}
	fu.mspClient = mspClient
	if err := fu.retrieveRegistrarEnrollmentInfo(); err != nil {
		logger.Errorf("Failed to retive ca admin details: %v", err)
		return err
	}
	if !fu.registerAdminUser() {
		logger.Errorf("Failed to enroll admin: %v", err)
		return fmt.Errorf("Admin_EnrollmentFailure")
	}
	fu.channelClientMap = make(map[string]*channel.Client)
	fu.ledgerClientMap = make(map[string]*ledger.Client)
	logger.Info("Registered organization name ", fu.org)
	logger.Info("Registered organization caID ", fu.caID)
	logger.Info("Registered organization ca name ", fu.caName)
	logger.Info("Registered organization ca adminId ", fu.caAdminID)

	return nil
}
func (fu *FabricUtil) EnrollUser(userID, secret string) bool {
	err := fu.mspClient.Enroll(userID, msp.WithSecret(secret))
	if err != nil {
		logger.Errorf("User enrollment failed: %v", err)
		return false
	}
	logger.Infof("User %s enrollment successful", userID)
	return true
}
func (fu *FabricUtil) RegisterUser(userID, secret string, attrs map[string]string) bool {
	err := fu.mspClient.Enroll(userID, msp.WithSecret(secret))
	if err == nil {
		logger.Infof("User %s already enrolled ", userID)
		return true
	}
	regRequest := msp.RegistrationRequest{
		Name:        userID,
		Type:        "client",
		Affiliation: fu.org,
		CAName:      fu.caName,
		Secret:      secret,
	}
	//TODO:Add attrs
	//if attrs != nil {
	//regRequest.Attributes = attrs
	//}
	_, err = fu.mspClient.Register(&regRequest)
	if err != nil {
		logger.Errorf("User registration failed: %v", err)
		return false
	}
	logger.Infof("User %s registration successful", userID)
	return true
}
func (fu *FabricUtil) Query(channelID, ccID, userID, funcName string, ccArgs [][]byte, peers ...string) (int32, []byte, error) {
	chClient := fu.getChannelClient(channelID, userID)
	if chClient == nil {
		logger.Errorf("Unable to create channel client for channel %s user %s", channelID, userID)
		return -2, nil, fmt.Errorf("ChannelClient_Not_Found")
	}
	request := channel.Request{
		ChaincodeID: ccID,
		Fcn:         funcName,
		Args:        ccArgs,
	}

	channelReqOptions := make([]channel.RequestOption, 0)
	if len(peers) > 0 {
		channelReqOptions = append(channelReqOptions, channel.WithTargetEndpoints(peers...))
	}

	response, err := chClient.Query(request, channelReqOptions...)
	if err != nil {
		fmt.Printf("%+v", err)
		logger.Errorf("Error in query on CCID %s Func %s %+v", ccID, funcName, err)
	}
	logger.Infof("CC query output %s Func %s CCStatus %d TxnValidationCode %s", ccID, funcName, response.ChaincodeStatus, response.TxValidationCode)
	return response.ChaincodeStatus, response.Payload, err

}
func (fu *FabricUtil) GetTrxnDetails(channelID string, trxnID string) error {
	lc := fu.getLedgerClient(channelID)
	_, err := lc.QueryTransaction(fab.TransactionID(trxnID))
	if err != nil {
		logger.Errorf("Error in QueryTransaction %+v", err)
		return err
	}

	return nil
}
func (fu *FabricUtil) GetBlockDetails(channelID string, blockNumber uint64) (*BlockDetails, error) {
	lc := fu.getLedgerClient(channelID)
	blockDetails, err := lc.QueryBlock(blockNumber)
	if err != nil {
		logger.Errorf("Error in QueryBlock %+v", err)
		return nil, err
	}
	return fu.decodeBlock(blockDetails), nil
}
func (fu *FabricUtil) GetBlockDetailsWithFilter(channelID string, blockNumber uint64, ccID string, eventID ...string) (*BlockDetails, error) {

	//Get the leger client
	lc := fu.getLedgerClient(channelID)
	//Fetch the block details
	blockDetails, err := lc.QueryBlock(blockNumber)
	if err != nil {
		logger.Errorf("Error in QueryBlock %+v", err)
		return nil, err
	}
	//Append the ncessary filees
	fileters := make([]string, 0)
	if len(ccID) > 0 {
		fileters = append(fileters, ccID)
	}
	if len(eventID) > 0 {
		fileters = append(fileters, eventID...)
	}
	return fu.decodeBlockWithFilter(blockDetails, fileters...), nil
}
func (fu *FabricUtil) decodeBlock(block *common.Block) *BlockDetails {
	var blockStructure BlockDetails
	blockStructure.BlockNumber = block.Header.Number
	blockStructure.BlockHash = fmt.Sprintf("%x", block.Header.GetDataHash())
	blockStructure.PrevHash = fmt.Sprintf("%x", block.Header.GetPreviousHash())

	txnMetaData := block.GetMetadata().GetMetadata()[2]
	logger.Infof("Transaction meta data length %d", len(txnMetaData))
	for index, trxnBytes := range block.GetData().GetData() {
		txnValidationCode := peer.TxValidationCode(txnMetaData[index])
		if peer.TxValidationCode_VALID != txnValidationCode {
			logger.Infof("Transaction is not valid")
			continue
		}
		logger.Infof("Transaction is valid")
		var trxnDetails TransactionDetails
		envelop, err := protoutil.UnmarshalEnvelope(trxnBytes)
		if err != nil {
			continue
		}
		payload, err := protoutil.UnmarshalPayload(envelop.Payload)
		if err != nil {
			continue
		}
		chHeader, err := protoutil.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		if err != nil {
			continue
		}
		trxnDetails.ChannelID = chHeader.ChannelId
		trxnDetails.TrxnID = chHeader.TxId
		trxn, err := protoutil.UnmarshalTransaction(payload.Data)
		if err != nil {
			continue
		}
		//Decode the transactions
		for _, trxnAction := range trxn.GetActions() {

			ccInput, ccAction, err := protoutil.GetPayloads(trxnAction)
			if err != nil {
				continue
			}
			inputDetails, err := protoutil.UnmarshalChaincodeProposalPayload(ccInput.ChaincodeProposalPayload)
			if err != nil {
				continue
			}
			event, _ := protoutil.UnmarshalChaincodeEvents(ccAction.GetEvents())
			if event != nil {
				logger.Debugf("Event name: %s", event.EventName)
				logger.Debugf("Event payload: %s", string(event.Payload))
				evetDetails := CCEventDetails{EventName: event.EventName, EventPayload: string(event.Payload)}
				trxnDetails.EventDetails = &evetDetails
			}
			ccInvokeInputs, err := protoutil.UnmarshalChaincodeInvocationSpec(inputDetails.GetInput())
			if err != nil {
				continue
			}
			trxnDetails.CCID = ccInvokeInputs.ChaincodeSpec.GetChaincodeId().GetName()
			trxnDetails.CCVersion = ccInvokeInputs.ChaincodeSpec.GetChaincodeId().GetVersion()
			args := make([]interface{}, 0)
			for index, arg := range ccInvokeInputs.ChaincodeSpec.Input.Args {
				logger.Debugf("CCID arg Index %d = %s", index, string(arg))
				args = append(args, string(arg))
			}
			trxnDetails.Parameters = args
			logger.Infof("CCID %s CCVersion %s", ccAction.ChaincodeId.Name, ccAction.ChaincodeId.Version)
			logger.Infof("Response message %s", ccAction.Response.Message)
			logger.Infof("Response payload %s", ccAction.Response.Payload)
			logger.Infof("Response status %d", (ccAction.Response.Status))
			trxnDetails.EndorsementResponse = string(ccAction.Response.Payload)
			trxnDetails.EndorsementMsg = ccAction.Response.Message
			trxnDetails.CCStatus = ccAction.Response.Status
			var rwSet rwset.TxReadWriteSet
			err = proto.Unmarshal(ccAction.Results, &rwSet)
			if err != nil {
				logger.Errorf("Blown RWSet")
			}
			for _, nsRwSetLine := range rwSet.NsRwset {
				logger.Debugf("Name_Space %s", nsRwSetLine.GetNamespace())
				var rwSet kvrwset.KVRWSet
				err = proto.Unmarshal(nsRwSetLine.GetRwset(), &rwSet)
				if err != nil {
					logger.Errorf("Blown RWKVSet")
				}
				for _, read := range rwSet.Reads {
					logger.Debugf("ReadSet: Key %s Version %s", read.GetKey(), read.GetVersion().String())
					trxnDetails.AddReadSet(RWSet{NameSpace: nsRwSetLine.GetNamespace(), Key: read.GetKey(), Version: read.GetVersion().String()})
				}
				for _, write := range rwSet.Writes {
					logger.Debugf("WriteSet: Key %s Value %s", write.GetKey(), string(write.Value))
					trxnDetails.AddWriteSet(RWSet{NameSpace: nsRwSetLine.GetNamespace(), Key: write.GetKey(), Value: write.GetValue(), IsDelete: write.GetIsDelete()})
				}
			}
		}
		blockStructure.AddTrxnDetails(trxnDetails)

	}

	return &blockStructure
}

//First filter is ccID, sendond filer is eventID
func (fu *FabricUtil) decodeBlockWithFilter(block *common.Block, filters ...string) *BlockDetails {
	var blockStructure BlockDetails

	blockStructure.BlockNumber = block.Header.Number
	blockStructure.BlockHash = fmt.Sprintf("%x", block.Header.GetDataHash())
	blockStructure.PrevHash = fmt.Sprintf("%x", block.Header.GetPreviousHash())

	txnMetaData := block.GetMetadata().GetMetadata()[2]
	logger.Infof("Transaction meta data length %d", len(txnMetaData))
	ccFilter := false
	eventFilter := false
	targetCCID := ""
	targetEventID := ""
	if len(filters) > 0 {
		ccFilter = true
		targetCCID = filters[0]
	}
	if len(filters) > 1 {
		eventFilter = true
		targetEventID = filters[1]
	}
	for index, trxnBytes := range block.GetData().GetData() {
		txnValidationCode := peer.TxValidationCode(txnMetaData[index])
		if peer.TxValidationCode_VALID != txnValidationCode {
			logger.Infof("Transaction is not valid")
			continue
		}
		logger.Infof("Transaction is valid")
		var trxnDetails TransactionDetails
		envelop, err := protoutil.UnmarshalEnvelope(trxnBytes)
		if err != nil {
			continue
		}
		payload, err := protoutil.UnmarshalPayload(envelop.Payload)
		if err != nil {
			continue
		}
		chHeader, err := protoutil.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		if err != nil {
			continue
		}
		trxnDetails.ChannelID = chHeader.ChannelId
		trxnDetails.TrxnID = chHeader.TxId
		trxn, err := protoutil.UnmarshalTransaction(payload.Data)
		if err != nil {
			continue
		}
		//Decode the transactions
		for _, trxnAction := range trxn.GetActions() {

			ccInput, ccAction, err := protoutil.GetPayloads(trxnAction)
			if err != nil {
				continue
			}
			inputDetails, err := protoutil.UnmarshalChaincodeProposalPayload(ccInput.ChaincodeProposalPayload)
			if err != nil {
				continue
			}

			ccInvokeInputs, err := protoutil.UnmarshalChaincodeInvocationSpec(inputDetails.GetInput())
			if err != nil {
				continue
			}
			if ccFilter && ccInvokeInputs.ChaincodeSpec.GetChaincodeId().GetName() != targetCCID {
				logger.Debugf("CCID Not matching")
				continue
			}
			trxnDetails.CCID = ccInvokeInputs.ChaincodeSpec.GetChaincodeId().GetName()
			trxnDetails.CCVersion = ccInvokeInputs.ChaincodeSpec.GetChaincodeId().GetVersion()

			event, _ := protoutil.UnmarshalChaincodeEvents(ccAction.GetEvents())
			if event != nil && eventFilter && targetEventID == event.EventName {
				logger.Debugf("Event name: %s", event.EventName)
				logger.Debugf("Event payload: %s", string(event.Payload))
				evetDetails := CCEventDetails{EventName: event.EventName, EventPayload: string(event.Payload)}
				trxnDetails.EventDetails = &evetDetails
			}

			args := make([]interface{}, 0)
			for index, arg := range ccInvokeInputs.ChaincodeSpec.Input.Args {
				logger.Debugf("CCID arg Index %d = %s", index, string(arg))
				args = append(args, string(arg))
			}
			trxnDetails.Parameters = args
			logger.Infof("CCID %s CCVersion %s", ccAction.ChaincodeId.Name, ccAction.ChaincodeId.Version)
			logger.Infof("Response message %s", ccAction.Response.Message)
			logger.Infof("Response payload %s", ccAction.Response.Payload)
			logger.Infof("Response status %d", (ccAction.Response.Status))
			trxnDetails.EndorsementResponse = string(ccAction.Response.Payload)
			trxnDetails.EndorsementMsg = ccAction.Response.Message
			trxnDetails.CCStatus = ccAction.Response.Status
			var rwSet rwset.TxReadWriteSet
			err = proto.Unmarshal(ccAction.Results, &rwSet)
			if err != nil {
				logger.Errorf("Blown RWSet")
			}
			for _, nsRwSetLine := range rwSet.NsRwset {
				logger.Debugf("Name_Space %s", nsRwSetLine.GetNamespace())
				var rwSet kvrwset.KVRWSet
				err = proto.Unmarshal(nsRwSetLine.GetRwset(), &rwSet)
				if err != nil {
					logger.Errorf("Blown RWKVSet")
				}
				for _, read := range rwSet.Reads {
					logger.Debugf("ReadSet: Key %s Version %s", read.GetKey(), read.GetVersion().String())
					trxnDetails.AddReadSet(RWSet{NameSpace: nsRwSetLine.GetNamespace(), Key: read.GetKey(), Version: read.GetVersion().String()})
				}
				for _, write := range rwSet.Writes {
					logger.Debugf("WriteSet: Key %s Value %s", write.GetKey(), string(write.Value))
					trxnDetails.AddWriteSet(RWSet{NameSpace: nsRwSetLine.GetNamespace(), Key: write.GetKey(), Value: write.GetValue(), IsDelete: write.GetIsDelete()})
				}
			}
		}
		blockStructure.AddTrxnDetails(trxnDetails)

	}

	return &blockStructure
}

//Executes a transaction
func (fu *FabricUtil) Execute(channelID, ccID, userID, funcName string, ccArgs [][]byte, transientInputMap map[string][]byte, peers ...string) (int32, []byte, error) {
	chClient := fu.getChannelClient(channelID, userID)
	if chClient == nil {
		logger.Errorf("Unable to create channel client for channel %s user %s", channelID, userID)
		return -2, nil, fmt.Errorf("ChannelClient_Not_Found")
	}
	request := channel.Request{
		ChaincodeID: ccID,
		Fcn:         funcName,
		Args:        ccArgs,
	}
	if transientInputMap != nil {
		request.TransientMap = transientInputMap
	}
	channelReqOptions := make([]channel.RequestOption, 0)
	if len(peers) > 0 {
		channelReqOptions = append(channelReqOptions, channel.WithTargetEndpoints(peers...))
	}

	response, err := chClient.Execute(request, channelReqOptions...)
	if err != nil {
		logger.Errorf("Error in executing transaction on CCID %s Func %s %+v", ccID, funcName, err)
	}
	logger.Infof("CC execution output %s Func %s CCStatus %d TxnValidationCode %s", ccID, funcName, response.ChaincodeStatus, response.TxValidationCode)
	return response.ChaincodeStatus, response.Payload, err
}

func (fu *FabricUtil) registerAdminUser() bool {

	err := fu.mspClient.Enroll(fu.caAdminID, msp.WithSecret(fu.caAdminSecret))
	if err != nil {
		logger.Errorf("Admin enrollment failed: %v", err)
		return false
	}
	logger.Info("Admin enrollment successful")
	return true
}

func (fu *FabricUtil) retrieveRegistrarEnrollmentInfo() error {

	ctx, err := fu.ctxProvider()
	if err != nil {
		logger.Errorf("Failed to get context: %v", err)
		return err
	}

	fu.org = ctx.IdentityConfig().Client().Organization
	fu.caID = ctx.EndpointConfig().NetworkConfig().Organizations[fu.org].CertificateAuthorities[0]
	caConfig, ok := ctx.IdentityConfig().CAConfig(fu.caID)
	if !ok {
		logger.Errorf("Failed to get caconfig")
		return fmt.Errorf("No_CAConfig")
	}
	fu.caAdminID = caConfig.Registrar.EnrollID
	fu.caAdminSecret = caConfig.Registrar.EnrollSecret
	fu.caName = caConfig.CAName
	return nil
}

//Returns the chaincode execution status and error object incase of any error
func (fu *FabricUtil) getChannelClient(channelID, userID string) *channel.Client {
	mapKey := fmt.Sprintf("%s_%s", channelID, userID)
	//Check in the cache first
	//TODO: To check if they are live or not
	if client, isFound := fu.channelClientMap[mapKey]; isFound {
		return client
	}
	clientChannelContext := fu.fabsdk.ChannelContext(channelID, fabsdk.WithUser(userID), fabsdk.WithOrg(fu.org))
	// Channel client is used to query and execute transactions (Org1 is default org)
	client, err := channel.New(clientChannelContext)
	if err != nil {
		logger.Errorf("Failed to create new channel client: %s", err)
		return nil
	}
	fu.channelClientMap[mapKey] = client
	return client
}
func (fu *FabricUtil) getLedgerClient(channelID string) *ledger.Client {
	if client, isFound := fu.ledgerClientMap[channelID]; isFound {
		logger.Infof("Returning from ledger client cache: %s", channelID)
		return client
	}
	clientChannelContext := fu.fabsdk.ChannelContext(channelID, fabsdk.WithUser(fu.caAdminID), fabsdk.WithOrg(fu.org))
	// Ledger client is used to query and execute transactions (Org1 is default org)
	client, err := ledger.New(clientChannelContext)
	if err != nil {
		logger.Errorf("Failed to create ledger client: %s", err)
		return nil
	}
	fu.ledgerClientMap[channelID] = client
	return client
}

//BlockDetails represents hyperledger fabric block details
type BlockDetails struct {
	BlockNumber     uint64               `json:"blockNum"`
	PrevHash        string               `json:"prevHash"`
	BlockHash       string               `json:"hash"`
	TransactionList []TransactionDetails `json:"trxnList"`
}

type TransactionDetails struct {
	EventDetails        *CCEventDetails `json:"eventDetails"`
	CCID                string          `json:"ccID"`
	CCVersion           string          `json:"ccVersion"`
	ChannelID           string          `json:"channelID"`
	CCStatus            int32           `json:"ccStatusCode"`
	Parameters          []interface{}   `json:"ccParams"`
	EndorsementResponse string          `json:"ccResponse"`
	EndorsementMsg      string          `json:"ccMessage"`
	TrxnID              string          `json:"trxnID"`
	ReadSet             []RWSet         `json:"reads"`
	WriteSet            []RWSet         `json:"writes"`
}

type CCEventDetails struct {
	EventName    string `json:"eventName"`
	EventPayload string `json:"eventPayload"`
}
type RWSet struct {
	NameSpace string `json:"namespace"`
	Key       string `json:"key"`
	Version   string `json:"version"`
	Value     []byte `json:"value"`
	IsDelete  bool   `json:"isDelete"`
}

func (bd *BlockDetails) AddTrxnDetails(trxn TransactionDetails) {
	if len(bd.TransactionList) == 0 {
		bd.TransactionList = make([]TransactionDetails, 0)
	}
	bd.TransactionList = append(bd.TransactionList, trxn)
}

func (t *TransactionDetails) AddReadSet(entry RWSet) {
	if len(t.ReadSet) == 0 {
		t.ReadSet = make([]RWSet, 0)
	}
	t.ReadSet = append(t.ReadSet, entry)
}
func (t *TransactionDetails) AddWriteSet(entry RWSet) {
	if len(t.WriteSet) == 0 {
		t.WriteSet = make([]RWSet, 0)
	}
	t.WriteSet = append(t.WriteSet, entry)
}
