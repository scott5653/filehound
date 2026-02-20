#!/usr/bin/env python3
import boto3
from botocore.config import Config
import time

time.sleep(2)

s3 = boto3.client(
    "s3",
    endpoint_url="http://minio:9000",
    aws_access_key_id="minioadmin",
    aws_secret_access_key="minioadmin",
    region_name="us-east-1",
    config=Config(signature_version="s3v4"),
)

for bucket in ["test-bucket", "logs", "images"]:
    try:
        s3.create_bucket(Bucket=bucket)
        print(f"Created bucket: {bucket}")
    except Exception as e:
        print(f"Bucket {bucket}: {e}")

s3.put_object(Bucket="test-bucket", Key="test.txt", Body=b"test content")
s3.put_object(Bucket="test-bucket", Key="config.json", Body=b'{"key": "value"}')
print("Uploaded test files")
