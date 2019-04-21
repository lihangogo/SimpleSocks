package main

type cipher struct {
	// 编码用的密码
	encodeKey *key
	// 解码用的密码
	decodeKey *key
}

// 加密原数据
func (cipher *cipher) encode(bs []byte) {
	for i, v := range bs {
		bs[i] = cipher.encodeKey[v]
	}
}

// 解码加密后的数据到原数据
func (cipher *cipher) decode(bs []byte) {
	for i, v := range bs {
		bs[i] = cipher.decodeKey[v]
	}
}

// 新建一个编码解码器
func newCipher(encodeKey *key) *cipher {
	decodeKey := &key{}
	for i, v := range encodeKey {
		encodeKey[i] = v
		decodeKey[v] = byte(i)
	}
	return &cipher{
		encodeKey: encodeKey,
		decodeKey: decodeKey,
	}
}
