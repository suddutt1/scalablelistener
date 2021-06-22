#!/bin/bash

echo "Building channel for sample" 
export CHANNEL_NAME="sample"



#Register the channel with orderer

. setpeer.sh Buyer peer0
peer channel create -o orderer0.testfabric.net:7050 -c $CHANNEL_NAME -f ./sample.tx --tls true --cafile $ORDERER_CA -t 1000s

# Joining sample for org peers of Buyer


. setpeer.sh Buyer peer0
peer channel join -b $CHANNEL_NAME.block


#Update the anchor peers for org Buyer
. setpeer.sh Buyer peer0
peer channel update -o  orderer0.testfabric.net:7050 -c $CHANNEL_NAME -f ./sampleBuyerMSPAnchor.tx --tls --cafile $ORDERER_CA 
