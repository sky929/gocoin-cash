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

// File:		bch_block_check.go
// Description:	Bictoin Cash bch_chain Package

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

package bch_chain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	bch "github.com/counterpartyxcpc/gocoin-cash/lib/bch"
	"github.com/counterpartyxcpc/gocoin-cash/lib/script"
)

var (
	DBG_SCR = false
)

// Make sure to call this function with ch.BchBlockIndexAccess locked
func (ch *Chain) BchPreCheckBlock(bl *bch.BchBlock) (er error, dos bool, maybelater bool) {

	// Debugging Output (Optional)
	if DBG_SCR {
		fmt.Println("Raw block data:", bl.Raw)
	}

	// Size limitsn (NOTE: This is more a BCH )
	// @todo [BCH] Bitcoin Cash - Size Fri Sep 21, 2018 - Julian Smith
	if len(bl.Raw) < 81 {
		er = errors.New("CheckBlock() : size limits failed - RPC_Result:bad-blk-length")
		dos = true
		return
	}

	// Debugging Output (Optional)
	if DBG_SCR {
		fmt.Println("Block version:", bl.Version())
	}

	ver := bl.Version()
	if ver == 0 {
		er = errors.New("CheckBlock() : Block version 0 not allowed - RPC_Result:bad-version")
		dos = true
		return
	}

	// Check proof-of-work
	if !bch.CheckProofOfWork(bl.Hash, bl.Bits()) {
		er = errors.New("CheckBlock() : proof of work failed - RPC_Result:high-hash")
		dos = true
		return
	}

	// Check timestamp (must not be higher than now +2 hours)
	if int64(bl.BchBlockTime()) > time.Now().Unix()+2*60*60 {
		er = errors.New("CheckBlock() : block timestamp too far in the future - RPC_Result:time-too-new")
		dos = true
		return
	}

	if prv, pres := ch.BchBlockIndex[bl.Hash.BIdx()]; pres {
		if prv.Parent == nil {
			// This is genesis block
			er = errors.New("Genesis")
			return
		} else {
			er = errors.New("CheckBlock: " + bl.Hash.String() + " already in - RPC_Result:duplicate")
			return
		}
	}

	prevblk, ok := ch.BchBlockIndex[bch.NewUint256(bl.ParentHash()).BIdx()]
	if !ok {
		er = errors.New("Newblk 2hr+ CheckBlock: " + bl.Hash.String() + " parent not found - RPC_Result:bad-prevblk")
		maybelater = true
		return
	}

	bl.Height = prevblk.Height + 1

	// Reject the block if it reaches into the chain deeper than our unwind buffer
	lst_now := ch.LastBlock()
	if prevblk != lst_now && int(lst_now.Height)-int(bl.Height) >= MovingCheckopintDepth {
		er = errors.New(fmt.Sprint("CheckBlock: bch.BchBlock ", bl.Hash.String(),
			" hooks too deep into the chain: ", bl.Height, "/", lst_now.Height, " ",
			bch.NewUint256(bl.ParentHash()).String(), " - RPC_Result:bad-prevblk"))
		return
	}

	// Check proof of work
	gnwr := ch.GetNextWorkRequired(prevblk, bl.BchBlockTime())
	if bl.Bits() != gnwr {
		er = errors.New("CheckBlock: incorrect proof of work - RPC_Result:bad-diffbits")
		dos = true
		return
	}

	// Check timestamp against prev
	bl.MedianPastTime = prevblk.GetMedianTimePast()
	if bl.BchBlockTime() <= bl.MedianPastTime {
		er = errors.New("CheckBlock: block's timestamp is too early - RPC_Result:time-too-old")
		dos = true
		return
	}

	if ver < 2 && bl.Height >= ch.Consensus.BIP34Height ||
		ver < 3 && bl.Height >= ch.Consensus.BIP66Height ||
		ver < 4 && bl.Height >= ch.Consensus.BIP65Height {
		// bad block version
		erstr := fmt.Sprintf("0x%08x", ver)
		er = errors.New("CheckBlock() : Rejected Version=" + erstr + " block - RPC_Result:bad-version(" + erstr + ")")
		dos = true
		return
	}

	// Debugging Output (Optional)
	if DBG_SCR {
		fmt.Println("Consensus.Enforce_UAHF: ", ch.Consensus.Enforce_UAHF)
	}

	if ch.Consensus.Enforce_UAHF != 0 {
		if bl.Height >= ch.Consensus.Enforce_UAHF {

			// if (ver&0xE0000000) != 0x20000000 || (ver&2) == 0 {
			// er = errors.New("CheckBlock() : relayed block must signal for UAHF - RPC_Result:bad-no-uahf")
			// }

			// Debugging Output (Optional)
			if DBG_SCR {
				fmt.Println("Actively evaluating UAHF block:", (ver & 0xE0000000))
			}

		}
	}

	if ch.Consensus.BIP91Height != 0 && ch.Consensus.Enforce_SEGWIT != 0 {
		if bl.Height >= ch.Consensus.BIP91Height && bl.Height < ch.Consensus.Enforce_SEGWIT-2016 {
			if (ver&0xE0000000) != 0x20000000 || (ver&2) == 0 {
				er = errors.New("CheckBlock() : relayed block must signal for segwit - RPC_Result:bad-no-segwit")
			}
		}
	}

	return
}

