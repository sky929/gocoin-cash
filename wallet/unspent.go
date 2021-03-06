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

// File:        unspent.go
// Description: Bictoin Cash Cash main Package

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

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	bch "github.com/counterpartyxcpc/gocoin-cash/lib/bch"
)

type unspRec struct {
	bch.TxPrevOut
	label string
	key   *bch.PrivateAddr
	spent bool
}

var (
	// set in load_balance():
	unspentOuts []*unspRec
)

func (u *unspRec) String() string {
	return fmt.Sprint(u.TxPrevOut.String(), " ", u.label)
}

func NewUnspRec(l []byte) (uns *unspRec) {
	if l[64] != '-' {
		return nil
	}

	txid := bch.NewUint256FromString(string(l[:64]))
	if txid == nil {
		return nil
	}

	rst := strings.SplitN(string(l[65:]), " ", 2)
	vout, e := strconv.ParseUint(rst[0], 10, 32)
	if e != nil {
		return nil
	}

	uns = new(unspRec)
	uns.TxPrevOut.Hash = txid.Hash
	uns.TxPrevOut.Vout = uint32(vout)
	if len(rst) > 1 {
		uns.label = rst[1]
	}

	return
}

// load the content of the "balance/" folder
func load_balance() error {
	f, e := os.Open("balance/unspent.txt")
	if e != nil {
		return e
	}
	rd := bufio.NewReader(f)
	for {
		l, _, e := rd.ReadLine()
		if len(l) == 0 && e != nil {
			break
		}
		if uns := NewUnspRec(l); uns != nil {
			if uns.key == nil {
				uns.key = pkscr_to_key(getUO(&uns.TxPrevOut).Pk_script)
			}
			unspentOuts = append(unspentOuts, uns)
		} else {
			println("ERROR in unspent.txt: ", string(l))
		}
	}
	f.Close()
	return nil
}

func show_balance() {
	var totBtc, msBtc, knownInputs, unknownInputs, multisigInputs uint64
	for i := range unspentOuts {
		uo := getUO(&unspentOuts[i].TxPrevOut)

		if unspentOuts[i].key != nil {
			totBtc += uo.Value
			knownInputs++
			continue
		}

		if bch.IsP2SH(uo.Pk_script) {
			msBtc += uo.Value
			multisigInputs++
			continue
		}

		unknownInputs++
		if *verbose {
			fmt.Println("WARNING: Don't know how to sign", unspentOuts[i].TxPrevOut.String())
		}
	}
	fmt.Printf("You have %.8f BCH in %d keyhash outputs\n", float64(totBtc)/1e8, knownInputs)
	if multisigInputs > 0 {
		fmt.Printf("There is %.8f BCH in %d multisig outputs\n", float64(msBtc)/1e8, multisigInputs)
	}
	if unknownInputs > 0 {
		fmt.Println("WARNING:", unknownInputs, "unspendable inputs (-v to print them).")
	}
}

// apply the chnages to the balance folder
func apply_to_balance(tx *bch.Tx) {
	f, _ := os.Create("balance/unspent.txt")
	if f != nil {
		// append new outputs at the end of unspentOuts
		ioutil.WriteFile("balance/"+tx.Hash.String()+".tx", tx.Serialize(), 0600)

		fmt.Println("Adding", len(tx.TxOut), "new output(s) to the balance/ folder...")
		for out := range tx.TxOut {
			if k := pkscr_to_key(tx.TxOut[out].Pk_script); k != nil {
				uns := new(unspRec)
				uns.key = k
				uns.TxPrevOut.Hash = tx.Hash.Hash
				uns.TxPrevOut.Vout = uint32(out)
				uns.label = fmt.Sprint("# ", bch.UintToBtc(tx.TxOut[out].Value), " BCH @ ", k.BtcAddr.String())
				unspentOuts = append(unspentOuts, uns)
			}
		}

		for j := range unspentOuts {
			if !unspentOuts[j].spent {
				fmt.Fprintln(f, unspentOuts[j].String())
			}
		}
		f.Close()
	} else {
		println("ERROR: Cannot create balance/unspent.txt")
	}
}
