# 基于 fabric 的资产交易平台

## 主要功能
- 用户开户&销户
- 资产登记 资产上链 or 用户绑定资产
- 资产转让 资产所有权的变更
- 查询功能 用户查询、资产查询、资产变更历史查询

### 前提
- 1.`curl`
- 2.`docker` >=1.19.*
- 3.`docker-compose` >=1.24.*
- 4.`python` 2.7
- 5.`node` >=8.* (or node 10.*)
- 6.`npm` (`cnpm`)
- 7.`go` >= 1.13.*
- 8.`Gin` 框架
- 9.`fabric` [二进制文件1](https://github.com/hyperledger/fabric/releases/download/v1.4.6/hyperledger-fabric-linux-amd64-1.4.6.tar.gz) 、[二进制文件2](https://github.com/hyperledger/fabric-ca/releases/download/v1.4.6/hyperledger-fabric-ca-linux-amd64-1.4.6.tar.gz) 下载完成后，解压到当前项目的`bin`目录(和app目录同级)
- 10.`fabric` v1.4.6 docker images
- 11.`fabric-sdk-go`

### 运行

```bash
# 1 进入 network 目录，启动网络
./networkstart.sh up

# 2 进入 app 目录
# 运行 go build 进行编译，会生成和目录同名的可执行程序，这里是 app
# 启动后端服务
./app
```

### 停止
```bash
# 关闭后端 
Ctrl + C

# 关闭网络
./networkstart.sh down
```