package huntbot

import (
	"math"
	"math/rand"

	"github.com/sempatowo/sempatowo/internal/config"
)

type traitFormula struct {
	inc    float64
	pow    float64
	weight float64
	max    int
}

var traitFormulas = map[Trait]traitFormula{
	TraitEfficiency: {inc: 10, pow: 1.748, weight: 4, max: 215},
	TraitDuration:   {inc: 10, pow: 1.7, weight: 2, max: 235},
	TraitCost:       {inc: 1000, pow: 3.4, weight: 5, max: 5},
	TraitGain:       {inc: 10, pow: 1.8, weight: 4, max: 200},
	TraitExp:        {inc: 10, pow: 1.8, weight: 3, max: 200},
	TraitRadar:      {inc: 50, pow: 2.5, weight: 1, max: 999},
}

// AllocateEssence distributes available essence across enabled traits (greedy optimizer).
func AllocateEssence(input UpgradeDetails, weights config.HuntbotWeights) map[string]int {
	formulas := copyFormulas(weights)
	enabled := enabledTraits(input)
	allocation := make(map[string]int)
	levels := make(map[string]int)
	invested := make(map[string]int)

	for trait, state := range enabled {
		allocation[trait] = 0
		levels[trait] = state.CurrentLevel
		invested[trait] = state.Invested
	}

	remaining := input.Essence

	for remaining > 0 {
		var bestTrait string
		bestRatio := -1.0
		var costForBest int

		for trait := range allocation {
			f := formulas[Trait(trait)]
			level := levels[trait]
			if level >= f.max {
				continue
			}

			required := upgradeCost(f, level+1, invested[trait])
			if required == 0 {
				levels[trait]++
				invested[trait] = 0
				continue
			}

			ratio := f.weight / float64(required)
			if required <= remaining && ratio > bestRatio {
				bestRatio = ratio
				bestTrait = trait
				costForBest = required
			}
		}

		if bestTrait != "" {
			allocation[bestTrait] += costForBest
			remaining -= costForBest
			levels[bestTrait]++
			invested[bestTrait] = 0
			continue
		}

		// Partial allocation for the best remaining trait.
		bestTrait = ""
		bestRatio = -1
		for trait := range allocation {
			f := formulas[Trait(trait)]
			if levels[trait] >= f.max {
				continue
			}
			required := upgradeCost(f, levels[trait]+1, invested[trait])
			if required <= 0 {
				continue
			}
			ratio := f.weight / float64(required)
			if ratio > bestRatio {
				bestRatio = ratio
				bestTrait = trait
			}
		}

		if bestTrait != "" {
			allocation[bestTrait] += remaining
			invested[bestTrait] += remaining
			remaining = 0
		} else {
			break
		}
	}

	return allocation
}

func upgradeCost(f traitFormula, nextLevel, alreadyInvested int) int {
	full := int(math.Floor(f.inc * math.Pow(float64(nextLevel), f.pow)))
	required := full - alreadyInvested
	if required < 0 {
		return 0
	}
	return required
}

func copyFormulas(weights config.HuntbotWeights) map[Trait]traitFormula {
	out := make(map[Trait]traitFormula, len(traitFormulas))
	for t, f := range traitFormulas {
		f.weight = weightFor(t, weights)
		out[t] = f
	}
	return out
}

func weightFor(t Trait, w config.HuntbotWeights) float64 {
	switch t {
	case TraitEfficiency:
		return w.Efficiency
	case TraitDuration:
		return w.Duration
	case TraitCost:
		return w.Cost
	case TraitGain:
		return w.Gain
	case TraitExp:
		return w.Exp
	case TraitRadar:
		return w.Radar
	default:
		return 0
	}
}

func enabledTraits(input UpgradeDetails) map[string]TraitState {
	m := make(map[string]TraitState)
	add := func(name string, s TraitState) {
		if s.Enabled {
			m[name] = s
		}
	}
	add("efficiency", input.Efficiency)
	add("duration", input.Duration)
	add("cost", input.Cost)
	add("gain", input.Gain)
	add("exp", input.Exp)
	add("radar", input.Radar)
	return m
}

func upgraderCooldown(st config.HuntbotUpgrader) float64 {
	s := st.Sleeptime
	if s.Range != nil {
		min, max := s.Range[0], s.Range[1]
		return min + rand.Float64()*(max-min)
	}
	if s.Single != nil {
		return *s.Single
	}
	return 0
}