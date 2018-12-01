# strike

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



