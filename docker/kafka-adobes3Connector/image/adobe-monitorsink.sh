#!/bin/sh
echo "===========================Monitoring started==================================="
sleep 60
export MAXLAG=0
export GROUP="connect-${CONNECTORNAME}"
#export pid="$(lsof -t -i:8083)"
export ELAPSEDTIME="$(ps -e -o "pid,etimes,command" |awk -v processid=$PID '{if($1==processid) print $2}')"
export LAG="$(/bin/kafka-consumer-groups --bootstrap-server "$BOOTSTRAPSERVER" --describe --group "$GROUP"| awk 'BEGIN{maxLag=   0}{if ($6>0+maxLag) maxLag=$6} END{print maxLag}')"
echo "==========================GROUP=$GROUP, MaxTime=$TIMEINSECONDS, MAXLAG=$MAXLAG, pid=$PID, ELAPSEDTIME=$ELAPSEDTIME, LAG=$LAG, ==============================="
if [ $LAG = "LAG" ]
then
    export LAG=0
fi
while [[ "$LAG" -gt "$MAXLAG"  && "$ELAPSEDTIME" -lt "$TIMEINSECONDS" ]]
do
LAG="$(/bin/kafka-consumer-groups --bootstrap-server "$BOOTSTRAPSERVER" --describe --group "$GROUP"| awk 'BEGIN{maxLag=   0}{if ($6>0+maxLag) maxLag=$6} END{print maxLag}')"
ELAPSEDTIME="$(ps -e -o "pid,etimes,command" |awk -v processid=$PID '{if($1==processid) print $2}')"
echo "========================== LAG = $LAG , ELAPSEDTIME = $ELAPSEDTIME ======================================="
sleep 2
done
if [ -z "$ELAPSEDTIME" ]
then
    echo "================Connector failed======================"
    curl -X DELETE http://localhost:8083/connectors/$CONNECTORNAME
    echo "================Connector Deleted======================"
    exit 1
fi
echo "========================== Connector Job done successfully Killing the process ==================="

cat /tmp/config/s3config.properties | grep "topics=" | awk -F "=" '{print $2}' | tr , "\n" > /tmp/config/topics.txt

while IFS= read -r TOPIC
do
    # getting topic name from configuration file
    echo "=====================getting number of message to $TOPIC =============================="
    # getting retention period as set for the topic
    export MESSAGECOUNT="$(/bin/kafka-run-class kafka.tools.GetOffsetShell --broker-list "$BOOTSTRAPSERVER" --topic "$TOPIC" --time -1 --offsets 1 | awk -F  ":" '{sum += $3} END {print sum}')"

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
curl -X DELETE http://localhost:8083/connectors/$CONNECTORNAME
echo "================Connector Deleted======================"
kill -INT $PID
exit 0