// Make sure to call this function with ch.BchBlockIndexAccess locked
func (ch *Chain) PreCheckBlock(bl *bch.BchBlock) (er error, dos bool, maybelater bool) {

	// Debugging Output (Optional)
	if DBG_SCR {
		fmt.Println("PreCheckBlock Raw Length:", len(bl.Raw))
	}

	// Size limits
	if len(bl.Raw) < 81 {
		er = errors.New("CheckBlock() : size limits failed - RPC_Result:bad-blk-length")
		dos = true
		return
	}

	ver := bl.Version()
	if ver == 0 {
		er = errors.New("CheckBlock() : Block version 0 not allowed - RPC_Result:bad-version")
		dos = true
		return
	}

	// Check proof-of-work
	if !bch.CheckProofOfWork(bl.Hash, bl.Bits()) {
		er = errors.New("CheckBlock() : proof of work failed - RPC_Result:high-hash")
		dos = true
		return
	}

	// Check timestamp (must not be higher than now +2 hours)
	if int64(bl.BchBlockTime()) > time.Now().Unix()+2*60*60 {
		er = errors.New("CheckBlock() : block timestamp too far in the future - RPC_Result:time-too-new")
		dos = true
		return
	}

	if prv, pres := ch.BchBlockIndex[bl.Hash.BIdx()]; pres {
		if prv.Parent == nil {
			// This is genesis block
			er = errors.New("Genesis")
			return
		} else {
			er = errors.New("CheckBlock: " + bl.Hash.String() + " already in - RPC_Result:duplicate")
			return
		}
	}

	prevblk, ok := ch.BchBlockIndex[bch.NewUint256(bl.ParentHash()).BIdx()]
	if !ok {
		er = errors.New("2hr+ CheckBlock: " + bl.Hash.String() + " parent not found - RPC_Result:bad-prevblk")
		maybelater = true
		return
	}

	bl.Height = prevblk.Height + 1

	// Reject the block if it reaches into the chain deeper than our unwind buffer
	lst_now := ch.LastBlock()
	if prevblk != lst_now && int(lst_now.Height)-int(bl.Height) >= MovingCheckopintDepth {
		er = errors.New(fmt.Sprint("CheckBlock: bch.BchBlock ", bl.Hash.String(),
			" hooks too deep into the chain: ", bl.Height, "/", lst_now.Height, " ",
			bch.NewUint256(bl.ParentHash()).String(), " - RPC_Result:bad-prevblk"))
		return
	}

	// Check proof of work
	gnwr := ch.GetNextWorkRequired(prevblk, bl.BchBlockTime())
	if bl.Bits() != gnwr {
		er = errors.New("CheckBlock: incorrect proof of work - RPC_Result:bad-diffbits")
		dos = true
		return
	}

	// Check timestamp against prev
	bl.MedianPastTime = prevblk.GetMedianTimePast()
	if bl.BchBlockTime() <= bl.MedianPastTime {
		er = errors.New("CheckBlock: block's timestamp is too early - RPC_Result:time-too-old")
		dos = true
		return
	}

	if ver < 2 && bl.Height >= ch.Consensus.BIP34Height ||
		ver < 3 && bl.Height >= ch.Consensus.BIP66Height ||
		ver < 4 && bl.Height >= ch.Consensus.BIP65Height {
		// bad block version
		erstr := fmt.Sprintf("0x%08x", ver)
		er = errors.New("CheckBlock() : Rejected Version=" + erstr + " block - RPC_Result:bad-version(" + erstr + ")")
		dos = true
		return
	}

	if ch.Consensus.BIP91Height != 0 && ch.Consensus.Enforce_SEGWIT != 0 {
		if bl.Height >= ch.Consensus.BIP91Height && bl.Height < ch.Consensus.Enforce_SEGWIT-2016 {
			if (ver&0xE0000000) != 0x20000000 || (ver&2) == 0 {
				er = errors.New("CheckBlock() : relayed block must signal for segwit - RPC_Result:bad-no-segwit")
			}
		}
	}

	return
}

