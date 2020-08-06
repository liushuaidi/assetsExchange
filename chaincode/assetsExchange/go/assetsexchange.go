package main

// shim 包提供API访问链码chaincode的状态变量，事务下上文和其它链码。
// https://godoc.org/github.com/hyperledger/fabric/core/chaincode/shim

// 导入的两个 github 开头的包是从 $GOPATH/src 目录开始的路径，也即这两个包在fabric的仓库中

import (
	"fmt"
	"encoding/json"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// AssertsManageCC 实现一个智能合约来管理不良资产
type AssertsManageCC struct{}

const (
	originOwner = "originOwnerPlaceholder"
)

// User 用户
type User struct {
	Name string `json:"name"` // messagepack || protobuf 格式也可
	Id   string `json:"id"`
	//Assets map[string]string `json:"assets"` // key:资产id, value:资产Name,但是map是无序的，换用切片
	Assets []string `json:"assets"` // 存储资产 id
}

// Asset 资产
type Asset struct {
	Name string `json:"name"`
	Id   string `json:"id"`
	//Metadata map[string]string `json:"metadata"` // 特殊属性，map无序，数据结构不合适，换为切片
	Metadata string `json:"metadata"` // 特殊属性
}

// AssetHistory 资产变更历史
type AssetHistory struct {
	AssetId        string `json:"asset_id"`
	OriginOwnerId  string `json:"origin_owner_id"`  // 资产的原始拥有者
	CurrentOwnerId string `json:"current_owner_id"` // 变更后当前的拥有者
}

// 以 user_ 开头的，认为是用户
func constructUserKey(userId string) string {
	return fmt.Sprintf("user_%s", userId)
}

// 以 asset_ 开头的，认为是资产
func constructAssetKey(assetId string) string {
	return fmt.Sprintf("asset_%s", assetId)
}

// 换用 shim 自带的 创造组合键方法 
// func constructAssetHistoryKey(OriginOwnerId, AssetId, CurrentOwnerId string) string {
// 	return fmt.Sprintf("history_%s_%s_%s",OriginOwnerId, AssetId, CurrentOwnerId)
// }

// 用户开户
func userRegister(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 1：检查参数的个数
	if len(args) != 2 {
		return shim.Error("not enough args")
	}

	// 2：验证参数的正确性
	name := args[0]
	id := args[1]
	if name == "" || id == "" {
		return shim.Error("invalid args")
	}

	// 3：验证数据是否存在 
	// 验证需要读取 stateDB，需要 shim 包中的 GetState 方法
	// stateDB（KV类型）需要定义组合键的方法来区分用户和资产
	if userBytes, err := stub.GetState(constructUserKey(id)); err == nil && len(userBytes) != 0 {
		return shim.Error("user already exist")
	}

	// 4： 状态写入
	user := &User{
		Name:   name,
		Id:     id,
		Assets: make([]string, 0),
	}

	// 序列化对象
	userBytes, err := json.Marshal(user)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error %s", err))
	}

	if err := stub.PutState(constructUserKey(id), userBytes); err != nil {
		return shim.Error(fmt.Sprintf("put user error %s", err))
	}

	// 成功返回
	return shim.Success(nil)
}

// 用户销户
func userDestroy(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 1：检查参数的个数
	if len(args) != 1 {
		return shim.Error("not enough args")
	}

	// 2：验证参数的正确性
	id := args[0]
	if id == "" {
		return shim.Error("invalid args")
	}

	// 3：验证数据是否存在 
	userBytes, err := stub.GetState(constructUserKey(id))
	if err != nil || len(userBytes) == 0 {
		return shim.Error("user not found")
	}

	// 4： 状态写入
	if err := stub.DelState(constructUserKey(id)); err != nil {
		return shim.Error(fmt.Sprintf("delete user error: %s", err))
	}

	// 删除用户名下的资产
	user := new(User)
	if err := json.Unmarshal(userBytes, user); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	for _, assetid := range user.Assets {
		if err := stub.DelState(constructAssetKey(assetid)); err != nil {
			return shim.Error(fmt.Sprintf("delete asset error: %s", err))
		}
	}

	return shim.Success(nil)
}

