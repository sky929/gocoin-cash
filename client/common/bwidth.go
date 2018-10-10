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

// File:		bwidth.go
// Description:	Bictoin Cash common Package

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

package common

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	bw_mutex sync.Mutex

	dl_last_sec     int64 = time.Now().Unix()
	dl_bytes_so_far int

	DlBytesPrevSec    [0x10000]uint64 // this buffer takes 524288 bytes (hope it's not a problem)
	DlBytesPrevSecIdx uint16

	dl_bytes_priod uint64
	DlBytesTotal   uint64

	upload_limit   uint64
	download_limit uint64

	ul_last_sec     int64 = time.Now().Unix()
	ul_bytes_so_far int

	UlBytesPrevSec    [0x10000]uint64 // this buffer takes 524288 bytes (hope it's not a problem)
	UlBytesPrevSecIdx uint16
	ul_bytes_priod    uint64
	UlBytesTotal      uint64
)

func TickRecv() (ms int) {
	tn := time.Now()
	ms = tn.Nanosecond() / 1e6
	now := tn.Unix()
	if now != dl_last_sec {
		for now-dl_last_sec != 1 {
			DlBytesPrevSec[DlBytesPrevSecIdx] = 0
			DlBytesPrevSecIdx++
			dl_last_sec++
		}
		DlBytesPrevSec[DlBytesPrevSecIdx] = dl_bytes_priod
		DlBytesPrevSecIdx++
		dl_bytes_priod = 0
		dl_bytes_so_far = 0
		dl_last_sec = now
	}
	return
}

func TickSent() (ms int) {
	tn := time.Now()
	ms = tn.Nanosecond() / 1e6
	now := tn.Unix()
	if now != ul_last_sec {
		for now-ul_last_sec != 1 {
			UlBytesPrevSec[UlBytesPrevSecIdx] = 0
			UlBytesPrevSecIdx++
			ul_last_sec++
		}
		UlBytesPrevSec[UlBytesPrevSecIdx] = ul_bytes_priod
		UlBytesPrevSecIdx++
		ul_bytes_priod = 0
		ul_bytes_so_far = 0
		ul_last_sec = now
	}
	return
}

// Reads the given number of bytes, but respecting the download limit
// Returns -1 and no error if we can't read any data now, because of bw limit
func SockRead(con net.Conn, buf []byte) (n int, e error) {
	var toread int
	bw_mutex.Lock()
	ms := TickRecv()
	if DownloadLimit() == 0 {
		toread = len(buf)
	} else {
		toread = ms*int(DownloadLimit())/1000 - dl_bytes_so_far
		if toread > len(buf) {
			toread = len(buf)
			if toread > 4096 {
				toread = 4096
			}
		} else if toread < 0 {
			toread = 0
		}
	}
	dl_bytes_so_far += toread
	bw_mutex.Unlock()

	if toread > 0 {
		// Wait 10 millisecond for a data, timeout if nothing there
		con.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
		n, e = con.Read(buf[:toread])
		bw_mutex.Lock()
		dl_bytes_so_far -= toread
		if n > 0 {
			dl_bytes_so_far += n
			DlBytesTotal += uint64(n)
			dl_bytes_priod += uint64(n)
		}
		bw_mutex.Unlock()
	} else {
		n = -1
	}
	return
}

// Send all the bytes, but respect the upload limit (force delays)
// Returns -1 and no error if we can't send any data now, because of bw limit
func SockWrite(con net.Conn, buf []byte) (n int, e error) {
	var tosend int
	bw_mutex.Lock()
	ms := TickSent()
	if UploadLimit() == 0 {
		tosend = len(buf)
	} else {
		tosend = ms*int(UploadLimit())/1000 - ul_bytes_so_far
		if tosend > len(buf) {
			tosend = len(buf)
			if tosend > 4096 {
				tosend = 4096
			}
		} else if tosend < 0 {
			tosend = 0
		}
	}
	ul_bytes_so_far += tosend
	bw_mutex.Unlock()
	if tosend > 0 {
		// We used to have SetWriteDeadline() here, but it was causing problems because
		// in case of a timeout returned "n" was always 0, even if some data got sent.
		n, e = con.Write(buf[:tosend])
		bw_mutex.Lock()
		ul_bytes_so_far -= tosend
		if n > 0 {
			ul_bytes_so_far += n
			UlBytesTotal += uint64(n)
			ul_bytes_priod += uint64(n)
		}
		bw_mutex.Unlock()
	} else {
		n = -1
	}
	return
}

func LockBw() {
	bw_mutex.Lock()
}

func UnlockBw() {
	bw_mutex.Unlock()
}

func GetAvgBW(arr []uint64, idx uint16, cnt int) uint64 {
	var sum uint64
	if cnt <= 0 {
		return 0
	}
	for i := 0; i < cnt; i++ {
		idx--
		sum += arr[idx]
	}
	return sum / uint64(cnt)
}

func PrintBWStats() {
	bw_mutex.Lock()
	TickRecv()
	TickSent()
	fmt.Printf("Downloading at %d/%d KB/s, %s total",
		GetAvgBW(DlBytesPrevSec[:], DlBytesPrevSecIdx, 5)>>10, DownloadLimit()>>10, BytesToString(DlBytesTotal))
	fmt.Printf("  |  Uploading at %d/%d KB/s, %s total\n",
		GetAvgBW(UlBytesPrevSec[:], UlBytesPrevSecIdx, 5)>>10, UploadLimit()>>10, BytesToString(UlBytesTotal))
	bw_mutex.Unlock()
	return
}

func SetDownloadLimit(val uint64) {
	atomic.StoreUint64(&download_limit, val)
}

func DownloadLimit() uint64 {
	return atomic.LoadUint64(&download_limit)
}

func SetUploadLimit(val uint64) {
	atomic.StoreUint64(&upload_limit, val)
}

func UploadLimit() (res uint64) {
	return atomic.LoadUint64(&upload_limit)
}
