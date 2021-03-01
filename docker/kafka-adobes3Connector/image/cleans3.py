import os
import boto3
# Create an S3 client
s3 = boto3.client('s3', aws_access_key_id = os.environ.get('AWS_ACCESS_KEY'), aws_secret_access_key = os.environ.get('AWS_SECRET_KEY'), region_name = os.environ.get('region'))
# bucket="kafka-blueprint-bucket"
# s3path = "s3://kafka-blueprint-bucket/topics_2021-02-27T15:25:54"
bucket = os.environ.get('bucket')
s3path = os.environ.get('s3path')

prefix = s3path.split('/')[3] + '/'

response = s3.list_objects_v2(Bucket=bucket, Prefix=prefix)

for object in response['Contents']:
    print('Deleting', object['Key'])
    s3.delete_object(Bucket=bucket, Key=object['Key'])