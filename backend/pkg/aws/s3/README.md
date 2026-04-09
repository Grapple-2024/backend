# How to do multi-part uploads


1. Start a multipart upload and returns you the Upload ID and Key. The frontend needs to save these two fields for future requests

    `GET /s3/start-upload?file=my_filename.mp4&gym_id=<gym_id>&series_id=<series_id>`

2. Split input file into chunks based on its total size, get a presigned upload URL for each part of the file:
This will be a for-loop in the frontend code "For each part of the file i want to upload, send presigned upload request and upload that part"

untested frontend code example:
```typescript
for (let partNumber = 1; partNumber <= totalParts; partNumber++) {
    const chunk = file.slice((partNumber - 1) * PART_SIZE, partNumber * PART_SIZE);

    // Request a presigned URL from the backend
    const presignedRes = await fetch("/gym-series/{id}/presign?upload_id={uploadid_from_step1}&upload_path=<uploadpath_from_step1>&part_number={partNumber}&type=video", {
        method: "PUT",
        body: null,
        headers: { "Content-Type": "application/json" },
    });

    const { url } = await presignedRes.json();

    // Upload the chunk directly to S3
    await fetch(url, {
        method: "PUT",
        body: chunk,
        headers: { "Content-Type": "application/octet-stream" },
    });
}

// Notify backend to complete upload
await fetch("/s3/complete-upload", {
    method: "POST",
    body: JSON.stringify({ upload_id: uploadId, upload_path: "your/file/path.ext" }),
    headers: { "Content-Type": "application/json" },
});

```

3. Once all parts are finished, frontend needs to send a final request:

    `POST /s3/complete-upload?upload_id=<upload_id_from_step1>&upload_path=<upload_path_from_step1>`