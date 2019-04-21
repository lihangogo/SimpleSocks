package main

import (
	"encoding/base64"
	"errors"
	"math/rand"
	"strings"
	"time"
)

const keyLength = 256

type key [keyLength]byte

func init() {
	// 重置种子
	rand.Seed(time.Now().Unix())
}

/*
	采用base64编码把密钥转换为字符串
*/
func (key *key) String() string {
	return base64.StdEncoding.EncodeToString(key[:])
}

/*
	解析采用base64编码的字符串获取密钥
 */
func parseKey(keyString string) (*key, error) {
	bs, err := base64.StdEncoding.DecodeString(strings.TrimSpace(keyString))
	if err != nil || len(bs) != keyLength {
		return nil, errors.New("不合法的密码")
	}
	key := key{}
	copy(key[:], bs)
	bs = nil
	return &key, nil
}

/*
	0-255的随机编排，返回base64编码后的字符串
 */
func RandKey() string {
	// 随机生成一个由  0~255 组成的 byte 数组
	intArr := rand.Perm(keyLength)
	key := &key{}
	for i, v := range intArr {
		key[i] = byte(v)
		if i == v {
			// 确保不会出现如何一个byte位出现重复
			return RandKey()
		}
	}
	return key.String()
}
