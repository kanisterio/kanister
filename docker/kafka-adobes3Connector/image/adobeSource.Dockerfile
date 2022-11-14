FROM confluentinc/cp-kafka-connect:6.1.0

USER root
# copy the jar files

RUN microdnf install -y \
   java-1.8.0-openjdk \
   java-1.8.0-openjdk-devel

ENV JAVA_HOME /usr/lib/jvm/java-1.8.0-openjdk/
RUN microdnf install git -y
RUN java -version
RUN git clone https://github.com/adobe/kafka-connect-s3.git
RUN cd kafka-connect-s3 && ./gradlew shadowJar

RUN cp ./kafka-connect-s3/build/libs/kafka-connect-s3-chart/kafka-connect/0.0.4-2a8a4aa-all.jar /opt/

# adding script to monitor source connector
COPY docker/kafka-adobes3Connector/image/adobe-monitorsource.sh monitorconnect.sh

COPY docker/kafka-adobes3Connector/image/cleans3.py cleanup.py
