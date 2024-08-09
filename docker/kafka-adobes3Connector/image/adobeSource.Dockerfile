FROM confluentinc/cp-kafka-connect:7.7.0

USER root

RUN microdnf install -y \
   platform-python python3-libs

# TODO: maybe use builder image for that
RUN microdnf install -y \
   java-1.8.0-openjdk \
   java-1.8.0-openjdk-devel

ENV JAVA_HOME /usr/lib/jvm/java-1.8.0-openjdk/
RUN microdnf install git -y
RUN java -version
RUN git clone https://github.com/adobe/kafka-connect-s3.git
# Temp patch until vulnerable deps are fixed
RUN sed -i "s/versions.awsSdkS3 = '1.11.803'/versions.awsSdkS3 = '1.12.261'/g" kafka-connect-s3/dependencies.gradle
RUN sed -i "s/versions.jackson = '2.10.4'/versions.jackson = '2.12.7.1'/g" kafka-connect-s3/dependencies.gradle
RUN cd kafka-connect-s3 && ./gradlew shadowJar
# copy the jar files
RUN cp ./kafka-connect-s3/build/libs/kafka-connect-s3-chart/kafka-connect/0.0.4-2a8a4aa-all.jar /opt/
# cleanup
RUN rm -rf ~/.gradle ./kafka-connect-s3

# adding script to monitor source connector
COPY docker/kafka-adobes3Connector/image/adobe-monitorsource.sh monitorconnect.sh

COPY docker/kafka-adobes3Connector/image/cleans3.py cleanup.py
