FROM confluentinc/cp-kafka-connect:latest

USER root

RUN yum install -y lsof

# Install kando
ADD kando /usr/local/bin/

COPY ./0.0.4-2a8a4aa-all.jar /opt/

COPY adobe-monitorsink.sh monitorconnect.sh
