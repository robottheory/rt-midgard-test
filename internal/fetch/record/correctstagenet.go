package record

import "github.com/rs/zerolog/log"

const ChainIDStagenet202201 = "E95731338D751A2F511BBD0DDD9D34C5464B01F707C14E4B280A41AAA452F2D9"

func loadStagenet202201Corrections(chainID string) {
	if chainID == ChainIDStagenet202201 {
		log.Info().Msgf(
			"Loading corrections for stagenet started on 2021-01 id: %s",
			chainID)

		loadStagenetMissingNodeAccountStatus()
	}
}

func loadStagenetMissingNodeAccountStatus() {
	// There was a case where the first stagenet churn resulted in a node getting churned
	// in that didn't have the minimum bond, so it had a status of "Active" with a
	// preflight status "Standby" and the `UpdateNodeAccountStatus` event was never sent.
	AdditionalEvents.Add(88592, func(d *Demux, meta *Metadata) {
		d.reuse.UpdateNodeAccountStatus = UpdateNodeAccountStatus{
			NodeAddr: []byte("sthor1vzenszq5gh0rsnft55kwfgk3vzfme4pks8r0se"),
			Former:   empty,
			Current:  []byte("Active"),
		}
		Recorder.OnUpdateNodeAccountStatus(&d.reuse.UpdateNodeAccountStatus, meta)
	})
}
