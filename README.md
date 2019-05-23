# strike

[![Build Status](https://travis-ci.org/6congyao/strike.svg?branch=master)](https://travis-ci.org/6congyao/strike)

## Principle
* Stable first
* Flexibility prioritize
* Dependency-less
* Performance aware
* Easy to deploy

## Layers
```
    -----------------------
   |         PROXY         |
    -----------------------
   |       STREAMING       |
    -----------------------
   |        PROTOCOL       |
    -----------------------
   |         NET/IO        |
    -----------------------
```

## NET/IO
```

Listener:
    - Event listener
        - ListenerEventListener
    - Filter
        - ListenerFilter
 	    
Connection:
    - Event listener
        - ConnectionEventListener
    - Filter
        - ReadFilter
        - WriteFilter

    ---------------------------------------------
    
        EventListener       EventListener           
           *|                   |*          		  
            |                   |       			  
           1|     1      *      |1          		  
        Listener --------- Connection      		  
           1|      [accept]     |1          		  
            |                   |-----------         
           *|                   |*          |*       
        ListenerFilter       ReadFilter  WriteFilter 
                                                     

```

## Stream
```
Stream:
    - Event listener
        - StreamEventListener
    - Encoder
        - StreamSender
    - Decoder
        - StreamReceiver
```

## Network Filters
* Delegation (net.Conn)
* Proxy
* Controller

## Stream Filters
* QPS
* Rate
* Auth (jwt)
* User QPS (TokenBucket)

## Supported protocols
* HTTP v1.1
* MQTT v3.1.1
* TLS v1.2

## 示例请参考：
+ [iot-gateway](/docs/quickstart/gateway.md)


