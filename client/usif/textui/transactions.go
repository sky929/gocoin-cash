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

// File:		tranction.go
// Description:	Bictoin Cash textui Package

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

package textui

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/counterpartyxcpc/gocoin-cash/client/common"
	"github.com/counterpartyxcpc/gocoin-cash/client/network"
	"github.com/counterpartyxcpc/gocoin-cash/client/usif"
	bch "github.com/counterpartyxcpc/gocoin-cash/lib/bch"
)

func load_tx(par string) {
	if par == "" {
		fmt.Println("Specify a name of a transaction file")
		return
	}
	f, e := os.Open(par)
	if e != nil {
		println(e.Error())
		return
	}
	n, _ := f.Seek(0, os.SEEK_END)
	f.Seek(0, os.SEEK_SET)
	buf := make([]byte, n)
	f.Read(buf)
	f.Close()
	fmt.Println(usif.LoadRawTx(buf))
}

func send_tx(par string) {
	txid := bch.NewUint256FromString(par)
	if txid == nil {
		fmt.Println("You must specify a valid transaction ID for this command.")
		list_txs("")
		return
	}
	network.TxMutex.Lock()
	if ptx, ok := network.TransactionsToSend[txid.BIdx()]; ok {
		network.TxMutex.Unlock()
		cnt := network.NetRouteInv(1, txid, nil)
		ptx.Invsentcnt += cnt
		fmt.Println("INV for TxID", txid.String(), "sent to", cnt, "node(s)")
		fmt.Println("If it does not appear in the chain, you may want to redo it.")
	} else {
		network.TxMutex.Unlock()
		fmt.Println("No such transaction ID in the memory pool.")
		list_txs("")
	}
}

func send1_tx(par string) {
	txid := bch.NewUint256FromString(par)
	if txid == nil {
		fmt.Println("You must specify a valid transaction ID for this command.")
		list_txs("")
		return
	}
	network.TxMutex.Lock()
	if ptx, ok := network.TransactionsToSend[txid.BIdx()]; ok {
		network.TxMutex.Unlock()
		usif.SendInvToRandomPeer(1, txid)
		ptx.Invsentcnt++
		fmt.Println("INV for TxID", txid.String(), "sent to a random node")
		fmt.Println("If it does not appear in the chain, you may want to redo it.")
	} else {
		network.TxMutex.Unlock()
		fmt.Println("No such transaction ID in the memory pool.")
		list_txs("")
	}
}

func del_tx(par string) {
	txid := bch.NewUint256FromString(par)
	if txid == nil {
		fmt.Println("You must specify a valid transaction ID for this command.")
		list_txs("")
		return
	}
	network.TxMutex.Lock()
	defer network.TxMutex.Unlock()
	tx, ok := network.TransactionsToSend[txid.BIdx()]
	if !ok {
		network.TxMutex.Unlock()
		fmt.Println("No such transaction ID in the memory pool.")
		list_txs("")
		return
	}
	tx.Delete(true, 0)
	fmt.Println("Transaction", txid.String(), "and all its children removed from the memory pool")
}

func dec_tx(par string) {
	txid := bch.NewUint256FromString(par)
	if txid == nil {
		fmt.Println("You must specify a valid transaction ID for this command.")
		list_txs("")
		return
	}
	if tx, ok := network.TransactionsToSend[txid.BIdx()]; ok {
		s, _, _, _, _ := usif.DecodeTx(tx.Tx)
		fmt.Println(s)
	} else {
		fmt.Println("No such transaction ID in the memory pool.")
	}
}

func save_tx(par string) {
	txid := bch.NewUint256FromString(par)
	if txid == nil {
		fmt.Println("You must specify a valid transaction ID for this command.")
		list_txs("")
		return
	}
	if tx, ok := network.TransactionsToSend[txid.BIdx()]; ok {
		fn := tx.Hash.String() + ".tx"
		ioutil.WriteFile(fn, tx.Raw, 0600)
		fmt.Println("Saved to", fn)
	} else {
		fmt.Println("No such transaction ID in the memory pool.")
	}
}

func mempool_stats(par string) {
	fmt.Print(usif.MemoryPoolFees())
}

