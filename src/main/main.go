package main

import (
	"flag"
	"log"
	"net"
)

func init()  {
	flag.BoolVar(&ReGenerateKey,"r",false,"generate new key")
	flag.StringVar(&ConfigPath1,"c","conf.json",
		"Specify the configuration file path")
}

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile)
	// 若首次启动，则生成密钥; 非首次启动，延用原密钥
	config:=&Config{}
	config.Init()

	// 启动 server 端并监听
	ssServer, err := NewSsServer(config.TransportLayerProtocol,config.Key, config.ListenAddr)
	if err != nil {
		log.Fatalln(err)
	}

	// 启动心跳检测
	go ssServer.HeartBeat(config.HeartBeatAddr)

	// 设置监听建立的回调函数
	log.Fatalln(ssServer.Listen(func(listenAddr net.Addr) {
		log.Println(config.String())
		log.Printf("SimpleSocks-Server%s 启动成功 监听地址:  %s\n", config.Version, listenAddr.String())
	}))
}

