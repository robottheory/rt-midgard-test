package models

type PoolSlippage struct {
	TotalSlippage    uint64
	BuySlippage      uint64
	SellSlippage     uint64
	MeanPoolSlippage uint64
	MeanBuySlippage  uint64
	MeanSellSlippage uint64
}
