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

// File:		segwit.go
// Description:	Bictoin Cash Cash Adress Package

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

package bech32

import (
	"bytes"
)

// Return nil on error
func convert_bits(outbits uint, in []byte, inbits uint, pad bool) []byte {
	var val uint32
	var bits uint
	maxv := uint32(1<<outbits) - 1
	out := new(bytes.Buffer)
	for inx := range in {
		val = (val << inbits) | uint32(in[inx])
		bits += inbits
		for bits >= outbits {
			bits -= outbits
			out.WriteByte(byte((val >> bits) & maxv))
		}
	}
	if pad {
		if bits != 0 {
			out.WriteByte(byte((val << (outbits - bits)) & maxv))
		}
	} else if ((val<<(outbits-bits))&maxv) != 0 || bits >= inbits {
		return nil
	}
	return out.Bytes()
}

// Returns empty string on error
func SegwitEncode(hrp string, witver int, witprog []byte) string {
	if witver > 16 {
		return ""
	}
	if witver == 0 && len(witprog) != 20 && len(witprog) != 32 {
		return ""
	}
	if len(witprog) < 2 || len(witprog) > 40 {
		return ""
	}
	return Encode(hrp, append([]byte{byte(witver)}, convert_bits(5, witprog, 8, true)...))
}

// returns (0, nil) on error
func SegwitDecode(hrp, addr string) (witver int, witdata []byte) {
	hrp_actual, data := Decode(addr)
	if hrp_actual == "" || len(data) == 0 || len(data) > 65 {
		return
	}
	if hrp != hrp_actual {
		return
	}
	if data[0] > 16 {
		return
	}
	witdata = convert_bits(8, data[1:], 5, false)
	if witdata == nil {
		return
	}
	if len(witdata) < 2 || len(witdata) > 40 {
		witdata = nil
		return
	}
	if data[0] == 0 && len(witdata) != 20 && len(witdata) != 32 {
		witdata = nil
		return
	}
	witver = int(data[0])
	return
}
