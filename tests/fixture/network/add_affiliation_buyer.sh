
#!/bin/bash
fabric-ca-client enroll  -u https://admin:adminpw@ca.testbuyer.com:7054 --tls.certfiles /etc/hyperledger/fabric-ca-server-config/ca.testbuyer.com-cert.pem 
fabric-ca-client affiliation add buyer  -u https://admin:adminpw@ca.testbuyer.com:7054 --tls.certfiles /etc/hyperledger/fabric-ca-server-config/ca.testbuyer.com-cert.pem 
