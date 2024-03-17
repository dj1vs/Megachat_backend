#!/bin/bash

cd /usr/share/kafka

bin/kafka-topics.sh --create --topic megachat --bootstrap-server localhost:9092