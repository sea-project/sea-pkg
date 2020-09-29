package bip44

// Copyright 2016 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

import (
	"github.com/sea-project/sea-pkg/crypto/bip32"
	"github.com/sea-project/sea-pkg/crypto/bip39"
)

const (
	Purpose   uint32 = 0x8000002C
	Purpose45 uint32 = 0x8000002d
)

// https://github.com/satoshilabs/slips/blob/master/slip-0044.md
func NewKeyFromMnemonic(mnemonic string, purpose, coinType, org, account, chain, address uint32) (*bip32.Key, error) {
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, err
	}

	// 主私钥
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, err
	}
	// 立即生成一个子私钥
	return NewKeyFromMasterKey(masterKey, purpose, coinType, org, account, chain, address)
}

// bip44 m / purpose' / coin_type' / account' / change / address_index
// bip45 m / purpose' / algorithmType'/ orgOrCoinType' / account' / change / address_index
// CKD: m: 使用 CKDpriv, M 则表示使用 CKDPub
// purpose 根据BIP43建议将常量设置为44'（或0x8000002C）。它指示根据此规范使用了此节点的子树
// org 组织
// coinType 币种
// account  将密钥空间划分为独立的用户身份
// change 0用于外部接收地址 1用于找零地址
// addressIndex  地址索引

func NewKeyFromMasterKey(masterKey *bip32.Key, purpose, coinType, org, account, change, addressIndex uint32) (*bip32.Key, error) {
	child, err := masterKey.NewChildKey(purpose)
	if err != nil {
		return nil, err
	}

	child, err = child.NewChildKey(coinType)
	if err != nil {
		return nil, err
	}

	if purpose != Purpose {
		child, err = child.NewChildKey(org)
		if err != nil {
			return nil, err
		}
	}

	child, err = child.NewChildKey(account)
	if err != nil {
		return nil, err
	}

	child, err = child.NewChildKey(change)
	if err != nil {
		return nil, err
	}

	child, err = child.NewChildKey(addressIndex)
	if err != nil {
		return nil, err
	}

	return child, nil
}
