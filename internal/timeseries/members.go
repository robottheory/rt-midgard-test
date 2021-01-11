package timeseries

import (
	"context"
	"fmt"

	"gitlab.com/thorchain/midgard/internal/db"
)

// Represents membership in a pool
type membership struct {
	runeAddress    string
	assetAddress   string
	liquidityUnits int64
}

type addrIndex map[string](map[string]*membership)

func (index addrIndex) getMembership(address, pool string) (*membership, bool) {
	_, ok := index[address]
	if ok {
		ret, ok := index[address][pool]
		return ret, ok
	} else {
		return nil, false
	}
}

func (index addrIndex) setMembership(address, pool string, newMembership *membership) {
	if index[address] == nil {
		index[address] = make(map[string]*membership)
	}
	index[address][pool] = newMembership
}

// MemberAddrs gets all member known addresses.
// When there's a rune/asset address pair or a rune addres for the member,
// the rune asset is shown.
// Else the asset address is shown.
// If an address participates in multiple pools it will be shown only once
func MemberAddrs(ctx context.Context) (addrs []string, err error) {
	// Build indexes: nested maps -> address and pools for each address as keys
	// Needed to access each member from any address and also to identify unique addresses

	// runeAddrIndex: all memberships with a rune address
	// using the rune address as key
	runeAddrIndex := make(addrIndex)

	// symAddrIndex: all memberships with an asset/rune address pair
	// using the asset address as key.
	// All the pointers here should also be in stored in runeAddrIndex
	symAssetAddrIndex := make(addrIndex)

	// asymAddrIndex: all memberships with only an asset address
	// none of the pointes here should be stored in runeAddrIndex
	// A single asset address can stake in different pools
	// (E.g.: ETH address in mutiple ERC20 tokens)
	asymAssetAddrIndex := make(addrIndex)

	// Symetrical addLiquidity actions are the only ones that
	// contain the address pair, if it's not here then it is not
	// a pair. Query this first and build rune and symAsset indexes
	const symALQ = `
		SELECT
			rune_addr,
			asset_addr,
			pool,
			SUM(stake_units) as liquidity_units
		FROM stake_events 
		WHERE asset_addr IS NOT NULL and rune_addr IS NOT NULL
		GROUP BY rune_addr, asset_addr, pool
	`
	symALRows, err := db.Query(ctx, symALQ)
	if err != nil {
		return nil, err
	}
	defer symALRows.Close()

	for symALRows.Next() {
		var newMembership membership
		var pool string
		err := symALRows.Scan(
			&newMembership.runeAddress,
			&newMembership.assetAddress,
			&pool,
			&newMembership.liquidityUnits)

		if err != nil {
			return nil, err
		}
		runeAddrIndex.setMembership(newMembership.runeAddress, pool, &newMembership)
		symAssetAddrIndex.setMembership(newMembership.assetAddress, pool, &newMembership)
	}

	// Asymmetrical addLiquidity with rune address only
	// may be part of a sym membership (found in symALQ so liquidiytUnits are is added to those)
	// or a new runeAddres only membership (Create it in the runeAddrIndex)
	const asymRuneALQ = `
		SELECT
			rune_addr,
			pool,
			SUM(stake_units) as liquidity_units
		FROM stake_events 
		WHERE asset_addr IS NULL and rune_addr IS NOT NULL
		GROUP BY rune_addr, pool
	`

	asymRuneALRows, err := db.Query(ctx, asymRuneALQ)
	if err != nil {
		return nil, err
	}
	defer asymRuneALRows.Close()
	for asymRuneALRows.Next() {
		var runeAddress, pool string
		var liquidityUnits int64
		err := asymRuneALRows.Scan(&runeAddress, &pool, &liquidityUnits)
		if err != nil {
			return nil, err
		}

		currentMembership, ok := runeAddrIndex.getMembership(runeAddress, pool)
		if ok {
			currentMembership.liquidityUnits += liquidityUnits
		} else {
			newMembership := membership{
				runeAddress:    runeAddress,
				liquidityUnits: liquidityUnits,
			}
			runeAddrIndex.setMembership(runeAddress, pool, &newMembership)
		}
	}

	// Asymmetrical addLiquidity with asset only
	// may be part of a sym membership (found in symALQ so liquidiytUnits are is added to those)
	// or a new assetAddress only membership (Create it in the asymAssetAddrIndex)
	const asymAssetALQ = `
		SELECT
			asset_addr,
			pool,
			SUM(stake_units) as liquidity_units
		FROM stake_events 
		WHERE asset_addr IS NOT NULL and rune_addr IS NULL
		GROUP BY asset_addr, pool
	`

	asymAssetALRows, err := db.Query(ctx, asymAssetALQ)
	if err != nil {
		return nil, err
	}
	defer asymAssetALRows.Close()
	for asymAssetALRows.Next() {
		var assetAddress, pool string
		var liquidityUnits int64
		err := asymAssetALRows.Scan(&assetAddress, &pool, &liquidityUnits)
		if err != nil {
			return nil, err
		}

		// If there's an existing member for liquidity change address + pool
		// in sym addr index add liquidity units there, else add in asym addr index
		symMember, isSym := symAssetAddrIndex.getMembership(assetAddress, pool)
		if isSym {
			symMember.liquidityUnits = +liquidityUnits
		} else {
			newMembership := membership{
				assetAddress:   assetAddress,
				liquidityUnits: liquidityUnits,
			}

			asymAssetAddrIndex.setMembership(assetAddress, pool, &newMembership)
		}
	}

	// Withdraws: try matching from address to a membreship from
	// the index and subtract addLiquidityUnits.
	// If there's no match either there's an error with the
	// implementation or the Thorchain events.
	const withdrawQ = `
		SELECT
			from_addr,
			pool,
			SUM(stake_units) as liquidity_units
		FROM unstake_events
		GROUP BY from_addr, pool
	`
	withdrawRows, err := db.Query(ctx, withdrawQ)
	if err != nil {
		return nil, err
	}
	defer withdrawRows.Close()

	for withdrawRows.Next() {
		var fromAddr, pool string
		var liquidityUnits int64
		err := withdrawRows.Scan(&fromAddr, &pool, &liquidityUnits)
		if err != nil {
			return nil, err
		}

		existingMembership, ok := runeAddrIndex.getMembership(fromAddr, pool)
		if ok && (existingMembership.runeAddress == fromAddr) {
			existingMembership.liquidityUnits -= liquidityUnits
			continue
		}

		existingMembership, ok = symAssetAddrIndex.getMembership(fromAddr, pool)
		if ok && (existingMembership.assetAddress == fromAddr) {
			existingMembership.liquidityUnits -= liquidityUnits
			continue
		}

		existingMembership, ok = asymAssetAddrIndex.getMembership(fromAddr, pool)
		if ok && (existingMembership.assetAddress == fromAddr) {
			existingMembership.liquidityUnits -= liquidityUnits
			continue
		}

		return nil, fmt.Errorf("Address %s, pool %s, found in withdraw events should have a matching membership", fromAddr, pool)
	}

	// Lookup membership addresses:
	// Either in runeIndex or asymIndex with at least one pool
	// with positive liquidityUnits balance
	addrs = make([]string, 0, len(runeAddrIndex)+len(asymAssetAddrIndex))

	for address, poolMemberships := range runeAddrIndex {
		// if it has at least a non zero balance, add it to the result
		isMember := false
		for _, memb := range poolMemberships {
			if memb.liquidityUnits > 0 {
				isMember = true
				break
			}
		}

		if isMember {
			addrs = append(addrs, address)
		}
	}

	for address, poolMemberships := range asymAssetAddrIndex {
		// if it has at least a non zero balance, add it to the result
		isMember := false
		for _, memb := range poolMemberships {
			if memb.liquidityUnits > 0 {
				isMember = true
				break
			}
		}

		if isMember {
			addrs = append(addrs, address)
		}
	}

	return addrs, nil
}
