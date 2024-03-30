# API Documentation

## Table of Contents
- [S3](#s3)
  - [Upload a File](#upload-a-file)
  - [Download a File](#download-a-file)


---

## S3
This section documents the Grapple S3 API.

### Upload a File
---

This section walks you through uploading a file to the Grapple S3 backend.

#### 1. Get the Pre-signed Upload URL
First, execute a `GET` request against the Grapple S3 API to retrieve the pre-signed upload url:

    curl -X GET /s3-presign-url?gym=<gym_pk>=&key=<file-name>&operation=upload

Query parameters:
- gym: the primary key of the gym that the file is to be associated with.
- key: the name of the file to upload.
- operation: `upload`
- ttl: the amount of time for the pre-signed URL to be valid for. Defaults to 5 minutes. The value of this must be a [valid Go duration string](https://pkg.go.dev/maze.io/x/duration#:~:text=ParseDuration%20parses%20a%20duration%20string.%20A%20duration%20string%20is%20a%20possibly%20signed%20sequence%20of%20decimal%20numbers%2C%20each%20with%20optional%20fraction%20and%20a%20unit%20suffix%2C%20such%20as%20%22300ms%22%2C%20%22%2D1.5h%22%20or%20%222h45m%22.%20Valid%20time%20units%20are%20%22ns%22%2C%20%22us%22%20(or%20%22%C2%B5s%22)%2C%20%22ms%22%2C%20%22s%22%2C%20%22m%22%2C%20%22h%22%2C%20%22d%22%2C%20%22w%22%2C%20%22y%22.).


The response will contain the pre-signed upload URL. Copy this URL and move onto the next step.

#### 2. Upload the file
Next, execute a `PUT` request against the pre-signed upload URL you received from the previous step:

    curl -X PUT <presigned-upload-url> --data-binary "@<file-name>"

The response body will be empty with a status code of `200` if the request was successful. You can verify the file was uploaded by following the instructions below to download the same file.

---

### Download File(s)
This section walks you through downloading file(s) from the Grapple S3 backend.

#### 1. Get Presigned Download URL(s)
First, execute a `GET` request against the Grapple S3 API to retrieve the pre-signed upload urls for each file you want to download

    GET /s3-presign-url?gym=<gym_pk>=&key=<file1>&key=<file2>&operation=download

Query parameters:
- gym: the primary key of the gym that the file is associated with.
- key: The name of the file in S3 to generate a pre-signed download url for. This parameter can be specified multiple times to retrieve download URLs for multiple files.
- operation: `download`
- ttl: the amount of time for the pre-signed URL to be valid for. Defaults to 5 minutes. The value of this must be a [valid Go duration string](https://pkg.go.dev/maze.io/x/duration#:~:text=ParseDuration%20parses%20a%20duration%20string.%20A%20duration%20string%20is%20a%20possibly%20signed%20sequence%20of%20decimal%20numbers%2C%20each%20with%20optional%20fraction%20and%20a%20unit%20suffix%2C%20such%20as%20%22300ms%22%2C%20%22%2D1.5h%22%20or%20%222h45m%22.%20Valid%20time%20units%20are%20%22ns%22%2C%20%22us%22%20(or%20%22%C2%B5s%22)%2C%20%22ms%22%2C%20%22s%22%2C%20%22m%22%2C%20%22h%22%2C%20%22d%22%2C%20%22w%22%2C%20%22y%22.).

Example request to generate presigned download URLs for two files: `<file1>` and `<file2>`:

`GET /s3-presign-url?gym=<gym_pk>=&key=<file1>&key=<file2>&operation=download`

```json
[
    {
        "s3_object": "<gym_pk>/<file1>",
        "url": "https://grapple-gym-videos.s3.us-west-1.amazonaws.com/<gym_pk>/<file1>?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAVRUVQ2TBCDCY7BLB%2F20240330%2Fus-west-1%2Fs3%2Faws4_request&X-Amz-Date=20240330T162003Z&X-Amz-Expires=300&X-Amz-SignedHeaders=host&x-id=GetObject&X-Amz-Signature=b454c9524ecd48d8c67aac15b984474ed9dfbb18cb56077404507f91119575a2"
    },
    {
        "s3_object": "<gym_pk>/<file2>",
        "url": "https://grapple-gym-videos.s3.us-west-1.amazonaws.com/<gym_pk>/<file2>?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAVRUVQ2TBCDCY7BLB%2F20240330%2Fus-west-1%2Fs3%2Faws4_request&X-Amz-Date=20240330T162003Z&X-Amz-Expires=300&X-Amz-SignedHeaders=host&x-id=GetObject&X-Amz-Signature=3f4b5f419ccf0b786c7162123b31a654490444baa6c19499a7ba3613daf16994"
    }
]
```

> `URL` is the pre-signed URL you can use to download the file. 

#### 2. Download the File
Simply execute a GET request with your web browser or `curl` against the pre-signed download URL to retrieve the raw contents of the file:

    curl -X GET <presigned-download-url>
