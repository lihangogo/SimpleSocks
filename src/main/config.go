package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

var (
	configPath string
	IsNeedCertification bool
	ReGenerateKey bool
	ConfigPath1 string
)

type Config struct {
	Version string `json:"version"`
	TransportLayerProtocol string `json:"transport_layer_protocol"`
	ListenAddr string `json:"listen_addr"`
	Key   string `json:"key"`
	Initialized bool `json:"initialized"`
	NeedCertification bool `json:"need_certification"`
	HeartBeatAddr string `json:"heart_beat_addr"`
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

/*
	保存配置到配置文件
*/
func (config *Config) SaveConfig() {
	configJson, _ := json.MarshalIndent(config, "", "	")
	err := ioutil.WriteFile(configPath, configJson,os.ModeAppend)
	if err != nil {
		fmt.Printf("保存配置到文件 %s 出错: %s\n", configPath, err)
	}else{
		log.Printf("保存配置到文件 %s 成功\n", configPath)
	}
}

/*
	读取配置文件
 */
func (config *Config) ReadConfig() {
	// 如果配置文件存在，就读取配置文件中的配置 assign 到 config
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		log.Printf("从文件 %s 中读取配置\n", configPath)
		err=load(configPath, &config)
		if err!=nil{
			log.Println("读取配置文件失败")
		}
		IsNeedCertification=config.NeedCertification
	}
}

/*
	json文件加载
 */
func load(fileName string, v interface{}) error {
	data, err:=ioutil.ReadFile(fileName)
	if err!=nil{
		fmt.Println(err)
		return err
	}
	err=json.Unmarshal(data, v)
	return  err
}

func (config *Config) String() string{
	return fmt.Sprintf("使用传输层协议: %s\n" +
		"本地监听地址: %s\n" +
		"密钥:%s\n",
		config.TransportLayerProtocol,
		config.ListenAddr,
		config.Key)
}

/*
	判断配置信息是否有缺项，赋默认值
 */
func (config * Config) JudgeConfigSituation()  {
	if !config.Initialized{
		config.Key=RandKey()
		config.Initialized=true
	}else {
		if ReGenerateKey{
			config.Key=RandKey()
			log.Println("重新生成密钥成功")
		}
	}
	if config.ListenAddr==""{
		config.ListenAddr=":6333"
	}
	if config.TransportLayerProtocol==""{
		config.TransportLayerProtocol="tcp"
	}
}

/*
	初始化配置信息
 */
func (config *Config) Init(){
	configPath=ConfigPath1
	config.ReadConfig()
	config.JudgeConfigSituation()
	config.SaveConfig()
}