// 资产登记
func assetEnroll(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 1：检查参数的个数
	if len(args) != 4 {
		return shim.Error("not enough args")
	}

	// 2：验证参数的正确性
	assetName := args[0]
	assetId := args[1]
	metadata := args[2]
	ownerId := args[3]
	if assetName == "" || assetId == "" || ownerId == "" {
		return shim.Error("invalid args")
	}

	// 3：验证数据是否存在 
	userBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(userBytes) == 0 {
		return shim.Error("user not found")
	}

	if assetBytes, err := stub.GetState(constructAssetKey(assetId)); err == nil && len(assetBytes) != 0 {
		return shim.Error("asset already exist")
	}

	// 4： 状态写入
	// 1. 写入资产对象 2. 更新用户对象 3. 写入资产变更记录
	asset := &Asset{
		Name:     assetName,
		Id:       assetId,
		Metadata: metadata,
	}
	assetBytes, err := json.Marshal(asset)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal asset error: %s", err))
	}
	if err := stub.PutState(constructAssetKey(assetId), assetBytes); err != nil {
		return shim.Error(fmt.Sprintf("save asset error: %s", err))
	}

	user := new(User)
	// 反序列化user
	if err := json.Unmarshal(userBytes, user); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	user.Assets = append(user.Assets, assetId)
	// 序列化user
	userBytes, err = json.Marshal(user)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(user.Id), userBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 资产变更历史
	history := &AssetHistory{
		AssetId:        assetId,
		OriginOwnerId:  originOwner, // 第一次登记的资产持有人标记为 originOwnerPlaceholder
		CurrentOwnerId: ownerId,
	}
	historyBytes, err := json.Marshal(history)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal assert history error: %s", err))
	}
	 
	// CreateCompositeKey 创建组合键，并验证
	historyKey, err := stub.CreateCompositeKey("history", []string{
		assetId,
		originOwner,
		ownerId,
	})
	if err != nil {
		return shim.Error(fmt.Sprintf("create key error: %s", err))
	}// 验证结束

	if err := stub.PutState(historyKey, historyBytes); err != nil {
		return shim.Error(fmt.Sprintf("save assert history error: %s", err))
	}

	return shim.Success(nil)
}

// 资产转让
func assetExchange(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 1：检查参数的个数
	if len(args) != 3 {
		return shim.Error("not enough args")
	}

	// 2：验证参数的正确性
	ownerId := args[0]
	assetId := args[1]
	currentOwnerId := args[2]
	if ownerId == "" || assetId == "" || currentOwnerId == "" {
		return shim.Error("invalid args")
	}

	// 3：验证数据是否存在 

	// 资产出让者
	originOwnerBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(originOwnerBytes) == 0 {
		return shim.Error("user not found")
	}
	// 资产接收者
	currentOwnerBytes, err := stub.GetState(constructUserKey(currentOwnerId))
	if err != nil || len(currentOwnerBytes) == 0 {
		return shim.Error("user not found")
	}
	// 被处置的资产
	assetBytes, err := stub.GetState(constructAssetKey(assetId))
	if err != nil || len(assetBytes) == 0 {
		return shim.Error("asset not found")
	}

	// 校验原始拥有者确实拥有当前所要变更的资产
	originOwner := new(User)
	// 反序列化user
	if err := json.Unmarshal(originOwnerBytes, originOwner); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	aidexist := false
	for _, aid := range originOwner.Assets {
		if aid == assetId {
			aidexist = true
			break
		}
	}
	if !aidexist {
		return shim.Error("asset owner not match")
	}

	// 4： 状态写入
	// 1. 原始拥有者删除资产id 2. 新拥有者加入资产id 3. 资产变更记录
	assetIds := make([]string, 0)
	for _, aid := range originOwner.Assets {
		// 1
		if aid == assetId {
			continue
		}
		assetIds = append(assetIds, aid)
	}
	originOwner.Assets = assetIds
	// 原始拥有者 进行更新
	originOwnerBytes, err = json.Marshal(originOwner)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(ownerId), originOwnerBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 当前拥有者插入资产id 并更新
	currentOwner := new(User)
	// 反序列化user
	if err := json.Unmarshal(currentOwnerBytes, currentOwner); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	currentOwner.Assets = append(currentOwner.Assets, assetId)

	currentOwnerBytes, err = json.Marshal(currentOwner)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(currentOwnerId), currentOwnerBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 插入资产变更记录
	history := &AssetHistory{
		AssetId:        assetId,
		OriginOwnerId:  ownerId,
		CurrentOwnerId: currentOwnerId,
	}
	historyBytes, err := json.Marshal(history)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal assert history error: %s", err))
	}

	historyKey, err := stub.CreateCompositeKey("history", []string{
		assetId,
		ownerId,
		currentOwnerId,
	})
	if err != nil {
		return shim.Error(fmt.Sprintf("create key error: %s", err))
	}

	if err := stub.PutState(historyKey, historyBytes); err != nil {
		return shim.Error(fmt.Sprintf("save assert history error: %s", err))
	}

	return shim.Success(nil)
}

