import os
import boto3
# Create an S3 client
s3 = boto3.client('s3', aws_access_key_id = os.environ.get('AWS_ACCESS_KEY'), aws_secret_access_key = os.environ.get('AWS_SECRET_KEY'), region_name = os.environ.get('REGION'))
bucket = os.environ.get('BUCKET')
prefix = os.environ.get('topicsDir')+'/'

result = s3.list_objects(Bucket=bucket, Prefix=prefix, Delimiter='/')
for o in result.get('CommonPrefixes'):
    print(o.get('Prefix'))
