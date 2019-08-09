#!/usr/bin/env bash

protoc -I server --go_out=plugins=grpc:server server/*.proto