func (ch *Chain) ApplyBlockFlags(bl *bch.BchBlock) {

	// Debugging Output (Optional)
	if DBG_SCR {
		fmt.Println("Applying block flag.")
	}

	if bl.BchBlockTime() >= BIP16SwitchTime {
		bl.VerifyFlags = script.VER_P2SH
	} else {
		bl.VerifyFlags = 0
	}

	if bl.Height >= ch.Consensus.BIP66Height {
		bl.VerifyFlags |= script.VER_DERSIG
	}

	if bl.Height >= ch.Consensus.BIP65Height {
		bl.VerifyFlags |= script.VER_CLTV
	}

	if ch.Consensus.Enforce_CSV != 0 && bl.Height >= ch.Consensus.Enforce_CSV {
		bl.VerifyFlags |= script.VER_CSV
	}

	if ch.Consensus.Enforce_UAHF != 0 && bl.Height > ch.Consensus.Enforce_UAHF {
		bl.VerifyFlags |= script.VER_UAHF | script.VER_STRICTENC
	}

	if ch.Consensus.Enforce_SEGWIT != 0 && bl.Height >= ch.Consensus.Enforce_SEGWIT {
		bl.VerifyFlags |= script.VER_WITNESS | script.VER_NULLDUMMY
	}

	// Debugging Output (Optional)
	if DBG_SCR {
		fmt.Println("Block flags are: ", bl.VerifyFlags)
	}

}

