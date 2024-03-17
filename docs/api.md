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

### Download a File
This section walks you through downloading a file to the Grapple S3 backend.

#### 1. Get Presigned Download URL
First, execute a `GET` request against the Grapple S3 API to retrieve the pre-signed upload url:

    GET /s3-presign-url?gym=<gym_pk>=&key=<file-name>&operation=download

Query parameters:
- gym: the primary key of the gym that the file is associated with.
- key: the name of the file in S3 to download.
- operation: `download`
- ttl: the amount of time for the pre-signed URL to be valid for. Defaults to 5 minutes. The value of this must be a [valid Go duration string](https://pkg.go.dev/maze.io/x/duration#:~:text=ParseDuration%20parses%20a%20duration%20string.%20A%20duration%20string%20is%20a%20possibly%20signed%20sequence%20of%20decimal%20numbers%2C%20each%20with%20optional%20fraction%20and%20a%20unit%20suffix%2C%20such%20as%20%22300ms%22%2C%20%22%2D1.5h%22%20or%20%222h45m%22.%20Valid%20time%20units%20are%20%22ns%22%2C%20%22us%22%20(or%20%22%C2%B5s%22)%2C%20%22ms%22%2C%20%22s%22%2C%20%22m%22%2C%20%22h%22%2C%20%22d%22%2C%20%22w%22%2C%20%22y%22.).

Example response:
```json
{
    "URL": "https://grapple-gym-videos.s3.us-west-1.amazonaws.com/Z3ltIzBhYzkxZTk2LTg5ZjUtNGU1Zi05ZGRlLTc5NDQxOGI4Yjg4OC9BbGVjJ3MgR3ltOQ%3D%3D/CODE_OF_CONDUCT.md?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAVRUVQ2TBCDCY7BLB%2F20240317%2Fus-west-1%2Fs3%2Faws4_request&X-Amz-Date=20240317T191509Z&X-Amz-Expires=300&X-Amz-SignedHeaders=host&x-id=GetObject&X-Amz-Signature=56a93e70c9329ec2fcda3f3412f0070248e36cb59b96f571d6b16d46541ae033",
    "Method": "GET",
    "SignedHeader": {
        "Host": [
            "grapple-gym-videos.s3.us-west-1.amazonaws.com"
        ]
    }
}
```

> `URL` is the pre-signed URL you can use to download the file. 

#### 2. Download the File
Simply execute a GET request with your web browser or `curl` against the pre-signed download URL to retrieve the raw contents of the file:

    curl -X GET <presigned-download-url>
