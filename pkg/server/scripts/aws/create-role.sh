#!/bin/sh

ACCOUNT_ID=$1
AWS_PROFILE=$2
TRUST_POLICY="scripts/aws/TRUST_POLICY_${ACCOUNT_ID}.json"
PERMISSION_POLICY="scripts/aws/PERMISSION_POLICY.json"
ROLE_NAME="ROLE-${ACCOUNT_ID}"
POLICY_NAME="POLICY-${ACCOUNT_ID}"

echo "Creating trust policy"
/bin/cat > $TRUST_POLICY <<EOL
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "${ACCOUNT_ID}"
            },
            "Action": "sts:AssumeRole"
        }
    ]
}
EOL

echo "Checking for the existence of a role"
GET_ROLE_RESULT=`aws --profile $AWS_PROFILE iam get-role --role-name $ROLE_NAME --output text`
echo $GET_ROLE_RESULT

if echo $GET_ROLE_RESULT | grep -q $ROLE_NAME; then
    echo "ROLE EXISTS"
else
    echo "ROLE DOES NOT EXIST"
    echo "Creating role"
    CREATE_ROLE_RESULT=`aws --profile $AWS_PROFILE iam create-role --role-name $ROLE_NAME --assume-role-policy-document file://$TRUST_POLICY --output text`
    echo $CREATE_ROLE_RESULT

    echo "Creating inline permissions policy"
    CREATE_PERMISSIONS_RESULT=`aws --profile $AWS_PROFILE iam put-role-policy --role-name $ROLE_NAME --policy-name $POLICY_NAME --policy-document file://$PERMISSION_POLICY --output text`
    echo $CREATE_PERMISSIONS_RESULT
fi

# cleanup
rm -f $TRUST_POLICY

