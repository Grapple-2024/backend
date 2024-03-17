for i in $(seq 11 100);
do
    curl --location --request POST 'http://localhost:3000/gyms' \
    --header 'Authorization: Bearer eyJraWQiOiIyYUxLcWhnZk1sMWFLK1RZaGNDS3ZYNmFFYlNkaU5aM1hNZzVPemQxQ1hNPSIsImFsZyI6IlJTMjU2In0.eyJzdWIiOiIwYWM5MWU5Ni04OWY1LTRlNWYtOWRkZS03OTQ0MThiOGI4ODgiLCJjb2duaXRvOmdyb3VwcyI6WyJjb2FjaCJdLCJlbWFpbF92ZXJpZmllZCI6ZmFsc2UsImJpcnRoZGF0ZSI6IjEwXC8yOFwvMTk5NyIsImNvZ25pdG86cHJlZmVycmVkX3JvbGUiOiJhcm46YXdzOmlhbTo6MzgxNDkxOTI2MjEwOnJvbGVcL3VzLXdlc3QtMV9IVDVvUjZBd08tY29hY2hHcm91cFJvbGUiLCJpc3MiOiJodHRwczpcL1wvY29nbml0by1pZHAudXMtd2VzdC0xLmFtYXpvbmF3cy5jb21cL3VzLXdlc3QtMV9IVDVvUjZBd08iLCJwaG9uZV9udW1iZXJfdmVyaWZpZWQiOmZhbHNlLCJjb2duaXRvOnVzZXJuYW1lIjoiam9yZGFuIiwiZ2l2ZW5fbmFtZSI6IkpvcmRhbiIsInBpY3R1cmUiOiIxMjMxMjMiLCJvcmlnaW5fanRpIjoiNWE4NTM4NzctMmJkOC00MGEyLTk2N2YtOGYwYTIwZTAyNTM1IiwiY29nbml0bzpyb2xlcyI6WyJhcm46YXdzOmlhbTo6MzgxNDkxOTI2MjEwOnJvbGVcL3VzLXdlc3QtMV9IVDVvUjZBd08tY29hY2hHcm91cFJvbGUiXSwiYXVkIjoiNDBzOW9vcDVlOXNyYWlyOG1sanVwbjAwMGoiLCJldmVudF9pZCI6ImQ0MWUxMGM5LTk4M2ItNDY5OS04ZWFkLWYxNjE4MTFmNWNhZiIsInRva2VuX3VzZSI6ImlkIiwiYXV0aF90aW1lIjoxNzA5NDA4NTY1LCJwaG9uZV9udW1iZXIiOiIrMTk0OTg3MDU1ODgiLCJleHAiOjE3MDk0MTIxNjUsImlhdCI6MTcwOTQwODU2NSwiZmFtaWx5X25hbWUiOiJMZXZpbiIsImp0aSI6ImEzYmY2OTJlLWI4ZmUtNDc0NS05NjkwLWNhMGNjYmVjNDE0ZCIsImVtYWlsIjoiam9yZGFuQGRpb255c3VzdGVjaG5vbG9neWdyb3VwLmNvbSJ9.Y9eD38lM7sQN9zi4ZKtfDHJfCRBaQ4hNSoliKCQhyh2ALVwFN_22Rod6_GVguIsH-K8E8TKgVsuOUU0x1KKHccO49FKncrFEBYe98FrBMc7zbw1BKwyGXcSm9W-uPBEF6ZL1JymvQGG4yIWTR6b7gtztylEp1Kx8xXp9vdFFu3doAgZfmqnlhHqcnWbzN5vdO3AyUh6SP7XpsIAha2aPjlAPY-IDq4a2UQMbaSWXN0WFPjGizgYyG2zxchR1sjX78SOIBGcHhPWLMmQw9ylhVPBwgKr6kiC-rKIkleTV2DcdLm-tF3lcSc9JHTTmT9SBSm3ycOHsFTGqfocYernP1w' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "Alec Gym'$i'",
        "address_line_1": "31881 Via Puntero 123",
        "city": "San Juan Capistrano",
        "state": "CA",
        "country": "USA",
        "disciplines": ["MMA", "kick boxing"],
        "creator": "jordan",
        "schedule": {
            "sun": [
                {
                    "title": "Session 1",
                    "start": "2024-02-18T15:04:05Z",
                    "end": "2024-02-19T15:04:05Z"
                },
                {
                    "title": "Session 2",
                    "start": "2024-02-19T15:04:05Z",
                    "end": "2024-02-19T15:04:05Z"
                }
            ]
        }
    }'
done


