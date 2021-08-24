// Sometimes ThorNode state is updated but the events doesn't reflect that perfectly.
//
// In these cases we open a bug report so future events are correct, but the old events will
// stay the same, and we apply these corrections to the existing events.
package record

import (
	"strconv"
	"strings"
)

func LoadCorrections(chainID string) {
	if chainID == "" {
		return
	}
	AdditionalEvents = AddEventsFuncMap{}
	WithdrawCorrections = WithdrawCorrectionMap{}
	FeeAcceptFuncs = FeeAcceptMap{}

	loadMainnet202104Corrections(chainID)
	loadTestnet202107Corrections(chainID)
}

/////////////// Corrections for Missing Events

func AddMissingEvents(d *Demux, meta *Metadata) {
	f, ok := AdditionalEvents[meta.BlockHeight]
	if ok {
		f(d, meta)
	}
}

type (
	AddEventsFunc    func(d *Demux, meta *Metadata)
	AddEventsFuncMap map[int64]AddEventsFunc
)

var AdditionalEvents AddEventsFuncMap

/////////////// Corrections for Withdraws

func CorrectWithdaws(withdraw *Unstake, meta *Metadata) {
	f, ok := WithdrawCorrections[meta.BlockHeight]
	if ok {
		f(withdraw, meta)
	}
}

type (
	WithdrawCorrection    func(withdraw *Unstake, meta *Metadata)
	WithdrawCorrectionMap map[int64]WithdrawCorrection
)

var WithdrawCorrections WithdrawCorrectionMap

/////////////// Blacklist of fee events

func CorrectionsFeeEventIsOK(fee *Fee, meta *Metadata) bool {
	f, ok := FeeAcceptFuncs[meta.BlockHeight]
	if !ok {
		return true
	}
	return f(fee, meta)
}

type (
	FeeAcceptFunc func(fee *Fee, meta *Metadata) bool
	FeeAcceptMap  map[int64]FeeAcceptFunc
)

var FeeAcceptFuncs FeeAcceptMap

func (m FeeAcceptMap) Add(height int64, f FeeAcceptFunc) {
	fOrig, alreadyExists := m[height]
	if alreadyExists {
		m[height] = func(fee *Fee, meta *Metadata) bool {
			accepted := fOrig(fee, meta)
			if !accepted {
				return false
			}
			return f(fee, meta)
		}
	} else {
		m[height] = f
	}
}

/////////////// Artificial deposits to fix member pool units.

type artificialUnitChange struct {
	Pool  string
	Addr  string
	Units int64
}

type artificialUnitChanges map[int64][]artificialUnitChange

func registerArtificialDeposits(unitChanges artificialUnitChanges) {
	addAddEvent := func(d *Demux, meta *Metadata) {
		changes, ok := unitChanges[meta.BlockHeight]
		if ok {
			for _, change := range changes {
				if 0 <= change.Units {
					d.reuse.Stake = Stake{
						AddBase: AddBase{
							Pool: []byte(change.Pool),
						},
						StakeUnits: change.Units,
					}
					if AddressIsRune(change.Addr) {
						d.reuse.Stake.RuneAddr = []byte(change.Addr)
					} else {
						d.reuse.Stake.AssetAddr = []byte(change.Addr)
					}
					Recorder.OnStake(&d.reuse.Stake, meta)
				} else {
					d.reuse.Unstake = Unstake{
						Pool:       []byte(change.Pool),
						Asset:      []byte(change.Pool),
						FromAddr:   []byte(change.Addr),
						ToAddr:     []byte(change.Addr),
						StakeUnits: -change.Units,
						Tx:         []byte(change.Addr + strconv.Itoa(int(-change.Units))),
						Chain:      []byte(strings.Split(change.Pool, ".")[0]),
						Memo:       []byte("Midgard Fix"),
					}
					Recorder.OnUnstake(&d.reuse.Unstake, meta)
				}
			}
		}
	}
	for height := range unitChanges {
		AdditionalEvents.Add(height, addAddEvent)
	}
}

/////////////// Artificial pool balance changes to fix ThorNode/Midgard depth divergences.

type artificialPoolBallanceChange struct {
	Pool  string
	Rune  int64
	Asset int64
}

func absAndSign(x int64) (abs int64, pos bool) {
	if 0 <= x {
		return x, true
	} else {
		return -x, false
	}
}

func (x artificialPoolBallanceChange) toEvent() PoolBalanceChange {
	ret := PoolBalanceChange{
		Asset: []byte(x.Pool),
	}
	ret.RuneAmt, ret.RuneAdd = absAndSign(x.Rune)
	ret.AssetAmt, ret.AssetAdd = absAndSign(x.Asset)
	ret.Reason = "Fix in Midgard"
	return ret
}

type artificialPoolBallanceChanges map[int64][]artificialPoolBallanceChange

func registerArtificialPoolBallanceChanges(changes artificialPoolBallanceChanges, reason string) {
	addPoolBallanceChangeEvent := func(d *Demux, meta *Metadata) {
		changesAtHeight, ok := changes[meta.BlockHeight]
		if ok {
			for _, change := range changesAtHeight {
				d.reuse.PoolBalanceChange = change.toEvent()
				d.reuse.PoolBalanceChange.Reason = reason
				Recorder.OnPoolBalanceChange(&d.reuse.PoolBalanceChange, meta)
			}
		}
	}
	for height := range changes {
		AdditionalEvents.Add(height, addPoolBallanceChangeEvent)
	}
}

///////////////////////// MANUAL GENERICS
// We have copypasted Add functions because golang doesn't have templates yet.
// Rewrite this when generics arive (ETA end of 2021)

func (m AddEventsFuncMap) Add(height int64, f AddEventsFunc) {
	fOrig, alreadyExists := m[height]
	if alreadyExists {
		m[height] = func(d *Demux, meta *Metadata) {
			fOrig(d, meta)
			f(d, meta)
		}
		return
	}
	m[height] = f
}

func (m WithdrawCorrectionMap) Add(height int64, f WithdrawCorrection) {
	fOrig, alreadyExists := m[height]
	if alreadyExists {
		m[height] = func(withdraw *Unstake, meta *Metadata) {
			fOrig(withdraw, meta)
			f(withdraw, meta)
		}
		return
	}
	m[height] = f
}
