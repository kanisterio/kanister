FROM eclipse-temurin:11-jdk AS builder

RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

RUN git clone https://github.com/adobe/kafka-connect-s3.git
# Temp patch until vulnerable deps are fixed
RUN sed -i "s/versions.awsSdkS3 = '1.11.803'/versions.awsSdkS3 = '1.12.261'/g" kafka-connect-s3/dependencies.gradle
RUN sed -i "s/versions.jackson = '2.10.4'/versions.jackson = '2.12.7.1'/g" kafka-connect-s3/dependencies.gradle
RUN cd kafka-connect-s3 && ./gradlew shadowJar

FROM confluentinc/cp-kafka-connect:8.3.0

USER root

COPY --from=builder /kafka-connect-s3/build/libs/kafka-connect-s3-chart/kafka-connect/0.0.4-2a8a4aa-all.jar /opt/

# adding script to monitor source connector
COPY docker/kafka-adobes3Connector/image/adobe-monitorsource.sh monitorconnect.sh

COPY docker/kafka-adobes3Connector/image/cleans3.py cleanup.py
