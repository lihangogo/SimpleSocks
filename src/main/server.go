package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"
	"net"
	"os/exec"
	"strings"
	"sync"
)

type SsServer struct {
	Cipher     *cipher
	ListenAddr *net.TCPAddr
	Mutex *sync.RWMutex
	NetworkAvailable bool
}

/*
	创建SimpleSocks-Server
		1、根据字符串密钥生成字节密钥
		2、包装TCP模式监听地址 (握手阶段暂定使用TCP协议)
 */
func NewSsServer(tlp string, key string, listenAddr string) (*SsServer, error) {
	byteKey, err := parseKey(key)
	if err != nil {
		return nil, err
	}
	ConnectionListenAddr, err := net.ResolveTCPAddr(tlp, listenAddr)
	if err != nil {
		return nil, err
	}
	return &SsServer{
		Cipher:     newCipher(byteKey),
		ListenAddr: ConnectionListenAddr,
		Mutex:new(sync.RWMutex),
	}, nil
}

/*
	心跳检测
	根据ping的返回信息判断网络通断情况，用读写锁限制写操作
 */
func (ssServer *SsServer) HeartBeat(HeartBeatAddress string)  {
	command:=HeartBeatAddress
	// 设置发包间隔即可 5秒探测一次
	// 探测频率不易过高，会导致本goroutine占用锁的频率过高，执行效率降低
	cmd:=exec.Command("ping","-i", "5",command)
	stdout,err:=cmd.StdoutPipe()
	err=cmd.Start()
	log.Println("心跳检测启动成功")
	if err!=nil{
		log.Println("联网状态-命令执行失败")
	}
	reader:=bufio.NewReader(stdout)
	for {
		line,err2:=reader.ReadString('\n')
		if err2!=nil || io.EOF==err2{
			break
		}
		ssServer.Mutex.Lock()
		if strings.Contains(line,"ttl");strings.Contains(line,"ms"){
			ssServer.NetworkAvailable=true
		}else{
			ssServer.NetworkAvailable=false
		}
		ssServer.Mutex.Unlock()
	}
	err=cmd.Wait()
}

/*
	获取当前网络状态，限制写，不限制读
 */
func (ssServer *SsServer) GetNetworkStatus() bool {
	ssServer.Mutex.RLock()
	defer ssServer.Mutex.RUnlock()
	return ssServer.NetworkAvailable
}

/*
	调用监听SimpleSocks客户端请求的监听器
 */
func (ssServer *SsServer) Listen(doListen func(listenAddr net.Addr)) error {
	return ListenSecureTCP(ssServer.ListenAddr, ssServer.Cipher, ssServer.handleConn, doListen)
}

/*
	根据SOCKS5规范解析连接请求消息
 */
func (ssServer *SsServer) handleConn(localConn *SecureTCPConn) {
	defer localConn.Close()
	buf := make([]byte, 257)

	// 第一个字段VER代表Socks的版本，Socks5默认为0x05，其固定长度为1个字节
	_, err := localConn.DecodeRead(buf)
	if err != nil {
		log.Println(err)
		return
	}
	if buf[0]!=0x05{
		log.Println("SOCKS协议版本错误")
		// return
	}

	// 判断网络连接状态
	if !ssServer.GetNetworkStatus(){
		log.Println("网络连接异常")
		_, _=localConn.EncodeWrite([]byte{0x05, 0xFF})
		return
	}

	// 需要验证
	if IsNeedCertification{
		_, _=localConn.EncodeWrite([]byte{0x05, 0x02})
		pass:=make([]byte,3)
		_, err=localConn.DecodeRead(pass)
		if err!=nil{
			log.Println("密码协商通信异常")
			return
		}
		if pass[0]!=0x05{
			log.Println("SOCKS协议版本错误")
			return
		}
		if !(pass[1]==0x06&&pass[2]==0x06){
			log.Println("密码错误")
			return
		}
		_, _=localConn.EncodeWrite([]byte{0x05,0x06,0x06})
	}else {   //不需要验证
		_, _=localConn.EncodeWrite([]byte{0x05, 0x00})
	}

	// 获取真正的远程服务的地址
	n, err := localConn.DecodeRead(buf)
	if err != nil {
		log.Println("获取客户端远程访问请求失败")
		return
	}

	// 暂仅支持 CONNECT类型
	if buf[1] != 0x01 {
		log.Println("客户端请求类型无效")
		return
	}

	var dstIP []byte
	// 第四个字节代表请求的远程服务器地址类型
	switch buf[3] {
	case 0x01:
		// IPv4
		dstIP = buf[4 : 4+net.IPv4len]
	case 0x03:
		// 域名解析IP地址
		ipAddr, err := net.ResolveIPAddr("ip", string(buf[5:n-2]))
		if err != nil {
			return
		}
		dstIP = ipAddr.IP
	case 0x04:
		//	IPv6
		dstIP = buf[4 : 4+net.IPv6len]
	default:
		log.Println("请求远程服务器地址类型异常")
		return
	}
	dPort := buf[n-2:]
	dstAddr := &net.TCPAddr{
		IP:   dstIP,
		Port: int(binary.BigEndian.Uint16(dPort)),
	}

	// 转发请求，连接真正要访问的远程服务器
	dstServer, err := net.DialTCP("tcp", nil, dstAddr)
	if err != nil {
		log.Println("访问远程服务器失败")
		return
	} else {
		defer dstServer.Close()
		// 任何未发送或者未确认的数据丢弃
		_ =dstServer.SetLinger(0)
		// 响应客户端连接成功
		_, _=localConn.EncodeWrite([]byte{0x05, 0x00, 0x00, buf[3], 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	}

	// 不设置某种连接容错，若任何一端读取/发送数据失败，则认定此次代理断开，清除两端的连接
	// 从 客户端 连接中读取请求数据，解密后发送到 远程服务器
	go func() {
		err := localConn.DecodeCopy(dstServer)
		if err != nil {
			_=localConn.Close()
			_=dstServer.Close()
		}
	}()
	// 从 远程服务器 读取应答数据发送到 客户端
	go func() {
		localConn2:=&SecureTCPConn{
			Cipher:          localConn.Cipher,
			ReadWriteCloser: dstServer,
		}
		err:=localConn2.EncodeCopy(localConn)
		if err!=nil{
			_=localConn2.Close()
			_=localConn.Close()
		}
	}()
}
