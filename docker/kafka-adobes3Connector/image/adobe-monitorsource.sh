#!/bin/sh
echo "===========================Monitoring started==================================="
sleep 60
export ELAPSEDTIME="$(ps -e -o "pid,etimes,command" |awk -v processid=$PID '{if($1==processid) print $2}')"
if [ -z "$ELAPSEDTIME" ]
then
    echo "================Connector failed======================"
    curl -X DELETE http://localhost:8083/connectors/$CONNECTORNAME
    echo "================Connector Deleted======================"
    exit 1
fi
for i in $(echo $TOPIC_DETAIL | sed "s/,/ /g")
do

    export TOPIC="$(echo $i | awk -F ":" '{print $1}')"
    export FINAL_MESSAGE_COUNT="$(echo $i | awk -F ":" '{print $2}')"
    if echo ",$TOPIC_LIST," | grep -q ",$TOPIC,"
    then

        echo "======================Start Restore process for topic $TOPIC with messagecount = $FINAL_MESSAGE_COUNT ============================="
        export START_OFFSET="$(/bin/kafka-run-class kafka.tools.GetOffsetShell --broker-list "$BOOTSTRAPSERVER" --topic "$TOPIC" --time -1 | grep -e ':[[:digit:]]*:' | awk -F  ":" '{sum += $3} END {print sum}')"
        export END_OFFSET="$(/bin/kafka-run-class kafka.tools.GetOffsetShell --broker-list "$BOOTSTRAPSERVER" --topic "$TOPIC" --time -2 | grep -e ':[[:digit:]]*:' | awk -F  ":" '{sum += $3} END {print sum}')"
        export CURRENT_MESSAGE_COUNT=$((START_OFFSET - END_OFFSET))
        echo "======================Start offset = $START_OFFSET , endoffset = $END_OFFSET , message count = $CURRENT_MESSAGE_COUNT ============================="

        until [ "$CURRENT_MESSAGE_COUNT" = "$FINAL_MESSAGE_COUNT" ]
        do
        echo "======================Restore in process for $TOPIC ============================="
        START_OFFSET="$(/bin/kafka-run-class kafka.tools.GetOffsetShell --broker-list "$BOOTSTRAPSERVER" --topic "$TOPIC" --time -1 | grep -e ':[[:digit:]]*:' | awk -F  ":" '{sum += $3} END {print sum}')"
        END_OFFSET="$(/bin/kafka-run-class kafka.tools.GetOffsetShell --broker-list "$BOOTSTRAPSERVER" --topic "$TOPIC" --time -2 | grep -e ':[[:digit:]]*:' | awk -F  ":" '{sum += $3} END {print sum}')"
        CURRENT_MESSAGE_COUNT=$((START_OFFSET - END_OFFSET))
        echo "======================Start offset = $START_OFFSET , endoffset = $END_OFFSET , message count = $CURRENT_MESSAGE_COUNT ============================="
        sleep 3
        done

        echo "=======================restore complete for $TOPIC =================================="
    else
        echo "=================$TOPIC not listed in the $TOPIC_LIST, skipping restore====================="
    fi
done

echo "=========================== All topic restored as per backup details ==================================="
curl -X DELETE http://localhost:8083/connectors/$CONNECTORNAME
echo "================Connector Deleted======================"
kill -INT $PID
exit 0