// 用户查询
func queryUser(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 1：检查参数的个数
	if len(args) != 1 {
		return shim.Error("not enough args")
	}

	// 2：验证参数的正确性
	ownerId := args[0]
	if ownerId == "" {
		return shim.Error("invalid args")
	}

	// 3：验证数据是否存在 
	userBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(userBytes) == 0 {
		return shim.Error("user not found")
	}

	return shim.Success(userBytes)
}

// 资产查询
func queryAsset(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 1：检查参数的个数
	if len(args) != 1 {
		return shim.Error("not enough args")
	}

	// 2：验证参数的正确性
	assetId := args[0]
	if assetId == "" {
		return shim.Error("invalid args")
	}

	// 3：验证数据是否存在 
	assetBytes, err := stub.GetState(constructAssetKey(assetId))
	if err != nil || len(assetBytes) == 0 {
		return shim.Error("asset not found")
	}

	return shim.Success(assetBytes)
}

// 资产变更历史查询
func queryAssetHistory(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 1：检查参数的个数,可以有1个或2个
	if len(args) != 2 && len(args) != 1 {
		return shim.Error("not enough args")
	}

	// 2：验证参数的正确性
	assetId := args[0]
	if assetId == "" {
		return shim.Error("invalid args")
	}

	queryType := "all"
	if len(args) == 2 {
		queryType = args[1]
	}

	if queryType != "all" && queryType != "enroll" && queryType != "exchange" {
		return shim.Error(fmt.Sprintf("queryType unknown %s", queryType))
	}

	// 3：验证数据是否存在 
	assetBytes, err := stub.GetState(constructAssetKey(assetId))
	if err != nil || len(assetBytes) == 0 {
		return shim.Error("asset not found")
	}

	// 查询相关数据
	keys := make([]string, 0)
	keys = append(keys, assetId)
	switch queryType {
		case "enroll":
			keys = append(keys, originOwner)
		case "exchange", "all": // 不添加任何附件key
		default:
			return shim.Error(fmt.Sprintf("unsupport queryType: %s", queryType))
	}
	result, err := stub.GetStateByPartialCompositeKey("history", keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("query history error: %s", err))
	}
	defer result.Close()

	histories := make([]*AssetHistory, 0)
	for result.HasNext() {
		historyVal, err := result.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("query error: %s", err))
		}

		history := new(AssetHistory)
		if err := json.Unmarshal(historyVal.GetValue(), history); err != nil {
			return shim.Error(fmt.Sprintf("unmarshal error: %s", err))
		}

		// 过滤掉不是资产转让的记录
		//  ！= exchange ？
		if queryType == "exchange" && history.OriginOwnerId == originOwner {
			continue
		}

		histories = append(histories, history)
	}

	historiesBytes, err := json.Marshal(histories)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal error: %s", err))
	}

	return shim.Success(historiesBytes)
}

// Init is called during Instantiate transaction after the chaincode container
// has been established for the first time, allowing the chaincode to
// initialize its internal data
func (c *AssertsManageCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke is called to update or query the ledger in a proposal transaction.
// Updated state variables are not committed to the ledger until the
// transaction is committed.
func (c *AssertsManageCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	// 调用shim包的方法，获取函数/方法名和对应的参数
	funcName, args := stub.GetFunctionAndParameters()

	switch funcName {
	case "userRegister":
		return userRegister(stub, args)
	case "userDestroy":
		return userDestroy(stub, args)
	case "assetEnroll":
		return assetEnroll(stub, args)
	case "assetExchange":
		return assetExchange(stub, args)
	case "queryUser":
		return queryUser(stub, args)
	case "queryAsset":
		return queryAsset(stub, args)
	case "queryAssetHistory":
		return queryAssetHistory(stub, args)
	default:
		return shim.Error(fmt.Sprintf("unsupported function: %s", funcName))
	}

	// stub.SetEvent("name", []byte("data"))
}

func main() {
	err := shim.Start(new(AssertsManageCC))
	if err != nil {
		fmt.Printf("Error starting AssertsManage chaincode: %s", err)
	}
}
