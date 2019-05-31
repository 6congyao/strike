# Changelog

## 0.1.0
+ Network
+ Network filter (Proxy)
+ Protocol
+ Stream

## 0.2.0
+ HTTP v1.1 protocol supported
+ MQTT v3.1.1 protocol supported

## 0.2.1
+ Stream filter (QPS/Rate)

## 0.3.0
+ TLS 1.2 supported

## 0.3.1
+ Stream filter (JWT Auth)
+ Dockerfile & travis

## 0.4.0
+ Refactoring
+ Session read timeout

## 0.4.1
+ Controller network filter to control other listeners via APIs(Http restful)

## 0.4.2
+ Controller stream filter to avoid unsupported method

## 0.4.3
+ Shard worker pool supports for ratio sharding

## 0.5.0
+ New worker pool for handling connections

## 0.5.1
+ Move to go module

## 0.5.3
+ ShardWorkerPool: Use connection id instead of proxy id for sharding in order to keep the sequence of session's messages