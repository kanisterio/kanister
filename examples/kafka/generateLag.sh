# get the max task running value from the configuration
maxtask="$(kubectl describe cm -n kafka s3sinkconnector-config | grep ^tasks.max | sed 's/.* //g')"
# get the flushsize value from the configuration
flushsize="$(kubectl describe cm -n kafka s3sinkconnector-config | grep ^flush.size | sed 's/.* //g')"
# get the partsize value from the configuration
partsize="$(kubectl describe cm -n kafka s3sinkconnector-config | grep ^s3.part.size | sed 's/.* //g')"
# create a file on output.csv, If file doesnot exist, Create a new file and add header
if [ -f output.csv ]; then
grep -q '^TIMESTAMP' output.csv || sed -i '1s/^/TIMESTAMP,DESCRIPTION,GROUP,TOPIC,PARTITION,CURRENT-OFFSET,LOG-END-OFFSET,LAG,CONSUMER-ID,HOST,CLIENT-ID\n/' output.csv
else
echo TIMESTAMP,DESCRIPTION,GROUP,TOPIC,PARTITION,CURRENT-OFFSET,LOG-END-OFFSET,LAG,CONSUMER-ID,HOST,CLIENT-ID > output.csv
fi
# get the description and append timestamp, description
for i in `seq 10`; do kubectl -n kafka run kafka-monitor -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-consumer-groups.sh --bootstrap-server my-cluster-kafka-bootstrap:9092 --describe --all-groups | grep '^connect-kafka-s3.*' | sed "s/^/maxtask=$maxtask|flushSize=$flushsize|partsize=$partsize /" | sed "s/^/[$(date "+%Y-%m-%d-%H:%M:%S")] /" | sed 's/ \+/,/g' >> output.csv; sleep 5; done



