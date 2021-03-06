// ======================================================================

//      cccccccccc          pppppppppp
//    cccccccccccccc      pppppppppppppp
//  ccccccccccccccc    ppppppppppppppppppp
// cccccc       cc    ppppppp        pppppp
// cccccc          pppppppp          pppppp
// cccccc        ccccpppp            pppppp
// cccccccc    cccccccc    pppp    ppppppp
//  ccccccccccccccccc     ppppppppppppppp
//     cccccccccccc      pppppppppppppp
//       cccccccc        pppppppppppp
//                       pppppp
//                       pppppp

// ======================================================================
// Copyright © 2018. Counterparty Cash Association (CCA) Zug, CH.
// All Rights Reserved. All work owned by CCA is herby released
// under Creative Commons Zero (0) License.

// Some rights of 3rd party, derivative and included works remain the
// property of thier respective owners. All marks, brands and logos of
// member groups remain the exclusive property of their owners and no
// right or endorsement is conferred by reference to thier organization
// or brand(s) by CCA.

// File:		onoff.go
// Description:	Bictoin Cash wallet Package

// Credits:

// Piotr Narewski, Gocoin Founder

// Julian Smith, Direction + Development
// Arsen Yeremin, Development
// Sumanth Kumar, Development
// Clayton Wong, Development
// Liming Jiang, Development

// Includes reference work of btsuite:

// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2018 The bcext developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Credits:

// Piotr Narewski, Gocoin Founder

// Julian Smith, Direction + Development
// Arsen Yeremin, Development
// Sumanth Kumar, Development
// Clayton Wong, Development
// Liming Jiang, Development

// Includes reference work of btsuite:

// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2018 The bcext developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Includes reference work of Bitcoin Core (https://github.com/bitcoin/bitcoin)
// Includes reference work of Bitcoin-ABC (https://github.com/Bitcoin-ABC/bitcoin-abc)
// Includes reference work of Bitcoin Unlimited (https://github.com/BitcoinUnlimited/BitcoinUnlimited/tree/BitcoinCash)
// Includes reference work of gcash by Shuai Qi "qshuai" (https://github.com/bcext/gcash)
// Includes reference work of gcash (https://github.com/gcash/bchd)

// + Other contributors

// =====================================================================

package wallet

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"

	"github.com/counterpartyxcpc/gocoin-cash/client/common"
	"github.com/counterpartyxcpc/gocoin-cash/lib/bch_utxo"
)

var (
	FetchingBalanceTick func() bool
	OnOff               chan bool = make(chan bool, 1)
)

func InitMaps(empty bool) {
	var szs [4]int
	var ok bool

	if empty {
		goto init
	}

	LoadMapSizes()
	szs, ok = WalletAddrsCount[common.AllBalMinVal()]
	if ok {
		//fmt.Println("Have map sizes for MinBal", common.AllBalMinVal(), ":", szs[0], szs[1], szs[2], szs[3])
	} else {
		//fmt.Println("No map sizes for MinBal", common.AllBalMinVal())
		szs = [4]int{10e6, 3e6, 10e3, 1e3} // defaults
	}

init:
	AllBalancesP2KH = make(map[[20]byte]*OneAllAddrBal, szs[0])
	AllBalancesP2SH = make(map[[20]byte]*OneAllAddrBal, szs[1])
	AllBalancesP2WKH = make(map[[20]byte]*OneAllAddrBal, szs[2])
	AllBalancesP2WSH = make(map[[32]byte]*OneAllAddrBal, szs[3])
}

func LoadBalance() {
	if common.GetBool(&common.WalletON) {
		//fmt.Println("wallet.LoadBalance() ignore: ", common.GetBool(&common.WalletON))
		return
	}

	var aborted bool

	common.SetUint32(&common.WalletProgress, 1)
	common.ApplyBalMinVal()

	InitMaps(false)

	common.BchBlockChain.Unspent.RWMutex.RLock()
	defer common.BchBlockChain.Unspent.RWMutex.RUnlock()

	cnt_dwn_from := (len(common.BchBlockChain.Unspent.HashMap) + 999) / 1000
	cnt_dwn := cnt_dwn_from
	perc := uint32(1)

	for k, v := range common.BchBlockChain.Unspent.HashMap {
		NewUTXO(utxo.NewUtxoRecStatic(k, v))
		if cnt_dwn == 0 {
			perc++
			common.SetUint32(&common.WalletProgress, perc)
			cnt_dwn = cnt_dwn_from
		} else {
			cnt_dwn--
		}
		if FetchingBalanceTick != nil && FetchingBalanceTick() {
			aborted = true
			break
		}
	}
	if aborted {
		InitMaps(true)
	} else {
		common.BchBlockChain.Unspent.CB.NotifyTxAdd = TxNotifyAdd
		common.BchBlockChain.Unspent.CB.NotifyTxDel = TxNotifyDel
		common.SetBool(&common.WalletON, true)
	}
	common.SetUint32(&common.WalletProgress, 0)
}

func Disable() {
	if !common.GetBool(&common.WalletON) {
		//fmt.Println("wallet.Disable() ignore: ", common.GetBool(&common.WalletON))
		return
	}
	UpdateMapSizes()
	common.BchBlockChain.Unspent.CB.NotifyTxAdd = nil
	common.BchBlockChain.Unspent.CB.NotifyTxDel = nil
	common.SetBool(&common.WalletON, false)
	InitMaps(true)
}

const (
	MAPSIZ_FILE_NAME = "mapsize.gob"
)

var (
	WalletAddrsCount map[uint64][4]int = make(map[uint64][4]int) //index:MinValue, [0]-P2KH, [1]-P2SH, [2]-P2WSH, [3]-P2WKH
)

func UpdateMapSizes() {
	WalletAddrsCount[common.AllBalMinVal()] = [4]int{len(AllBalancesP2KH),
		len(AllBalancesP2SH), len(AllBalancesP2WKH), len(AllBalancesP2WSH)}

	buf := new(bytes.Buffer)
	gob.NewEncoder(buf).Encode(WalletAddrsCount)
	ioutil.WriteFile(common.GocoinCashHomeDir+MAPSIZ_FILE_NAME, buf.Bytes(), 0600)
}

func LoadMapSizes() {
	d, er := ioutil.ReadFile(common.GocoinCashHomeDir + MAPSIZ_FILE_NAME)
	if er != nil {
		println("LoadMapSizes:", er.Error())
		return
	}

	buf := bytes.NewBuffer(d)

	er = gob.NewDecoder(buf).Decode(&WalletAddrsCount)
	if er != nil {
		println("LoadMapSizes:", er.Error())
	}
}
