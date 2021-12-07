### Endpoints

`GET /user`

`200`

Response
```json
{
    "id": "de5817c0-57a6-11ec-bf63-0242ac130002",
    "email": "joe@dimo.zone",
    "email_verified": true
}
```

`PUT /user`

JSON body
```json
{
    "email": "eric@dimo.zone"
}
```

`200`

Response
Response
```json
{
    "id": "de5817c0-57a6-11ec-bf63-0242ac130002",
    "email": "eric@dimo.zone",
    "email_verified": false
}
```

`POST /send-confirmation-email`

`POST /confirm-email`

JSON body

```json
{
    "key": "60412984"
}
```

`200`

`400`

This can fail for a few reasons:

- We already confirmed the email
- We never sent a confirmation email for the current candidate email address
- The confirmation key expired; the default timeout for this is 15 minutes
