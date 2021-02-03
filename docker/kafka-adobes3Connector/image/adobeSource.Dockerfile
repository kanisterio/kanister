FROM confluentinc/cp-kafka-connect:latest

USER root
# Python script to get the list of topic in s3 bucket
COPY docker/kafka-adobes3Connector/image/0.0.4-2a8a4aa-all.jar /opt/

COPY docker/kafka-adobes3Connector/image/adobe-monitorsource.sh monitorconnect.sh
