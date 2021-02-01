FROM python:3.8

RUN apt-get update -y && apt-get install jq -y && pip install boto3

COPY cleanS3Object.py cleanup.py