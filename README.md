<a name="readme-top"></a>
<div align="center">

  <a href="https://github.com/imagination-it/rq">
    <img src="https://tech-studio-assets.s3.eu-west-1.amazonaws.com/rq/icon.png" alt="Logo" width="80" height="80">
  </a>
<h3>RQ</h3>
A queuing service for HTTP requests
</div>



# About
This service exists to allow applications to "fire and forget" HTTP requests to external systems, typically in environments where connectivity is poor or unreliable, and abstract away the implementation of the end service providers.

# Usage
RQ is usually installed locally on a networked device or server. RQ works best when installed centrally, but can be installed on more than one device. Use cases for this might be where devices are connected to mobile networks directly and cannot communicate between themselves.

### Quick Start
If you just want to get started and test, simply replace the URL you are currently calling directly with the URL for RQ,
and your target URL as a querystring parameter.

For example, instead of calling `https://example.com/api`, call `/api/rq/http?https://example.com/api`.

## Prerequisites

* [Go 1.21](https://go.dev/doc/install)


## API Reference

### HTTP Requests

```shell
METHOD /api/rq/http
```

RQ will queue requests that are of type:

* GET
* POST
* PATCH
* PUT

The HTTP Method used in the request to RQ will in turn be the method used in the future request to `url`.

All fields are composed into an object referred to as the `payload`. Where requests do not use the `application/json` 
Content-Type, this field will be unmarshalled when sent to the onward API as a form string.

Where `application/json` is the Content-Type, the payload will be sent as a data field, without encoding or alteration
to the data.

#### Example
Send a request to an API with two files and mixed data types in the form.
```shell
curl -v --location --request POST 'http://localhost:8080/api/rq/http?url=https://www.example.com' \
--header 'Authorization: Bearer mytokenishere' \
--form 'additional_data="[\"hello\", \"hola\", \"gday\"]"' \
--form 'file=@"file.mp4"' \
--form 'extra_data="{\"additional_context\":  {\"preferred_language\": \"go"}}"' \
--form 'another_file=@"file.mp4"' \
--form 'coffee="yes"' \
--form 'webhook_url="https://example.com/callback"'
```

```shell
{
  "id": "0da272fa-9c73-46b3-a9ca-d88f55cccd80",
  "record": {
    "id": "0da272fa-9c73-46b3-a9ca-d88f55cccd80",
    "method": "POST",
    "content_type": "multipart/form-data",
    "headers": {
      "Authorization": [
        "Bearer mytokenishere"
      ]
    },
    "url": "https://www.example.com",
    "file_keys": "[\"file\",\"another_file\"]",
    "payload": {
      "additional_data": [
        "[\"hello\", \"hola\", \"gday\"]"
      ],
      "coffee": [
        "yes"
      ],
      "extra_data": [
        "{\"additional_context\":  {\"preferred_language\": \"go"
      ],
      "webhook_url": [
        "https://example.com/callback"
      ]
    },
    "error": ""
  }
}
```

Send a request to an API using the `application/json` content type.

```shell
curl -X POST -H "Content-Type: application/json" -H "X-Special: Doit" --data '{"first_name": "foo", "last_name": "bar"}' http://localhost:8080/api/rq/http\?url\=https://example.com
```

```shell
{
  "id": "b836a981-b66e-4007-ae3d-1a2b1cc6a172",
  "record": {
    "id": "b836a981-b66e-4007-ae3d-1a2b1cc6a172",
    "method": "POST",
    "content_type": "application/json",
    "headers": {
      "X-Special": [
        "Doit"
      ]
    },
    "url": "https://example.com",
    "file_keys": "",
    "payload": {
      "first_name": "foo",
      "last_name": "bar"
    },
    "error": ""
  }
}
```

## Development
### Build and Run

1. Build the Docker image
   ```sh
   docker build -t rq:latest .
   ```
2. Run the container
   ```sh
   docker run -p 8080:8080 rq:latest 
   ```