FROM python:3.8

RUN apt-get update -y && apt-get install jq -y && pip install boto3
# python script to clean s3 objects as per retention policy
COPY docker/kafka-adobes3Connector/image/cleanS3Object.py cleanup.py
