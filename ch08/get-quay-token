#!/bin/bash

set -e

echo -n "Username: "
read USERNAME
echo -n "Password: "
read -s PASSWORD
echo

TOKEN_JSON=$(curl -s -H "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '
{
    "user": {
        "username": "'"${USERNAME}"'",
        "password": "'"${PASSWORD}"'"
    }
}')

TOKEN=`echo $TOKEN_JSON | awk '{split($0,a,"\""); print a[4]}'`

echo ""
echo "========================================"
echo "Auth Token is: ${TOKEN}"
echo "The following command will assign the token to the expected variable:"
echo "  export QUAY_TOKEN=\"${TOKEN}\""
echo "========================================"
