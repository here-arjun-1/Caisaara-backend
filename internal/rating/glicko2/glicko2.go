package glicko2

import "math"

const (
	DefaultRD         = 350.0
	DefaultVolatility = 0.06
	ProvisionalRD     = 110.0

	scale   = 173.7178
	tau     = 0.5
	epsilon = 0.000001
)

type Player struct {
	Rating     float64
	RD         float64
	Volatility float64
}

type Result struct {
	Opponent Player
	Score    float64
}

func Update(p Player, results []Result) Player {
	mu := (p.Rating - 1500) / scale
	phi := p.RD / scale

	if len(results) == 0 {
		newPhi := math.Sqrt(phi*phi + p.Volatility*p.Volatility)
		return Player{
			Rating:     p.Rating,
			RD:         math.Min(newPhi*scale, DefaultRD),
			Volatility: p.Volatility,
		}
	}

	var vInverse, scoreSum float64
	for _, r := range results {
		opponentMu := (r.Opponent.Rating - 1500) / scale
		opponentPhi := r.Opponent.RD / scale

		gValue := g(opponentPhi)
		expected := expectedScore(mu, opponentMu, gValue)

		vInverse += gValue * gValue * expected * (1 - expected)
		scoreSum += gValue * (r.Score - expected)
	}

	v := 1 / vInverse
	delta := v * scoreSum

	newVolatility := calculateVolatility(phi, p.Volatility, v, delta)

	phiStar := math.Sqrt(phi*phi + newVolatility*newVolatility)
	newPhi := 1 / math.Sqrt(1/(phiStar*phiStar)+1/v)
	newMu := mu + newPhi*newPhi*scoreSum

	return Player{
		Rating:     newMu*scale + 1500,
		RD:         newPhi * scale,
		Volatility: newVolatility,
	}
}

func IsProvisional(rd float64) bool {
	return rd > ProvisionalRD
}

func g(phi float64) float64 {
	return 1 / math.Sqrt(1+3*phi*phi/(math.Pi*math.Pi))
}

func expectedScore(mu, opponentMu, gValue float64) float64 {
	return 1 / (1 + math.Exp(-gValue*(mu-opponentMu)))
}

func calculateVolatility(phi, sigma, v, delta float64) float64 {
	a := math.Log(sigma * sigma)

	f := func(x float64) float64 {
		ex := math.Exp(x)
		top := ex * (delta*delta - phi*phi - v - ex)
		bottom := 2 * (phi*phi + v + ex) * (phi*phi + v + ex)
		return top/bottom - (x-a)/(tau*tau)
	}

	A := a
	var B float64
	if delta*delta > phi*phi+v {
		B = math.Log(delta*delta - phi*phi - v)
	} else {
		k := 1.0
		for f(a-k*tau) < 0 {
			k++
		}
		B = a - k*tau
	}

	fA := f(A)
	fB := f(B)

	for math.Abs(B-A) > epsilon {
		C := A + (A-B)*fA/(fB-fA)
		fC := f(C)

		if fC*fB <= 0 {
			A = B
			fA = fB
		} else {
			fA = fA / 2
		}

		B = C
		fB = fC
	}

	return math.Exp(A / 2)
}
