
#!/bin/bash

export VERSION=2.2.0
export ARCH=$(echo "$(uname -s|tr '[:upper:]' '[:lower:]'|sed 's/mingw64_nt.*/windows/')-$(uname -m | sed 's/x86_64/amd64/g')" | awk '{print tolower($0)}')
echo "===> Downloading platform binaries"
export URL="https://github.com/hyperledger/fabric/releases/download/v${VERSION}/hyperledger-fabric-${ARCH}-${VERSION}.tar.gz"
echo $URL
curl  -L $URL| tar xz

