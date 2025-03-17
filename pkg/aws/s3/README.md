# How to do multi-part uploads


1. Start a multipart upload and returns you the Upload ID and Key. The frontend needs to save these two fields for future requests

    `GET s3/start-upload?file=my_filename.mp4&gym_id=<gym_id>&series_id=<series_id>`

2. Split input file into chunks based on its total size, get a presigned upload URL for each part of the file:
This will be a for-loop in the frontend code "For each part of the file i want to upload, send presigned upload request and upload that part"

    `PUT gym-series/67d76a58bca85cca6775b5ef/presign?upload_path=<upload_path_from_step1>&type=video&part_number=<part_number>&upload_id=<upload_id_from_step1>`

3. Once all parts are finished, frontend needs to send a final request:

    `POST /s3/complete-upload?upload_id=<upload_id_from_step1>&upload_path=<upload_path_from_step1>`