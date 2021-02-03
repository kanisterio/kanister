#!/bin/sh
echo "===========================Monitoring started==================================="
sleep 60
export maxlag=0
export group="connect-${connectorName}"
#export pid="$(lsof -t -i:8083)"
export elapsedtime="$(ps -e -o "pid,etimes,command" |awk -v processid=$pid '{if($1==processid) print $2}')"
export lag="$(/bin/kafka-consumer-groups --bootstrap-server "$bootstrapServer" --describe --group "$group"| awk 'BEGIN{maxLag=   0}{if ($6>0+maxLag) maxLag=$6} END{print maxLag}')"
echo "==========================Group=$group, MaxTime=$timeinSeconds, maxlag=$maxlag, pid=$pid, elapsedtime=$elapsedtime, lag=$lag, ==============================="
if [ $lag = "LAG" ]
then
    export lag=0
fi
while [[ "$lag" -gt "$maxlag"  && "$elapsedtime" -lt "$timeinSeconds" ]]
do
lag="$(/bin/kafka-consumer-groups --bootstrap-server "$bootstrapServer" --describe --group "$group"| awk 'BEGIN{maxLag=   0}{if ($6>0+maxLag) maxLag=$6} END{print maxLag}')"
elapsedtime="$(ps -e -o "pid,etimes,command" |awk -v processid=$pid '{if($1==processid) print $2}')"
echo "========================== lag = $lag , elapsedtime = $elapsedtime ======================================="
sleep 2
done
if [ -z "$elapsedtime" ]
then
    echo "================Connector failed======================"
    curl -X DELETE http://localhost:8083/connectors/$connectorName
    echo "================Connector Deleted======================"
    exit 1
fi
echo "========================== Connector Job done successfully Killing the process ==================="

cat /tmp/config/s3config.properties | grep "topics=" | awk -F "=" '{print $2}' | tr , "\n" > /tmp/config/topics.txt

while IFS= read -r topic
do
    # getting topic name from configuration file
    echo "=====================getting number of message to $topic =============================="
    # getting retention period as set for the topic
    export messageCount="$(/bin/kafka-run-class kafka.tools.GetOffsetShell --broker-list "$bootstrapServer" --topic "$topic" --time -1 --offsets 1 | awk -F  ":" '{sum += $3} END {print sum}')"

    if [ -z "$topicDesc" ]
    then
        export topicDesc="$topic:$messageCount"
    else
        export topicDesc="$topicDesc,$topic:$messageCount"
    fi
    # print 
    echo "=============== topic description $topicDesc ===================================="
done < "/tmp/config/topics.txt"
kando output backupDetail ${topicDesc}
kando output s3path ${s3_path}
curl -X DELETE http://localhost:8083/connectors/$connectorName
echo "================Connector Deleted======================"
kill -INT $pid
exit 0
