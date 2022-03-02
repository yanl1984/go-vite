

# 简介 Introduction

先介绍每个目录的用途：Let's go through the purpose of each directory:

- .github 用来存放与github相关的配置，目前github action的配置存放在其中 | .github keeps github-related configurations. Configs for github action live there
- bin 一些在开发和部署阶段可能会用到的脚本 | bin keeps scripts used for development and deployment
- client 实现了远程调用gvite节点的go版本客户端 | client implements the go-version client for remote calls for gvite nodes
- cmd 命令行脚本，gvite的入口就是这里cmd/gvite/main.go | cmd is a command-line script. The entry to gvite is cmd/gvite/main.go
- common 一些公用的方法组建以及常量的定义 | Definition of some commonly used functions and constants
- conf 存放配置文件，conf/node_config.json是主网默认的 | conf keeps configuration files, conf/node_convig.json is the default for the Mainnet
- contracts-vite 存放vite相关的智能合约，contracts-vite/contracts/ViteToken.sol就是部署在eth网络上的VITE ERC20 Token的源码 | contracts-vite keeps vite-related smart contracts, contracts-vite/contracts/ViteToken.sol are VITE ERC20 token contracts deployed on the Ethereum network
- crypto 加密算法和hash算法的实现 ed25519和blake2b-512是vite链上的主要算法 | crypto - implementation of cryptographical algorithms and hash algorithms. ed25519 and blake2b-512 are the main algos used by the vite chain
- docker 存放dockerfile，配合docker编译时使用 | docker - keeps dockerfile, used for compiling docker
- docs go-vite相关的wiki存放，链接文档 https://docs.vite.org/go-vite/ | docs - wiki documents related to go-vite, the link is https://docs.vite.org/go-vite/
- interfaces 一些模块的接口定义放到这里，解决golang循环依赖问题 | definitions of some module interfaces live here. They solve the cyclical dependence problem with golang
- ledger 包含账本相关的实现 | ledger - ledger-related implementation
- log15 go-vite中采用的日志框架，由于当初改了不少东西，就copy过来了，原项目地址：https://github.com/inconshreveable/log15 | log15: logging framework for go-vite. Lots of things were previously changed, so got copied over from: https://github.com/inconshreveable/log15
- monitor 压力测试时，为了方便数据统计写的工具包 | monitor: toolkit for data aggregation in stress tests
- net 网络层实现，包含了节点发现，节点互通，账本广播和拉取 | net: implementation of the networking layer, including node discovery, inter-node communication, ledger propagation and retrieval
- node 一个ledger的上层包装，主要做一些组合的功能 | node: a top level package for ledger, mainly used for composability
- pow 本地实现的pow计算和远程调用pow-client进行计算pow | pow: locally implemented pow calculation and remotely called pow-client for pow calculation
- producer 触发snapshot block和contract block生产的入口 | producer: interface for activating snapshot block and production of contract block
- rpc 对于websocket/http/ipc三种远程调用方式的底层实现 | rpc: implementation of three remote invocation methods: websocket/http/ipc
- rpcapi 内部各个模块对外暴露的rpc接口 | rpcapi: externally exposing each module via rpc interface
- smart-contract 类似contracts-vite | smart-contract: like contracts-vite
- tools 类似collection，一些集合的实现 | tools: like collection, implementation of collections
- version 存放每次编译的版本号，make脚本会修改version/buildversion的内容 | version: saves version for each compilation, the make script will modify version/buildversion
- vm 虚拟机的实现和内置合约的实现，里面大量的虚拟机执行逻辑 | vm: implementation of the virtual machine and built-in smart contracts. Many VM-execution logic inside
- vm_db 虚拟机在执行的时候需要的存储接口，是虚拟机访问chain的封装 | vm_db: the interface for storage during VM execution. It's a wrap-around for when the VM visits chain
- wallet 钱包的实现，私钥助记词管理，以及签名和验证签名 | wallet: wallet implementation - private key/mnemonics/signing/signature verification


