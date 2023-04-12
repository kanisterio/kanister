#!/bin/bash

isCreated=false
isFailed=false
clusterName=$1
snapshotId=$2
action=$3

while [[ $isCreated != true && $isFailed == false ]];
do
	if [ "$action" == "backup" ]; then
        atlas backups snapshots describe ${snapshotId} --clusterName ${clusterName} -o json > output.json
        isCompleted=$(jq -r ".status" output.json)
        if [ $isCompleted == "failed" ]; then
            isFailed=true
        fi
        if [ $isCompleted == "completed" ]; then
            isCreated=true
        fi
    else
        atlas backups restores describe ${snapshotId} --clusterName ${clusterName} -o json > output.json
    	isFinished=$(jq -r ".finishedAt" output.json)
        isRestoreFailed=$(jq -r ".failed" output.json)
        if [ $isRestoreFailed == "true" ]; then
            isFailed=true
        fi
        if [ $isRestoreFailed == "false" ] && [ $isFinished != "null" ]; then
            isCreated=true
	    fi
    fi
    sleep 30
done

if [ $isCreated == true ]; then
    exit 0
else
    exit 1
fi