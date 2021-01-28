import os
import boto3
import datetime
from datetime import datetime, timedelta 
# Create an S3 client
s3 = boto3.client('s3', aws_access_key_id = os.environ.get('AWS_ACCESS_KEY'), aws_secret_access_key = os.environ.get('AWS_SECRET_KEY'), region_name = os.environ.get('region'))

years = 0
months = 0
days = 0
weeks = 0
hours = 0

yearly = os.environ.get('yearly')
if yearly != "null":
    years = int(yearly)
monthly = os.environ.get('monthly')
if monthly != "null":
    months = int(monthly)
daily = os.environ.get('daily')
if daily != "null":
    days = int(daily)
weekly = os.environ.get('weekly')
if weekly != "null":
    weeks = int(weekly)
hourly = os.environ.get('hourly')
if hourly != "null":
    hours = int(hourly)

frequency = os.environ.get('frequency')
if frequency == "null":
    print("frequency is null")
    exit()

print("frequency is",frequency)
now = datetime.now()
limit = now
print("Current time is ",limit.strftime("%Y-%m-%d, %H:%M:%S"))
if frequency == '@weekly':        
    earliestweek = max(weeks, months*4, years*12*4)
    print("weekly in weeks",weeks)
    print("monthly in weeks", months*4)
    print("yearly in weeks", years*12*4)
    print("max limit in weeks", earliestweek)
    limit = now - timedelta(weeks=earliestweek)
elif frequency == '@daily':
    earliestday = max(days, weeks*7, months*4*7, years*12*4*7)
    print("daily in days", days)
    print("weekly in days", weeks*7)
    print("monthly in days", months*4*7)
    print("yearly in days", years*12*4*7)
    print("max limit in days", earliestday)
    limit = now - timedelta(days=earliestday)
elif frequency == '@hourly':
    earliestHour = max( hours, days*24, weeks*7*24, months*4*7*24, years*12*4*7*24 )    
    print("hourly in hours", hours)    
    print("daily in hours", days*24)
    print("weekly in hours", weeks*7*24)
    print("monthly in hours", months*7*4*24)
    print("yearly in hours", years*12*4*7*24)
    print("max limit in hours", earliestHour)
    limit = now - timedelta(hours=earliestHour)
elif frequency == '@monthly':
    earliestMonth = max(months, years*12)
    print("monthly in month", months)
    print("yearly in month", years*12)
    print("max limit in month", earliestMonth)
    limit = now - timedelta(weeks=(earliestMonth*4))
elif frequency == '@yearly':
    earliestyear = years
    print("yearly in weeks", years*12*4)
    print("max limit in year", earliestyear)
    limit = now - timedelta(weeks=(earliestyear*12*4))


print("deleting every s3 object before ",limit.strftime("%Y-%m-%d, %H:%M:%S"))
bucket = os.environ.get('bucket')

res = s3.list_objects_v2(Bucket=bucket, Delimiter='/')
for o in res.get('CommonPrefixes'):
    topic = o.get('Prefix')
    if '_' in topic:
        timestamp = topic.split('_')[1].replace("/", "")
    else:
        continue
    creation_time = datetime.fromisoformat(timestamp+".000000")
    if creation_time <= limit :
        print("Deleting", topic)
        print(f"Getting S3 Key Name from the Bucket: {bucket} with Prefix: {topic}")
        key_names = []
        kwargs = {"Bucket": bucket, "Prefix": topic}
        while True:
            response = s3.list_objects_v2(**kwargs)
            for obj in response["Contents"]:
                key_names.append(obj["Key"])
            try:
                kwargs["ContinuationToken"] = response["NextContinuationToken"]
            except KeyError:
                break

        print(f'All Keys in {bucket} with {topic} Prefix found!')
        for i, val in enumerate(key_names):
            s3.delete_object(Bucket=bucket, Key=val)
        print(f"all objects deleted with prefix {topic}")
