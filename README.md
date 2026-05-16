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

### systemd

A systemd template unit for the distributed HTTP service is available at
`deploy/systemd/go-hook-http-multi@.service`. Build the binary, install it, then enable one or more
instances by port:

    $ make http-multi
    $ sudo useradd --system --no-create-home go-hook
    $ sudo cp htm /usr/local/bin/htm
    $ sudo cp deploy/systemd/go-hook-http-multi@.service /etc/systemd/system/
    $ sudo systemctl daemon-reload
    $ sudo systemctl enable --now go-hook-http-multi@8000
    $ sudo systemctl enable --now go-hook-http-multi@8001
    $ sudo systemctl enable --now go-hook-http-multi@8002

Check service status and logs with:

    $ systemctl status go-hook-http-multi@8000
    $ journalctl -u go-hook-http-multi@8000 -f

Each instance runs `htm -bind :<port>`.

## Usage

You can subscribe to a topic by opening your browser at `http://localhost/topic`.
Then if someone sends a post request to the sender address you will see it in your browser.
Example:

    $ curl -X POST -d '{"hello": "world"}' -H 'Content-Type: application/json' http://localhost/topic/sender

Initially the intent was to use this as a webhook server but it's essentially a HTTP pub/sub service.
