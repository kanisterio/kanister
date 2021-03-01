FROM confluentinc/cp-kafka-connect:latest

USER root

RUN yum install -y lsof

# copy the jar files

RUN yum install -y \
   java-1.8.0-openjdk \
   java-1.8.0-openjdk-devel

ENV JAVA_HOME /usr/lib/jvm/java-1.8.0-openjdk/
RUN yum install git -y
RUN java -version
RUN git clone https://github.com/adobe/kafka-connect-s3.git
RUN cd kafka-connect-s3 && ./gradlew shadowJar
# copy the jar files
RUN cp ./kafka-connect-s3/build/libs/kafka-connect-s3-chart/kafka-connect/0.0.4-2a8a4aa-all.jar /opt/

# Install kando
#ADD kando /usr/local/bin/
RUN wget https://raw.githubusercontent.com/canha/golang-tools-install-script/master/goinstall.sh
RUN chmod +x goinstall.sh
RUN source /root/.bashrc
RUN yum install perl-Digest-SHA -y
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
# adding script to monitor sink connector
COPY adobe-monitorsink.sh monitorconnect.sh
