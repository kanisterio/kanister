FROM confluentinc/cp-kafka-connect:6.1.0

# Using Confluent Hub client to install Kafka Connect S3 connector
RUN confluent-hub install --no-prompt confluentinc/kafka-connect-s3-source:1.3.2
# Python script to get the list of topic in s3 bucket
COPY getTopicNames.py .
