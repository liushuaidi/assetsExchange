package main

import (
	"fmt"
	"context"
	"time"
	"net/http"
	"bytes"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/event"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

func main() {
	router := gin.Default()
	// 定义路由， RESTful 一套web服务标准 
	{
		router.POST("/users", userRegister)	//用户注册
		router.GET("/users/:id", queryUser) //查询用户信息
		router.DELETE("/users/:id", deleteUser) //删除用户
		router.GET("/asset/get/:id", queryAsset) //资产查询
		router.GET("/asset/exchange/history", assetsExchangeHistory) //资产变更历史查询
		router.POST("/asset/enroll", assetsEnroll) //资产登记
		router.POST("/asset/exchange", assetsExchange) //资产转让
	}
	router.Run()
}

type UserRegisterRequest struct {
	Id   string `form:"id" binding:"required"`
	Name string `form:"name" binding:"required"`
}

// 用户开户
func userRegister(ctx *gin.Context) {
	// 参数处理 
	// name := args[0]
	// id := args[1]
	
	req := new(UserRegisterRequest)
	if err := ctx.ShouldBind(req); err != nil {
		ctx.AbortWithError(400, err)
		return
	}

	// 区块链交互
	resp, err := channelExecute("userRegister", [][]byte{
		[]byte(req.Name),
		[]byte(req.Id),
	})
	
	// 因为 postman 对于 非200-300 直接的错误，会直接返回错误编号，而不显示错误内容
	// 所以此处通过 200 直接返回，并显示错误内容
	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// 查询用户
func queryUser(ctx *gin.Context) {
	// 参数在 path 中，用 Param() 方法来提取参数
	// ownerId := args[0]
	userId := ctx.Param("id")

	resp, err := channelQuery("queryUser", [][]byte{
		[]byte(userId),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	//ctx.JSON(http.StatusOK, resp)
	// 返回 Payload（对于接收者有用的数据） 方便查看
	ctx.String(http.StatusOK, bytes.NewBuffer(resp.Payload).String())
}

// 用户销户
func deleteUser(ctx *gin.Context) {
	// id := args[0]
	userId := ctx.Param("id")

	resp, err := channelExecute("userDestroy", [][]byte{
		[]byte(userId),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// 资产查询
func queryAsset(ctx *gin.Context) {
	// assetId := args[0]
	assetId := ctx.Param("id")

	resp, err := channelQuery("queryAsset", [][]byte{
		[]byte(assetId),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	//ctx.JSON(http.StatusOK, resp)
	ctx.String(http.StatusOK, bytes.NewBuffer(resp.Payload).String())
}

type AssetsEnrollRequest struct {
	AssetName string `form:"assetname" binding:"required"`
	AssetId   string `form:"assetsid" binding:"required"`
	Metadata  string `form:"metadata" binding:"required"`
	OwnerId   string `form:"ownerid" binding:"required"`
}

// 资产登记
func assetsEnroll(ctx *gin.Context) {
	req := new(AssetsEnrollRequest)
	// 参数在 form 表单中，用 ShouldBind() 方法来提取参数
	// assetName := args[0]
	// assetId := args[1]
	// metadata := args[2]
	// ownerId := args[3]
	if err := ctx.ShouldBind(req); err != nil {
		ctx.AbortWithError(400, err)
		return
	}

	resp, err := channelExecute("assetEnroll", [][]byte{
		[]byte(req.AssetName),
		[]byte(req.AssetId),
		[]byte(req.Metadata),
		[]byte(req.OwnerId),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

type AssetsExchangeRequest struct {
	OriginOwnerId  string `form:"originownerid" binding:"required"`
	AssetId        string `form:"assetsid" binding:"required"`
	CurrentOwnerId string `form:"currentownerid" binding:"required"`
}

// 资产转让/交易
func assetsExchange(ctx *gin.Context) {
	req := new(AssetsExchangeRequest)
	// ownerId := args[0]
	// assetId := args[1]
	// currentOwnerId := args[2]
	if err := ctx.ShouldBind(req); err != nil {
		ctx.AbortWithError(400, err)
		return
	}

	resp, err := channelExecute("assetExchange", [][]byte{
		[]byte(req.OriginOwnerId),
		[]byte(req.AssetId),
		[]byte(req.CurrentOwnerId),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// 资产历史变更记录
func assetsExchangeHistory(ctx *gin.Context) {
	// 参数的个数,可以有1个或2个
	// assetId := args[0]
	// queryType = args[1]  {"all" "enroll" "exchange"}
	assetId := ctx.Query("assetid")
	queryType := ctx.Query("querytype") // 可为空

	resp, err := channelQuery("queryAssetHistory", [][]byte{
		[]byte(assetId),
		[]byte(queryType),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	//ctx.JSON(http.StatusOK, resp)
	ctx.String(http.StatusOK, bytes.NewBuffer(resp.Payload).String())
}

// 代码中用到的名字都是 yaml文件中的 key 而不是 value
var (
	sdk           *fabsdk.FabricSDK
	channelName   = "mychannel"
	chaincodeName = "assetscc"
	org           = "org1"	// 对应了 configtx.yaml 文件的160行
	user          = "Admin"
	configPath = "./config.yaml"
)

// 初始化 SDK，需要用到 配置文件：config.yaml
func init() {
	var err error
	sdk, err = fabsdk.New(config.FromFile(configPath))
	if err != nil {
		panic(err)
	}
}

// 区块链管理
func manageBlockchain() {
	// 表明身份
	ctx := sdk.Context(fabsdk.WithOrg(org), fabsdk.WithUser(user))

	cli, err := resmgmt.New(ctx) // resource management 资源管理包 resmgmt
	if err != nil {
		panic(err)
	}

	// 具体操作
	// SaveChannel 创建or更新通道
	cli.SaveChannel(resmgmt.SaveChannelRequest{}, resmgmt.WithOrdererEndpoint("orderer.example.com"),
	resmgmt.WithTargetEndpoints())
}

// 区块链查询   账本查询
func queryBlockchain() {
	ctx := sdk.ChannelContext(channelName, fabsdk.WithOrg(org), fabsdk.WithUser(user))

	cli, err := ledger.New(ctx) // 实例化一个账本客户端
	if err != nil {
		panic(err)
	}

	resp, err := cli.QueryInfo(ledger.WithTargetEndpoints("peer0.org1.example.com"))
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)

	// 1 查询当前区块哈希
	cli.QueryBlockByHash(resp.BCI.CurrentBlockHash)

	// 2 从第0个块开始遍历
	for i := uint64(0); i <= resp.BCI.Height; i++ {
		cli.QueryBlock(i)
	}
	// 以上两种方式都可以实现区块链浏览器读取区块的功能
}

// 区块链交互
func channelExecute(fcn string, args [][]byte) (channel.Response, error) {
	ctx := sdk.ChannelContext(channelName, fabsdk.WithOrg(org), fabsdk.WithUser(user))

	cli, err := channel.New(ctx)
	if err != nil {
		return channel.Response{}, err
	}
	
	// 状态更新，insert/update/delete
	resp, err := cli.Execute(channel.Request{
		ChaincodeID: chaincodeName,
		Fcn:         fcn,
		Args:        args,
	}, channel.WithTargetEndpoints("peer0.org1.example.com"))
	if err != nil {
		return channel.Response{}, err
	}

	// 链码事件监听
	go func() {
		// channel 
		reg, ccevt, err := cli.RegisterChaincodeEvent(chaincodeName, "eventname")
		if err != nil {
			return
		}
		defer cli.UnregisterChaincodeEvent(reg)

		timeoutctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		for {
			select {
			case evt := <-ccevt:
				fmt.Printf("received event of tx %s: %+v", resp.TransactionID, evt)
			case <-timeoutctx.Done():
				fmt.Println("event timeout, exit!")
				return
			}
		}

		// event
		// eventcli, err := event.New(ctx)
		// if err != nil {
		// 	return
		// }
		// eventcli.RegisterChaincodeEvent(chaincodeName, "eventname")
		// ... same as channel moudle
		// 
	}()

	// 交易状态事件监听
	go func() {
		eventcli, err := event.New(ctx)
		if err != nil {
			return
		}

		reg, status, err := eventcli.RegisterTxStatusEvent(string(resp.TransactionID))
		defer eventcli.Unregister(reg) // 注册必有注销

		timeoutctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		for {
			select {
			case evt := <-status:
				fmt.Printf("received event of tx %s: %+v", resp.TransactionID, evt)
			case <-timeoutctx.Done():
				fmt.Println("event timeout, exit!")
				return
			}
		}
	}()

	return resp, nil
}

func channelQuery(fcn string, args [][]byte) (channel.Response, error) {
	ctx := sdk.ChannelContext(channelName, fabsdk.WithOrg(org), fabsdk.WithUser(user))

	cli, err := channel.New(ctx)
	if err != nil {
		return channel.Response{}, err
	}

	// 状态的查询，select
	return cli.Query(channel.Request{
		ChaincodeID: chaincodeName,
		Fcn:         fcn,
		Args:        args,
	}, channel.WithTargetEndpoints("peer0.org1.example.com"))
}

// 事件监听
func eventHandle() {
	ctx := sdk.ChannelContext(channelName, fabsdk.WithOrg(org), fabsdk.WithUser(user))

	cli, err := event.New(ctx)
	if err != nil {
		panic(err)
	}

	// 交易状态事件		--区块链交互模块
	// 链码事件 业务事件	--区块链交互模块
	
	// 区块事件
	reg, blkevent, err := cli.RegisterBlockEvent()
	if err != nil {
		panic(err)
	}
	defer cli.Unregister(reg)

	timeoutctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	for {
		select {
		case evt := <-blkevent:
			fmt.Printf("received a block, %+v", evt)
		case <-timeoutctx.Done():
			fmt.Println("event timeout, exit!")
			return
		}
	}
}