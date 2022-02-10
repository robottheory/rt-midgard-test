# Earnings explained

Calculating gains of a pool is not a trivial task.

## Partial pool ownership

Let's define the share of a member in the pool.

```
A  = total Asset in pool
R  = total Rune in pool
U0 = pool units of a specific member in the pool
U  = total sum of pool units in the pool
```

In this case the share of the member from the pool is `A * U0 / U` Asset and `R * U0 / U` Rune.
These are the amounts the member would get on a withdraw.

## Price

The definition of the price is that the value of Rune and Asset in the pool is the same.

`p = R / A`

That means that for `p` Rune can be swapped for `1` Asset.

## Asymmetrical pool ownership doesn't exist

Symmetric deposit means that the member provides both Rune and Asset to the pool, in such a ratio
that it doesn't move the price.

For convenience members can deposit or withdraw asymmetrically, which means that they use a different ratio
or just provide one side (Rune or Asset).

Note that after the deposit happened the pool ownership is reduced to the single number of pool units `U0`,
so effectively they have converted their deposit to half Asset and half Rune right away.


## Pooled vs HOLD

Through time the `U0` units of the member is constant, but the corresponding Rune and Asset values change because:

* Fees collected, increases Rune and Asset
* Arbitrage of price change changes the ratio of Rune and Asset

Let's take an example where a member deposits then later withdraws all symmetrically:

```
Original price = 10
deposit = 100 Asset 1000 Rune
deposit value in Rune = 2000
deposit value in Asset = 200

- step 1: arbitrage without fees to new price of 2.5: 200 Asset 500 Rune
- step 2: collect fees in some way (10%): 20 Asset 50 Rune

withdraw: 220 Asset 550 Rune
withdraw value in Rune = 1100
withdraw value in Asset = 440
```

Calculating gains depends of how one thinks of the baseline what the hold position would mean.
Options:

1) new value vs holding the original amount in Rune (2000->1100 = -45%)
2) new value vs holding the original amount in Asset (200 -> 440 = +120%)
3) new value vs holding the original half Rune and half Asset position at deposit
   (`(100*2.5 + 1000) -> 1100 = -12%)

This document will discuss option 3) as a baseline because:
* 1) and 2) is volatile, mostly a measure of price and not the fees collected.
  One might be typically negative the other positive.
* ThorNode considers pool ownership as half rune half asset (3)
* 3) is sensitive to fees collected and impermanent loss too. In the example above it was negative
  because of the big price change and relatively small fees.
* ThorNode uses (3) for calculating impermanent loss protection. That means that (3) is under `1`, then
  a corresponding payout is done.

So in this document we will chose gain as

```
GainRatio = (WithdrawAsset * newPrice + WithdrawRune) / (DepositAsset * newPrice + DepositRune)
```

Which simplifies to

```
GainRatio = 2 * WithdrawRune / (DepositAsset * newPrice + DepositRune)
```

# Gain on a time interval

If we know the deposit date and the withdraw date of a fictive member then we can calculate gains,
without knowing the actual amounts, just by following how much a single pool unit is worth.

Pool ownership of a single pool unit: `R/U` `A/U`

## GainRatio decomposition

We can decompose the GainRatio into two factors:
* LUVI represents fees collected
* PriceShiftLoss represents the impermanent loss from the price shift.

Definitions:
```
A0 = Asset depth of pool at deposit
R0 = Rune depth of pool at deposit
U0 = Pool units of pool at deposit
A1 = Asset depth of pool at withdraw
R1 = Rune depth of pool at withdraw
U1 = Pool units of pool at withdraw
```

## Liquidity Unit Value Index (LUVI)

ThorChain uses a the fixed product formula for the pool: `A*R=K`. Ideal swaps would keep the pool
on this curve, but fees, rewards and donations increase the K.

Definition:
```
LUVI = sqrt(A*R) / U

LUVI0 = sqrt(A0*R0) / U0
LUVI1 = sqrt(A1*R1) / U1

LUVI_Increase = LUVI1 / LUVI0
```

Note that if there is no price change then LUVI_Increase is exactly the GainRatio. It is
always greater than `1`.

## Price Shift Loss

If we assume perfect swaps keeping the `K` constant in `A*R=K` then a changed price will cause an
"impermanent loss".

```
Price0 = R0 / A0
Price1 = R1 / A1
PriceShift = Price1 / Price0
PriceShiftLoss = 2*sqrt(PriceShift) / (1 + PriceShift)
```

Note that PriceShiftLoss is smaller then `1` indifferently of the price going up or down.
Check the characteristics here:

https://www.desmos.com/calculator/u7oa6z6wum

Contrary to the LUVI PriceShiftLoss doesn't accumulate over multiple time intervals, e.g.
two daily impermanent losses of 0.95 in the may end up cancelling each other.
Impermanent loss makes more sense on longer time frames.

## GainRatio = LUVI * PriceShiftLoss

As previously mentioned, the definition of GainRatio is:

```
GainRatio = 2 * WithdrawRune / (DepositAsset * newPrice + DepositRune)
```

By doing some calculations one can verify that this is equivalent to:

```
GainRatio = LUVIIncrease * PriceShiftLoss
```

In the above example assuming pool units `U = 100`

```
GainRatio = 2 * WithdrawRune / (DepositAsset * newPrice + DepositRune)
          =  1100 / (100*2.5 + 1000)
          = 0.88

LUVI0 = [sqrt(100*1000) / 100] = 3.16227
LUVI1 = [sqrt(220*550) / 100] = 3.47850
LUVI_Increase = LUVI1 / LUVI0 = 1.1

Price0 = 10
Price1 = 2.5
PriceShift = 2.5 / 10 = 0.25
PriceShiftLoss = 2*sqrt(PriceShift) / (1 + PriceShift)
               = 0.8

GainRatio = LUVIIncrease * PriceShiftLoss
          = 1.1 * 0.8
          = 0.88
```

Because the big price shift this example has a 20% impermanent loss and a 10% fee collection,
which sums up to a 12% overall loss.
