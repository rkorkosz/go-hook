[![Go](https://github.com/rkorkosz/go-hook/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/rkorkosz/go-hook/actions/workflows/go.yml)
# go-hook

## Overview

`go-hook` is a distributed HTTP pub/sub service. Service discovery is implemented via UDP broadcast,
so servers can see each other as long as they are in the same network.

## Deployment

    $ make docker

Alternatively you can build it and run it as any other executable:

    $ make build
    $ ./htm -bind :8000
    $ ./htm -bind :8001
    $ ./htm -bind :8002

## Usage

You can subscribe to a topic by opening your browser at `http://localhost/topic`.
Then if someone sends a post request to the sender address you will see it in your browser.
Example:

    $ curl -X POST -d '{"hello": "world"}' -H 'Content-Type: application/json' http://localhost/topic/sender

Initially the intent was to use this as a webhook server but it's essentially a HTTP pub/sub service.