func (ch *Chain) PostCheckBlock(bl *bch.BchBlock) (er error) {
	// Size limits
	if len(bl.Raw) < 81 {
		er = errors.New("CheckBlock() : size limits failed low - RPC_Result:bad-blk-length")
		return
	}

	if bl.Txs == nil {
		er = bl.BuildTxList()
		if er != nil {
			return
		}
		if bl.BchBlockWeight > ch.MaxBlockWeight(bl.Height) {
			er = errors.New("CheckBlock() : weight limits failed - RPC_Result:bad-blk-weight")
			return
		}
		//fmt.Println("New block", bl.Height, " Weight:", bl.BchBlockWeight, " Raw:", len(bl.Raw))
	}

	if !bl.Trusted {
		// We need to be satoshi compatible
		if len(bl.Txs) == 0 || !bl.Txs[0].IsCoinBase() {
			er = errors.New("CheckBlock() : first tx is not coinbase: " + bl.Hash.String() + " - RPC_Result:bad-cb-missing")
			return
		}

		// Enforce rule that the coinbase starts with serialized block height
		if bl.Height >= ch.Consensus.BIP34Height {
			var exp [6]byte
			var exp_len int
			binary.LittleEndian.PutUint32(exp[1:5], bl.Height)
			for exp_len = 5; exp_len > 1; exp_len-- {
				if exp[exp_len] != 0 || exp[exp_len-1] >= 0x80 {
					break
				}
			}
			exp[0] = byte(exp_len)
			exp_len++

			if !bytes.HasPrefix(bl.Txs[0].TxIn[0].ScriptSig, exp[:exp_len]) {
				er = errors.New("CheckBlock() : Unexpected block number in coinbase: " + bl.Hash.String() + " - RPC_Result:bad-cb-height")
				return
			}
		}

		// And again...
		for i := 1; i < len(bl.Txs); i++ {
			if bl.Txs[i].IsCoinBase() {
				er = errors.New("CheckBlock() : more than one coinbase: " + bl.Hash.String() + " - RPC_Result:bad-cb-multiple")
				return
			}
		}
	}

	// Check Merkle Root, even for trusted blocks - that's important, as they may come from untrusted peers
	merkle, mutated := bl.GetMerkle()
	if mutated {
		er = errors.New("CheckBlock(): duplicate transaction - RPC_Result:bad-txns-duplicate")
		return
	}

	if !bytes.Equal(merkle, bl.MerkleRoot()) {
		er = errors.New("CheckBlock() : Merkle Root mismatch - RPC_Result:bad-txnmrklroot")
		return
	}

	ch.ApplyBlockFlags(bl)

	if !bl.Trusted {
		var blockTime uint32
		var had_witness bool

		if (bl.VerifyFlags & script.VER_CSV) != 0 {
			blockTime = bl.MedianPastTime
		} else {
			blockTime = bl.BchBlockTime()
		}

		// Verify merkle root of witness data
		if (bl.VerifyFlags & script.VER_WITNESS) != 0 {
			var i int
			for i = len(bl.Txs[0].TxOut) - 1; i >= 0; i-- {
				o := bl.Txs[0].TxOut[i]
				if len(o.Pk_script) >= 38 && bytes.Equal(o.Pk_script[:6], []byte{0x6a, 0x24, 0xaa, 0x21, 0xa9, 0xed}) {
					if len(bl.Txs[0].SegWit) != 1 || len(bl.Txs[0].SegWit[0]) != 1 || len(bl.Txs[0].SegWit[0][0]) != 32 {
						er = errors.New("CheckBlock() : invalid witness nonce size - RPC_Result:bad-witness-nonce-size")
						println(er.Error())
						println(bl.Hash.String(), len(bl.Txs[0].SegWit))
						return
					}

					// The malleation check is ignored; as the transaction tree itself
					// already does not permit it, it is impossible to trigger in the
					// witness tree.
					merkle, _ := bch.GetWitnessMerkle(bl.Txs)
					with_nonce := bch.Sha2Sum(append(merkle, bl.Txs[0].SegWit[0][0]...))

					if !bytes.Equal(with_nonce[:], o.Pk_script[6:38]) {
						er = errors.New("CheckBlock(): Witness Merkle mismatch - RPC_Result:bad-witness-merkle-match")
						return
					}

					had_witness = true
					break
				}
			}
		}

		if !had_witness {
			for _, t := range bl.Txs {
				if t.SegWit != nil {
					er = errors.New("CheckBlock(): unexpected witness data found - RPC_Result:unexpected-witness")
					return
				}
			}
		}

		// Check transactions - this is the most time consuming task
		er = CheckTransactions(bl.Txs, bl.Height, blockTime)
	}
	return
}

func (ch *Chain) CheckBlock(bl *bch.BchBlock) (er error, dos bool, maybelater bool) {
	er, dos, maybelater = ch.PreCheckBlock(bl)
	if er == nil {
		er = ch.PostCheckBlock(bl)
		if er != nil { // all post-check errors are DoS kind
			dos = true
		}
	}
	return
}
