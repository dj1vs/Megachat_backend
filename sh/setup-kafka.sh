#!/bin/bash

cd /usr/share/kafka

sudo bin/zookeeper-server-start.sh config/zookeeper.properties &> /dev/null &
sudo bin/kafka-server-start.sh config/server.properties &> /dev/null &
bin/kafka-topics.sh --create --topic megachat --bootstrap-server localhost:9092

