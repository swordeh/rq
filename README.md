<a name="readme-top"></a>
<div align="center">

  <a href="https://github.com/imagination-it/rq">
    <img src="https://tech-studio-assets.s3.eu-west-1.amazonaws.com/rq/icon.png" alt="Logo" width="80" height="80">
  </a>
<h3>RQ</h3>
A queuing service for HTTP requests
</div>



## About
This service exists to allow applications to "fire and forget" HTTP upload requests to external systems, typically in environments where connectivity is poor or unreliable, and abstract away the implementation of the end service providers.

This project is written in Go to demonstrate the alternative languages available to Imagination within web based services.

## Usage
RQ is usually installed locally on a networked device or server. RQ works best when installed centrally, but can be installed on more than one device. Use cases for this might be where devices are connected to mobile networks directly and cannot communicate between themselves.

## Getting Started


### Prerequisites

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

#### Enqueue a Media Request

```http
  POST /api/rq/request
```
```sh
curl -d "@media.mp4" -X POST http://localhost:8080/api/rq/request
```