FROM confluentinc/cp-kafka-connect:latest

USER root

RUN yum install -y lsof

RUN wget https://raw.githubusercontent.com/canha/golang-tools-install-script/master/goinstall.sh && chmod +x goinstall.sh && source /root/.bashrc
RUN yum install perl-Digest-SHA -y
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash

COPY ./0.0.4-2a8a4aa-all.jar /opt/

COPY adobe-monitorsink.sh monitorconnect.sh
