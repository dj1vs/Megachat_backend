#!/bin/bash

cd /usr/share/kafka

bin/kafka-topics.sh --create --topic megachat --bootstrap-server localhost:9092
bin/kafka-configs.sh --bootstrap-server localhost:9092 -entity-type topics --entity-name megachat --alter --add-config retention.ms=10000