FROM confluentinc/cp-kafka-connect:latest

USER root

RUN yum install -y lsof

# Install kando
ADD kando /usr/local/bin/

COPY docker/kafka-adobes3Connector/image/0.0.4-2a8a4aa-all.jar /opt/

COPY docker/kafka-adobes3Connector/image/adobe-monitorsink.sh monitorconnect.sh
