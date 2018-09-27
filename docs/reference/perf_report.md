## 性能测试

### Strike 运行机器规格
Strike 运行在instance中，具体规格如下：

| 类别 | 信息 | 
| -------- | -------- | 
| 主机类型  | 性能型 |
| OS  | Ubuntu Server 16.04.5 LTS 64bit|
| CPU数量  | 2 |
| 内存 | 4G |

### Upstream 运行机器规格
Upstream 运行在instance中，具体规格如下：

| 类别 | 信息 | 
| -------- | -------- | 
| 主机类型  | 性能型 |
| OS  | Ubuntu Server 16.04.5 LTS 64bit|
| CPU数量  | 2 |
| 内存 | 4G |

### HTTP/1.1 测试结果
工具为wrk，网络为QingCloud基础网络内网。
 
 #### 并发20
 ./wrk -t4 -c20 -d30s http://10.91.24.5:8080
 
Thread Stats   Avg      Stdev     Max   +/- Stdev

    Latency   412.54us  508.40us  18.27ms   98.66%
    Req/Sec    13.05k     1.42k   18.91k    74.35%
    
  1559539 requests in 30.10s, 197.81MB read
  
Requests/sec:  51811.80

Transfer/sec:      6.57MB
 
 #### 并发40
 ./wrk -t4 -c40 -d30s http://10.91.24.5:8080

Thread Stats   Avg      Stdev     Max   +/- Stdev

    Latency   766.58us  499.29us  14.55ms   97.71%
    Req/Sec    13.37k     1.83k   25.71k    73.71%
    
  1598865 requests in 30.10s, 202.80MB read
  
Requests/sec:  53118.82

Transfer/sec:      6.74MB

 #### 并发100
 ./wrk -t4 -c100 -d30s http://10.91.24.5:8080
 
Thread Stats   Avg      Stdev     Max   +/- Stdev
 
    Latency     1.83ms    0.98ms  64.18ms   97.05%
    Req/Sec    13.80k     2.14k   19.20k    60.83%
     
  1647742 requests in 30.00s, 209.00MB read
   
Requests/sec:  54916.27
 
Transfer/sec:      6.97MB
 
 #### 并发200
 ./wrk -t4 -c200 -d30s http://10.91.24.5:8080
  
Thread Stats   Avg      Stdev     Max   +/- Stdev
  
    Latency     3.84ms    4.97ms 216.47ms   99.16%
    Req/Sec    13.62k     1.85k   22.35k    67.25%
      
  1626919 requests in 30.02s, 206.36MB read
    
Requests/sec:  54201.76
  
Transfer/sec:      6.87MB