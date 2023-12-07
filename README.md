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

This project is written in Go to demonstrate the alternative languages available to Imagination within web based services.

# Usage
RQ is usually installed locally on a networked device or server. RQ works best when installed centrally, but can be installed on more than one device. Use cases for this might be where devices are connected to mobile networks directly and cannot communicate between themselves.


## Prerequisites

* [Go 1.21](https://go.dev/doc/install)
* [Docker Engine](https://docs.docker.com/engine/install/)

### Build and Run

1. Build the Docker image
   ```sh
   docker build -t rq:latest .
   ```
2. Run the container
   ```sh
   docker run -p 8080:8080 rq:latest 
   ```

## API Reference

### HTTP Requests
The following fields can be processed by RQ, depending on the type of request being made.

| Field       | Description                                                  | Required                     |
|-------------|--------------------------------------------------------------|------------------------------|
| url         | The URL to make the request to                               | Yes                          |
| file        | The file to be sent to `url`                                 | Optional                     |
| destFileKey | The value to be used when uploading a file to the onward API | Optional (if no file upload) |


All fields are composed into an object referred to as the `payload`. Where requests do not use the `application/json` 
Content-Type, this field will be unmarshalled when sent to the onward API as a form string.

Where `application/json` is the Content-Type, the payload will be sent as a data field, without encoding or alteration
to the data.


The HTTP Method used in the request to RQ will in turn be the method used in the future request to `url`.

The following HTTP Methods are supported:
* GET
* POST
* PATCH

# Examples


### Enqueue a Request with Media
Enqueue an HTTP POST request, where a binary file is present.

```sh
curl -F "url=https://imaginattion.com" -F "dstFileKey=data" -F "file=@media.mp4" -H "Content-Type: x-www-form-urlencoded" -X POST http://localhost:8080/api/rq/http
```