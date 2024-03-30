# Grapple Gym Videos API Interface

This guide will walk you through the Grapple API for Gym Videos and S3 operations.

## Create a Gym Video

### Step 1:
First, upload the video to s3 by following the [docs](./api.md#upload-a-file).


### Step 2: Create the Gym  Video in Grapple Database
The next step is to store the Gym Video record in the Grapple DynamoDB database.

> **NOTE:** Make sure to store the name of the file you uploaded in the `s3_object` field of the request body.

Example request:
```
data='{
    "gym_id": "Z3ltIzBhYzkxZTk2LTg5ZjUtNGU1Zi05ZGRlLTc5NDQxOGI4Yjg4OC9BbGVjJ3MgR3ltOQ==",
    "title": "test video 2 - muy thai",
    "content": "my video description",
    "difficulty": "Advanced",
    "disciplines": ["muy thai"],
    "s3_object": "test_video_2_muy_thai.mp4"
}'

curl -X POST \
     -H "Authorization: Bearer <token>" \
     https://q6q57z2ve5.execute-api.us-west-1.amazonaws.com/Prod/gym-videos \
    -d $data
```

## Downloading Gym Video(s)

### Step 1:
First, get the Gym Video(s) you want to display from the Grapple Database:
```shell
curl -X GET -H "Authorization: Bearer <token>" https://q6q57z2ve5.execute-api.us-west-1.amazonaws.com/Prod/gym-videos?gym=Z3ltIzBhYzkxZTk2LTg5ZjUtNGU1Zi05ZGRlLTc5NDQxOGI4Yjg4OC9BbGVjJ3MgR3ltOQ==&limit=10
```

### Step 2:
Next, generate a presigned download URL for each video you want via the [S3 API](./api.md#download-files).

### Step 3:
For each video you intend on downloading, execute a GET request against the presigned URL.

