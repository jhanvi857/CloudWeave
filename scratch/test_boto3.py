import boto3

# Initialize boto3 client with CloudWeave endpoint
s3 = boto3.client(
    's3',
    endpoint_url='http://localhost:9000',
    aws_access_key_id='default-admin-key',
    aws_secret_access_key='default-admin-key',
    region_name='us-east-1'
)

print("1. Creating bucket 'boto3-bucket'...")
s3.create_bucket(Bucket='boto3-bucket')
print("Bucket created successfully.")

print("2. Putting object 'boto3-test.txt'...")
s3.put_object(Bucket='boto3-bucket', Key='boto3-test.txt', Body=b'Hello from boto3 SDK!')
print("Object put successfully.")

print("3. Listing objects in 'boto3-bucket'...")
res = s3.list_objects_v2(Bucket='boto3-bucket')
for obj in res.get('Contents', []):
    print(f" - Key: {obj['Key']}, Size: {obj['Size']} bytes")

print("4. Getting object 'boto3-test.txt'...")
obj = s3.get_object(Bucket='boto3-bucket', Key='boto3-test.txt')
data = obj['Body'].read()
print(f"Downloaded content: {data.decode('utf-8')}")

print("5. Deleting object 'boto3-test.txt'...")
s3.delete_object(Bucket='boto3-bucket', Key='boto3-test.txt')
print("Object deleted successfully.")

print("6. Deleting bucket 'boto3-bucket'...")
s3.delete_bucket(Bucket='boto3-bucket')
print("Bucket deleted successfully.")

print("\nALL BOTO3 OPERATIONS SUCCESSFUL!")
