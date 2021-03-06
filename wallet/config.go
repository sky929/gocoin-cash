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

// File:        config.go
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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

var (
	keycnt       uint = 250
	testnet      bool = false
	waltype      uint = 3
	type2sec     string
	uncompressed bool   = false
	fee          string = "0.001"
	apply2bal    bool   = true
	secret_seed  []byte
	litecoin     bool = false
	txfilename   string
	stdin        bool
)

func parse_config() {
	cfgfn := os.Getenv("GOCOIN_WALLET_CONFIG")
	if cfgfn == "" {
		cfgfn = "wallet.cfg"
		fmt.Println("GOCOIN_WALLET_CONFIG not set")
	}
	d, e := ioutil.ReadFile(cfgfn)
	if e != nil {
		fmt.Println(cfgfn, "not found")
	} else {
		fmt.Println("Using config file", cfgfn)
		lines := strings.Split(string(d), "\n")
		for i := range lines {
			line := strings.Trim(lines[i], " \n\r\t")
			if len(line) == 0 || line[0] == '#' {
				continue
			}

			ll := strings.SplitN(line, "=", 2)
			if len(ll) != 2 {
				println(i, "wallet.cfg: syntax error in line", ll)
				continue
			}

			switch strings.ToLower(ll[0]) {
			case "testnet":
				v, e := strconv.ParseBool(ll[1])
				if e == nil {
					testnet = v
				} else {
					println(i, "wallet.cfg: value error for", ll[0], ":", e.Error())
					os.Exit(1)
				}

			case "type":
				v, e := strconv.ParseUint(ll[1], 10, 32)
				if e == nil {
					if v >= 1 && v <= 4 {
						waltype = uint(v)
					} else {
						println(i, "wallet.cfg: incorrect wallet type", v)
						os.Exit(1)
					}
				} else {
					println(i, "wallet.cfg: value error for", ll[0], ":", e.Error())
					os.Exit(1)
				}

			case "type2sec":
				type2sec = ll[1]

			case "keycnt":
				v, e := strconv.ParseUint(ll[1], 10, 32)
				if e == nil {
					if v >= 1 {
						keycnt = uint(v)
					} else {
						println(i, "wallet.cfg: incorrect key count", v)
						os.Exit(1)
					}
				} else {
					println(i, "wallet.cfg: value error for", ll[0], ":", e.Error())
					os.Exit(1)
				}

			case "uncompressed":
				v, e := strconv.ParseBool(ll[1])
				if e == nil {
					uncompressed = v
				} else {
					println(i, "wallet.cfg: value error for", ll[0], ":", e.Error())
					os.Exit(1)
				}

			// case "secrand": <-- deprecated

			case "fee":
				fee = ll[1]

			case "apply2bal":
				v, e := strconv.ParseBool(ll[1])
				if e == nil {
					apply2bal = v
				} else {
					println(i, "wallet.cfg: value error for", ll[0], ":", e.Error())
					os.Exit(1)
				}

			case "secret":
				PassSeedFilename = ll[1]

			case "others":
				RawKeysFilename = ll[1]

			case "seed":
				if !*nosseed {
					secret_seed = []byte(strings.Trim(ll[1], " \t\n\r"))
				}

			case "litecoin":
				v, e := strconv.ParseBool(ll[1])
				if e == nil {
					litecoin = v
				} else {
					println(i, "wallet.cfg: value error for", ll[0], ":", e.Error())
					os.Exit(1)
				}

			}
		}
	}

	flag.UintVar(&keycnt, "n", keycnt, "Set the number of determinstic keys to be calculated by the wallet")
	flag.BoolVar(&testnet, "t", testnet, "Testnet mode")
	flag.UintVar(&waltype, "type", waltype, "Type of a deterministic wallet to be used (1 to 4)")
	flag.StringVar(&type2sec, "t2sec", type2sec, "Enforce using this secret for Type-2 wallet (hex encoded)")
	flag.BoolVar(&uncompressed, "u", uncompressed, "Deprecated in this version")
	flag.StringVar(&fee, "fee", fee, "Specify transaction fee to be used")
	flag.BoolVar(&apply2bal, "a", apply2bal, "Apply changes to the balance folder (does not work with -raw)")
	flag.BoolVar(&litecoin, "ltc", litecoin, "Litecoin mode")
	flag.StringVar(&txfilename, "txfn", "", "Use this filename for output transaction (otherwise use a random name)")
	flag.BoolVar(&stdin, "stdin", stdin, "Read password from stdin")
	if uncompressed {
		fmt.Println("WARNING: Using uncompressed keys")
	}
}
