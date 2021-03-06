
#!/bin/sh
export CHANNEL_NAME="sample"
export CC_NAME="samplecc"
export CC_VERSION="2.0"
export CC_SEQ=2

# Retrive last version 
. setpeer.sh  Buyer peer0

peer lifecycle chaincode querycommitted --channelID $CHANNEL_NAME --name ${CC_NAME} >&clog.txt
export SEQ=$(tail -n 1 clog.txt | awk -F ',' '{ print $2 }'| awk -F ':' '{print $2}')
export NEWSEQ=$((SEQ+1))
export CC_VERSION="${NEWSEQ}.0"
export CC_SEQ=$NEWSEQ
echo "Going to upgrade ${CC_NAME} on ${CHANNEL_NAME} to version ${CC_VERSION} sequence ${CC_SEQ} "


. setpeer.sh  Buyer peer0

# Package the chaincode 
cd chaincode/github.com/samplecc
peer lifecycle chaincode package /opt/ws/${CC_NAME}.tar.gz  --path .  --lang golang --label ${CC_NAME}_${CC_VERSION}
cd /opt/ws



# Install the chaincode package 
. setpeer.sh Buyer peer0
peer lifecycle chaincode install ${CC_NAME}.tar.gz



# Approve Organization Buyer 

. setpeer.sh Buyer peer0
peer lifecycle chaincode queryinstalled >&cqBuyerlog.txt
PACKAGE_ID=$(sed -n "/${CC_NAME}_${CC_VERSION}/{s/^Package ID: //; s/, Label:.*$//; p;}" cqBuyerlog.txt)
echo $PACKAGE_ID
peer lifecycle chaincode approveformyorg -o orderer0.testfabric.net:7050  --tls --cafile $ORDERER_CA --channelID $CHANNEL_NAME --name $CC_NAME --version $CC_VERSION --package-id $PACKAGE_ID --sequence $CC_SEQ --init-required  --signature-policy "  OR( 'BuyerMSP.member')  " 



#Commit chaincode installation 

export PEER_CONN=" --peerAddresses peer0.testbuyer.com:7051 --tlsRootCertFiles ${BUYER_PEER0_CA} "
peer lifecycle chaincode commit -o orderer0.testfabric.net:7050  --tls --cafile $ORDERER_CA --channelID $CHANNEL_NAME --name $CC_NAME --version $CC_VERSION --sequence $CC_SEQ --init-required --signature-policy "  OR( 'BuyerMSP.member') " $PEER_CONN



#Query commited in Org Buyer
. setpeer.sh Buyer peer0
peer lifecycle chaincode querycommitted --channelID $CHANNEL_NAME --name ${CC_NAME}


sleep 2
#Invoke init
peer chaincode invoke -o orderer0.testfabric.net:7050  --tls --cafile $ORDERER_CA -C $CHANNEL_NAME -n ${CC_NAME} $PEER_CONN --isInit -c '{"Args":[""]}'


