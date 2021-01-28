FROM python:3.8

RUN apt-get update -y
RUN apt-get install jq -y
RUN pip install boto3

COPY cleanS3Object.py cleanup.py