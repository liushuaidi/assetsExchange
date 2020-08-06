#!/bin/bash
#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# This script is designed to be run in the org3cli container as the
# final step of the EYFN tutorial. It simply issues a couple of
# chaincode requests through the org3 peers to check that org3 was
# properly added to the network previously setup in the tutorial.
#

echo
echo "New org test"
echo
CHANNEL_NAME="$1"
DELAY="$2"
LANGUAGE="$3"
TIMEOUT="$4"
VERBOSE="$5"
: ${CHANNEL_NAME:="mychannel"}
: ${TIMEOUT:="10"}
: ${LANGUAGE:="golang"}
: ${VERBOSE:="false"}
LANGUAGE=`echo "$LANGUAGE" | tr [:upper:] [:lower:]`
COUNTER=1
MAX_RETRY=5

CC_SRC_PATH="github.com/chaincode/assetsExchange/go/"
if [ "$LANGUAGE" = "node" ]; then
	CC_SRC_PATH="/opt/gopath/src/github.com/chaincode/assetsExchange/node/"
fi

echo "Channel name : "$CHANNEL_NAME

# import functions
. scripts/utils.sh

# Query chaincode on peer0.org3, check if the result
echo "Querying chaincode on peer0.org3..."
chaincodeQuery 0 3

echo "Querying chaincode on peer0.org2..."
chaincodeQuery 0 2

echo "Querying chaincode on peer0.org1..."
chaincodeQuery 0 1


echo
echo "========= All GOOD, execution completed =========== "
echo


exit 0
