FROM confluentinc/cp-kafka-connect:latest

USER root
# Python script to get the list of topic in s3 bucket
COPY ./0.0.4-2a8a4aa-all.jar /opt/

COPY adobe-monitorsource.sh monitorconnect.sh