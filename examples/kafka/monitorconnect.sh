#!/bin/sh
echo "===========================Monitoring started==================================="
sleep 60
export flushsize=`cat /tmp/config/s3config.properties | grep "flush.size=" | sed 's/.* //g'`
#export pid="$(lsof -t -i:8083)"
export elapsedtime="$(ps -e -o "pid,etimes,command" |awk -v processid=$pid '{if($1==processid) print $2}')"
export lag="$(/bin/kafka-consumer-groups --bootstrap-server "$bootstrapServer" --describe --group connect-kanister-kafka-S3SinkConnector| awk 'BEGIN{maxLag=   0}{if ($6>0+maxLag) maxLag=$6} END{print maxLag}')"
echo "========================== MaxTime=$timeinSeconds, flushsize=$flushsize, pid=$pid, elapsedtime=$elapsedtime, lag=$lag, ==============================="
if [ $lag = "LAG" ]
then
    export lag=0
fi
while [[ "$lag" -gt "$flushsize"  && "$elapsedtime" -lt "$timeinSeconds" ]]
do
echo "========================== lag = $lag , elapsedtime = $elapsedtime ======================================="
lag="$(/bin/kafka-consumer-groups --bootstrap-server "$bootstrapServer" --describe --group connect-kanister-kafka-S3SinkConnector| awk 'BEGIN{maxLag=   0}{if ($6>0+maxLag) maxLag=$6} END{print maxLag}')"
elapsedtime="$(ps -e -o "pid,etimes,command" |awk -v processid=$pid '{if($1==processid) print $2}')"
sleep 2
done
if [ -z "$elapsedtime" ]
then
    echo "================Connector failed======================"
    exit 1
fi
echo "========================== Connector Job done successfully Killing the process ==================="
kill -INT $pid
exit 0