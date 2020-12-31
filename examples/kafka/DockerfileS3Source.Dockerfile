FROM confluentinc/cp-kafka-connect:latest

# Using Confluent Hub client to install Kafka Connect S3 connector  
RUN confluent-hub install --no-prompt confluentinc/kafka-connect-s3-source:1.3.2