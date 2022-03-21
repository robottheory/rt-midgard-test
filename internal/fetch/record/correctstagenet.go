package record

import "github.com/rs/zerolog/log"

const ChainIDStagenet202202 = "D8140E24344F73819452F5D01C4DA8D7DDEF71376CD84FA537250CFD9E1D6CC5"

func loadStagenet202202Corrections(chainID string) {
	if chainID == ChainIDStagenet202202 {
		log.Info().Msgf(
			"Loading corrections for stagenet started on 2021-02 id: %s",
			chainID)

		loadStagenetMissingNodeAccountStatus()
	}
}

func loadStagenetMissingNodeAccountStatus() {
	// There was a case where the first stagenet churn resulted in a node getting churned
	// in that didn't have the minimum bond, so it had a status of "Active" with a
	// preflight status "Standby" and the `UpdateNodeAccountStatus` event was never sent.
	AdditionalEvents.Add(1, func(d *Demux, meta *Metadata) {
		d.reuse.UpdateNodeAccountStatus = UpdateNodeAccountStatus{
			NodeAddr: []byte("sthor1vzenszq5gh0rsnft55kwfgk3vzfme4pks8r0se"),
			Former:   empty,
			Current:  []byte("Active"),
		}
		Recorder.OnUpdateNodeAccountStatus(&d.reuse.UpdateNodeAccountStatus, meta)
	})
}
