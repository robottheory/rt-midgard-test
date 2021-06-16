// Sometimes ThorNode state is updated but the events doesn't reflect that perfectly.
//
// In these cases we open a bug report so future events are correct, but the old events will
// stay the same, and we apply these corrections to the existing events.
package record

func CorrectWithdaws(withdraw *Unstake, meta *Metadata) {
	f, ok := WithdrawCorrections[meta.BlockHeight]
	if ok {
		f(withdraw, meta)
	}
}

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

type (
	WithdrawCorrection    func(withdraw *Unstake, meta *Metadata)
	WithdrawCorrectionMap map[int64]WithdrawCorrection
)

var WithdrawCorrections WithdrawCorrectionMap

func LoadCorrections(chainID string) {
	if chainID == "" {
		return
	}
	AdditionalEvents = AddEventsFuncMap{}
	WithdrawCorrections = WithdrawCorrectionMap{}

	loadMainnet202104Corrections(chainID)
	loadTestnet202104Corrections(chainID)
}

// Note: we have copypasted Add functionsn because golang doesn't has templates yet.
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
