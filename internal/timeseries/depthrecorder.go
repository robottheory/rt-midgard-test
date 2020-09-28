// Depth recorder fills keeps track of historical depth values and inserts changes
// in the block_pool_depths table.
package timeseries

import (
	"fmt"
	"strings"
)

// MapDiff helps to get differences between snapshots of a map.
type mapDiff struct {
	snapshot map[string]int64
}

// Save the map as the new snapshot.
func (md *mapDiff) save(newMap map[string]int64) {
	md.snapshot = map[string]int64{}
	for k, v := range newMap {
		md.snapshot[k] = v
	}
}

// Check if there is a chage for this pool.
func (md *mapDiff) diffAtKey(pool string, newMap map[string]int64) (hasDiff bool, newValue int64) {
	oldV, hasOld := md.snapshot[pool]
	newV, hasNew := newMap[pool]
	if hasNew == true {
		return !hasOld || oldV != newV, newV
	} else {
		return hasOld, 0
	}
}

type depthManager struct {
	assetE8DepthSnapshot mapDiff
	runeE8DepthSnapshot  mapDiff
}

var depthRecorder depthManager

// Insert rows in the block_pool_depths for every changed value in the depth maps.
// If there is no change it doesn't write out anything.
// Both values will be writen out together (assetDepth, runeDepth), even if only one of the values
// changed in the pool.
// TODO(acsaba): we have pools with empty names. Figure out why.
func (sm *depthManager) update(
	height int64, assetE8DepthPerPool, runeE8DepthPerPool map[string]int64) error {

	// We need to iterate over all 4 maps: oldAssets, newAssets, oldRunes, newRunes.
	// First put all pool names into a set.
	poolNames := map[string]bool{}
	accumulatePoolNames := func(m map[string]int64) {
		for pool := range m {
			poolNames[pool] = true
		}
	}
	accumulatePoolNames(assetE8DepthPerPool)
	accumulatePoolNames(runeE8DepthPerPool)
	accumulatePoolNames(sm.assetE8DepthSnapshot.snapshot)
	accumulatePoolNames(sm.runeE8DepthSnapshot.snapshot)

	// TODO(acsaba): confirm that it's ok to insert multiple lines like this,
	//     and if there is a limit.
	queryFront := "INSERT INTO block_pool_depths (height, pool, asset_e8, rune_e8) VALUES "
	queryEnd := " ON CONFLICT DO NOTHING;"
	rowFormat := "($%d, $%d, $%d, $%d)"
	rowStrs := []string{}
	values := []interface{}{} // Finally there will be rowNum*4 parameters.

	for pool := range poolNames {
		assetDiff, assetValue := sm.assetE8DepthSnapshot.diffAtKey(pool, assetE8DepthPerPool)
		runeDiff, runeValue := sm.runeE8DepthSnapshot.diffAtKey(pool, runeE8DepthPerPool)
		if assetDiff || runeDiff {
			p := len(values)
			rowStrs = append(rowStrs, fmt.Sprintf(rowFormat, p+1, p+2, p+3, p+4))
			values = append(values, height, pool, assetValue, runeValue)
		}
	}
	sm.assetE8DepthSnapshot.save(assetE8DepthPerPool)
	sm.runeE8DepthSnapshot.save(runeE8DepthPerPool)

	diffNum := len(rowStrs)

	if 0 == diffNum {
		// There were no differences in depths.
		return nil
	}

	query := queryFront + strings.Join(rowStrs, ", ") + queryEnd
	result, err := DBExec(query, values...)
	if err != nil {
		return fmt.Errorf("Error saving depths %d: %w", height, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Error saving depths %d results: %w", height, err)
	}
	if n != int64(diffNum) {
		return fmt.Errorf(
			"Not all depths were saved at height %d (expected inserts: %d, actual: %d)",
			height, n, diffNum)
	}
	return nil
}
