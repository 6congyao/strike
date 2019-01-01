## 配置为MQ的gateway

## 简介

+ 该示例描述了strike作为MDMP的边界代理
+ 对外适配http v1 restful协议，对内可选

## 准备

需要一个编译好的strike程序
```
cd ${projectpath}/cmd/strike/strike
go build
```

+ 将编译好的程序移动到当前目录

```
mv strike ${targetpath}/
```

## gateway配置文件：

```
strike/examples/configs/mqgateway.json
```

### 启动strike

+ 使用mqgateway.json 运行mq边界代理

```
./strike -c mqgateway.json
```

### 调试strike
+ 使用mqgateway.json 运行调试,编辑program arguments:

```
-c /<your_working_dir>/examples/configs/mqgateway.json
```