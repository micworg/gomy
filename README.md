# gomy

A simpole mysql rest api

## Requirements:

- Linux or MacOS
- golang installed
- a running mysql/mariadb server

## Build:

`go build gomy`

## Setup:

`./gomy -s`

This will create a configuration file in your home and a database. The database contains access tokens and access rights.

## Run:

`./gomy -r`

Launches the gomy dervice

## Usage:

### Test if gomy service is running:

`curl -s https://your.domain/ping`

```
{
  "success": 1,
  "ping": "gomy service is running"
}
```

### Login:

Login into your database and generate an access token.

`curl -s https://your.domain/v1/login -d '{"user":"USER","pw":"PASSWORD","db":"DATABASE"}'`

```
{
  "success": 1,
  "token": "MYNEWSECRETTOKEN"
}
```

### SQL:

`curl -s https://your.domain/v1/sql -d '{"token":"MYNEWSECRETTOKEN","sql":"SELECT * FROM table"}' | jq`

```
{
  "success": 1,
  "data": [
    {
      "id": "1",
      "name": "a",
      "value": "1"
    },
    {
      "id": "2",
      "name": "b",
      "value": "2"
    }
  ]
}
```
