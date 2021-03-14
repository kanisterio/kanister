FROM confluentinc/cp-kafka-connect:6.1.0

# Using Confluent Hub client to install Kafka Connect S3 connector
RUN confluent-hub install --no-prompt confluentinc/kafka-connect-s3:5.5.2

USER root

RUN yum install -y lsof

RUN wget https://raw.githubusercontent.com/canha/golang-tools-install-script/master/goinstall.sh
RUN chmod +x goinstall.sh
RUN source /root/.bashrc
RUN yum install perl-Digest-SHA -y
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash

COPY monitorsink.sh .
