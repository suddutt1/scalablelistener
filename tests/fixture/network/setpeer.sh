
#!/bin/bash
export ORDERER_CA=/opt/ws/crypto-config/ordererOrganizations/testfabric.net/msp/tlscacerts/tlsca.testfabric.net-cert.pem

#For fabric 2.2.x extra environment variables

export BUYER_PEER0_CA=/opt/ws/crypto-config/peerOrganizations/testbuyer.com/peers/peer0.testbuyer.com/tls/ca.crt



if [ $# -lt 2 ];then
	echo "Usage : . setpeer.sh Buyer| <peerid>"
fi
export peerId=$2

if [[ $1 = "Buyer" ]];then
	echo "Setting to organization Buyer peer "$peerId
	export CORE_PEER_ADDRESS=$peerId.testbuyer.com:7051
	export CORE_PEER_LOCALMSPID=BuyerMSP
	export CORE_PEER_TLS_CERT_FILE=/opt/ws/crypto-config/peerOrganizations/testbuyer.com/peers/$peerId.testbuyer.com/tls/server.crt
	export CORE_PEER_TLS_KEY_FILE=/opt/ws/crypto-config/peerOrganizations/testbuyer.com/peers/$peerId.testbuyer.com/tls/server.key
	export CORE_PEER_TLS_ROOTCERT_FILE=/opt/ws/crypto-config/peerOrganizations/testbuyer.com/peers/$peerId.testbuyer.com/tls/ca.crt
	export CORE_PEER_MSPCONFIGPATH=/opt/ws/crypto-config/peerOrganizations/testbuyer.com/users/Admin@testbuyer.com/msp
fi

	