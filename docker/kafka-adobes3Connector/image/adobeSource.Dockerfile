FROM confluentinc/cp-kafka-connect:latest

USER root
# copy the jar files
COPY docker/kafka-adobes3Connector/image/0.0.4-2a8a4aa-all.jar /opt/
# adding script to monitor source connector
COPY docker/kafka-adobes3Connector/image/adobe-monitorsource.sh monitorconnect.sh