func list_txs(par string) {
	limitbytes, _ := strconv.ParseUint(par, 10, 64)
	fmt.Println("Transactions in the memory pool:", limitbytes)
	cnt := 0
	network.TxMutex.Lock()
	defer network.TxMutex.Unlock()

	sorted := network.GetSortedMempool()

	var totlen uint64
	for cnt = 0; cnt < len(sorted); cnt++ {
		v := sorted[cnt]
		totlen += uint64(len(v.Raw))

		if limitbytes != 0 && totlen > limitbytes {
			break
		}

		var oe, snt string
		if v.Local {
			oe = " *OWN*"
		} else {
			oe = ""
		}

		snt = fmt.Sprintf("INV sent %d times,   ", v.Invsentcnt)

		if v.SentCnt == 0 {
			snt = "never sent"
		} else {
			snt = fmt.Sprintf("sent %d times, last %s ago", v.SentCnt,
				time.Now().Sub(v.Lastsent).String())
		}

		spb := float64(v.Fee) / float64(len(v.Raw))

		fmt.Println(fmt.Sprintf("%5d) ...%10d %s  %6d bytes / %6.1fspb - %s%s", cnt, totlen, v.Tx.Hash.String(), len(v.Raw), spb, snt, oe))

	}
}

func baned_txs(par string) {
	fmt.Println("Rejected transactions:")
	cnt := 0
	network.TxMutex.Lock()
	for k, v := range network.TransactionsRejected {
		cnt++
		fmt.Println("", cnt, bch.NewUint256(k[:]).String(), "-", v.Size, "bytes",
			"-", v.Reason, "-", time.Now().Sub(v.Time).String(), "ago")
	}
	network.TxMutex.Unlock()
}

func send_all_tx(par string) {
	network.TxMutex.Lock()
	for k, v := range network.TransactionsToSend {
		if v.Local {
			cnt := network.NetRouteInv(1, bch.NewUint256(k[:]), nil)
			v.Invsentcnt += cnt
			fmt.Println("INV for TxID", v.Hash.String(), "sent to", cnt, "node(s)")
		}
	}
	network.TxMutex.Unlock()
}

func save_mempool(par string) {
	network.MempoolSave(true)
}

func check_txs(par string) {
	network.TxMutex.Lock()
	network.MempoolCheck()
	network.TxMutex.Unlock()
}

func load_mempool(par string) {
	if par == "" {
		par = common.GocoinCashHomeDir + "mempool.dmp"
	}
	var abort bool
	__exit := make(chan bool)
	__done := make(chan bool)
	go func() {
		for {
			select {
			case s := <-common.KillChan:
				fmt.Println(s)
				abort = true
			case <-__exit:
				__done <- true
				return
			}
		}
	}()
	fmt.Println("Press Ctrl+C to abort...")
	network.MempoolLoadNew(par, &abort)
	__exit <- true
	_ = <-__done
	if abort {
		fmt.Println("Aborted")
	}
}

func get_mempool(par string) {
	conid, e := strconv.ParseUint(par, 10, 32)
	if e != nil {
		fmt.Println("Specify ID of the peer")
		return
	}

	fmt.Println("Getting mempool from connection ID", conid, "...")
	network.GetMP(uint32(conid))
}

func init() {
	newUi("txload tx", true, load_tx, "Load transaction data from the given file, decode it and store in memory")
	newUi("txsend stx", true, send_tx, "Broadcast transaction from memory pool (identified by a given <txid>)")
	newUi("tx1send stx1", true, send1_tx, "Broadcast transaction to a single random peer (identified by a given <txid>)")
	newUi("txsendall stxa", true, send_all_tx, "Broadcast all the transactions (what you see after ltx)")
	newUi("txdel dtx", true, del_tx, "Remove a transaction from memory pool (identified by a given <txid>)")
	newUi("txdecode td", true, dec_tx, "Decode a transaction from memory pool (identified by a given <txid>)")
	newUi("txlist ltx", true, list_txs, "List all the transaction loaded into memory pool up to 1MB space <max_size>")
	newUi("txlistban ltxb", true, baned_txs, "List the transaction that we have rejected")
	newUi("mempool mp", true, mempool_stats, "Show the mempool statistics")
	newUi("txsave", true, save_tx, "Save raw transaction from memory pool to disk")
	newUi("txmpsave mps", true, save_mempool, "Save memory pool to disk")
	newUi("txcheck txc", true, check_txs, "Verify consistency of mempool")
	newUi("txmpload mpl", true, load_mempool, "Load transaction from the given file (must be in mempool.dmp format)")
	newUi("getmp mpg", true, get_mempool, "Get getmp message to the peer with teh given ID")
}
