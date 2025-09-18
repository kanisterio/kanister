#!/bin/sh
set -o errexit

echo "Monitoring process ${PID}"

get_consumer_lag() {
    consumer_groups=$(/bin/kafka-consumer-groups  --command-config /tmp/config/kafkaConfig.properties --bootstrap-server "$BOOTSTRAPSERVER" --describe --group "$GROUP")
    if grep -q 'does not exist' <<< $consumer_groups
    then
        echo "Connector consumer group ${GROUP} does not exist"
        return 1
    fi
    if grep -q 'failed' <<< $consumer_groups
    then
        echo "Failed to fetch consumer groups"
        return 1
    fi
    awk 'BEGIN{maxLag=   0}{if ($6>0+maxLag) maxLag=$6} END{print maxLag}' <<< $consumer_groups
}

cleanup() {
    curl -X DELETE http://localhost:8083/connectors/$CONNECTORNAME
    echo "================Connector Deleted======================"
    kill $PID
}

echo "===========================Monitoring started==================================="
sleep 60
export MAXLAG=0
export GROUP="connect-${CONNECTORNAME}"
#export pid="$(lsof -t -i:8083)"
export ELAPSEDTIME="$(ps -e -o "pid,etimes,command" |awk -v processid=$PID '{if($1==processid) print $2}')"

if ! LAG="$(get_consumer_lag)"; then
    echo "Error getting lag"
    cleanup
    exit 1
fi

echo "==========================GROUP=$GROUP, MaxTime=$TIMEINSECONDS, MAXLAG=$MAXLAG, pid=$PID, ELAPSEDTIME=$ELAPSEDTIME, LAG=$LAG, ==============================="
if [ $LAG = "LAG" ]
then
    LAG=0
fi

while [[ "$LAG" -gt "$MAXLAG"  && "$ELAPSEDTIME" -lt "$TIMEINSECONDS" ]]
do

if ! LAG="$(get_consumer_lag)"; then
    echo "Error getting lag."
    cleanup
    exit 1
fi

ELAPSEDTIME="$(ps -e -o "pid,etimes,command" |awk -v processid=$PID '{if($1==processid) print $2}')"
echo "========================== LAG = $LAG , ELAPSEDTIME = $ELAPSEDTIME ======================================="
sleep 2
done

if [ -z "$ELAPSEDTIME" ]
then
    echo "================Connector failed======================"
    cleanup
    exit 1
fi
echo "========================== Connector Job done successfully Killing the process ==================="

cat /tmp/config/s3config.properties | grep "topics=" | awk -F "=" '{print $2}' | tr , "\n" > /tmp/config/topics.txt

while IFS= read -r TOPIC
do
    # getting topic name from configuration file
    echo "=====================getting number of message to $TOPIC =============================="
    # getting retention period as set for the topic
    if ! MESSAGECOUNT="$(/bin/kafka-get-offsets --command-config /tmp/config/kafkaConfig.properties --bootstrap-server "$BOOTSTRAPSERVER" --topic "$TOPIC" --time -1 | awk -F  ":" '{sum += $3} END {print sum}')"
    then
        echo "Cannot fetch message count for a topic ${TOPIC}"
        exit 1
    fi

    if [ -z "$TOPICDESC" ]
    then
        export TOPICDESC="$TOPIC:$MESSAGECOUNT"
    else
        export TOPICDESC="$TOPICDESC,$TOPIC:$MESSAGECOUNT"
    fi
    # print
    echo "=============== topic description $TOPICDESC ===================================="
done < "/tmp/config/topics.txt"
kando output backupDetail ${TOPICDESC}
kando output s3path ${S3_PATH}
cleanup
exit 0
