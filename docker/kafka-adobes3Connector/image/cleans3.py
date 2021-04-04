import os
import boto3
# Create an S3 client
s3 = boto3.client('s3', aws_access_key_id = os.environ.get('AWS_ACCESS_KEY'), aws_secret_access_key = os.environ.get('AWS_SECRET_KEY'), region_name = os.environ.get('REGION'))
bucket = os.environ.get('BUCKET')
s3path = os.environ.get('S3PATH')

prefix = s3path.split('/')[3] + '/'

response = s3.list_objects_v2(Bucket=bucket, Prefix=prefix)

for object in response['Contents']:
    print('Deleting', object['Key'])
    s3.delete_object(Bucket=bucket, Key=object['Key'])