然后是gvite的实现整体框架：

主要分为以下几个大的模块：
1. net   网络互通
2. chain 底层存储
3. vm    虚拟机的执行
4. consensus  负责共识出块节点
5. pool  负责分叉选择
6. verifier   包装各种验证逻辑，是vite协议的包装
7. onroad 	  负责合约receive block的生产
8. generator  聚合create block的流程，会被onroad和api等模块调用

# 数据流

## 用户发送一笔转账交易

- rpcapi/api/ledger_v2.go#SendRawTransaction    	 		rpc是所有访问节点的入口
- ledger/pool/pool.go#AddDirectAccountBlock		 			通过pool，将block直接插入到账本中
- ledger/verifier/verifier.go#VerifyPoolAccountBlock  		调用verifier，验证block的有效性
- ledger/chain/insert.go#InsertAccountBlock		 			调用chain的接口插入到链中
- ledger/chain/index/insert.go#InsertAccountBlock           维护block的hash关系，例如高度与hash间关系，send-receive之间关系等等，涉及存储文件ledger/index/*和ledger/blocks
- ledger/chain/state/write.go#Write							将block改变的state进行维护，例如余额，storage等，涉及存储文件ledger/state


## 生成一个合约的receive block

- ledger/consensus/trigger.go#update 										产生一个合约开始出块的事件，传递到下游
- producer/producer.go#producerContract										接收共识层消息，并将消息传递给onroad
- ledger/onroad/manager.go#producerStartEventFunc							onroad内部启动协程，单独处理待出块的send block
- ledger/onroad/contract.go#ContractWorker.Start							将所有合约按照配额进行排序，依次进行出块
- ledger/onroad/taskprocessor.go#ContractTaskProcessor.work					从已经排序好的队列中取出某个合约地址，进行处理
- ledger/onroad/taskprocessor.go#ContractTaskProcessor.processOneAddress 	针对该合约进行处理
- ledger/generator/generator.go#GenerateWithOnRoad							依据一个send block进行生成receive block
- vm/vm.go#VM.RunV2															运行vm逻辑
- ledger/onroad/access.go#insertBlockToPool									将生成好的block插入到pool中
- ledger/pool/pool.go#AddDirectAccountBlock		 							通过pool，将block直接插入到账本中

后面的流程参考用户发送一笔转账交易

## sbp生成一个snapshot block

- ledger/consensus/trigger.go#update 										产生一个snapshot block开始出块的事件，传递到下游
- producer/worker.go#produceSnapshot										接收共识层消息，并单独开辟一个协程来进行snapshot block生成
	- producer/worker.go#randomSeed											计算随机数种子
	- producer/tools.go#generateSnapshot
		- ledger/chain/unconfirmed.go#GetContentNeedSnapshot				计算有哪些account block需要快照
	- producer/tools.go#insertSnapshot										
		- ledger/pool/pool.go#AddDirectSnapshotBlock                        插入snapshot block到账本中
			- ledger/verifier/snapshot_verifier.go#VerifyReferred			验证snapshot block的合法性
			- ledger/chain/insert.go#InsertSnapshotBlock					插入snapshot block到chain中
				- ledger/chain/insert.go#insertSnapshotBlock				更新indexDB,stateDB



# 架构概括

vite协议是一个dag数据结构的协议，block分为两个部分，account block和snapshot block。
如果做一下与eth的类比：
- send account block 类似于eth的transaction
- receive account block 类似于eth的receipt
- snapshot block 类似于eth的block

用户签名account block来改变账户中的状态，单个账户的account block高度以此连接，形成一条账户链。
account block上的height相当于eth中每个account的nonce，都是代表着账户链条的高度。
在数据结构上，和eth有个很大的区别是，send block 和receive block都占用账户链的高度，而且vite的account block是以链表的形式存在，每个block引用前一个block的hash作为自己的prevHash。
