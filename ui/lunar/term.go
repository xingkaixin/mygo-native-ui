package lunar

import (
	"math"
	"time"
)

// terms are the 24 solar terms in the order of the year, two in each
// month: the first starts a month of the 干支, the second, the 中气, falls
// in the month of the Chinese calendar of its number.
var terms = [24]string{
	"小寒", "大寒", "立春", "雨水", "惊蛰", "春分", "清明", "谷雨", "立夏", "小满", "芒种", "夏至",
	"小暑", "大暑", "立秋", "处暑", "白露", "秋分", "寒露", "霜降", "立冬", "小雪", "大雪", "冬至",
}

// Term returns the solar term on t's day, in t's location, as 清明, or ""
// on a day without one. Its day is the one of the moment the Sun reaches
// the term's longitude, in China's time, UTC+8.
func Term(t time.Time) string {
	y, m, d := t.Date()
	for i := range 2 {
		if termDay(y, m, i) == d {
			return terms[2*(int(m)-1)+i]
		}
	}
	return ""
}

// termDay returns the day of month m of year y of its first solar term,
// with i 0, or its second, with i 1.
func termDay(y int, m time.Month, i int) int {
	k := 2*(int(m)-1) + i
	// The Sun is at 285° at 小寒, about January 6, and moves 15° a term,
	// about 15.2 days.
	jde := 2440587.5 + float64(days(time.Date(y, 1, 6, 0, 0, 0, 0, time.UTC))) + float64(k)*15.2184
	deg := math.Mod(285+15*float64(k), 360)
	for range 5 {
		jde += math.Remainder(deg-sun(jde), 360) * 365.2422 / 360
	}
	china := jde - deltaT(float64(y)+float64(k)/24)/86400 + 8.0/24
	return time.Unix(int64(math.Floor(china-2440587.5))*86400, 0).UTC().Day()
}

// earth is the series of VSOP87 for the Earth's heliocentric longitude,
// L0 to L2, as Meeus gives them in Astronomical Algorithms, without the
// terms too small to move a solar term by a second between 1900 and 2100:
// amplitude in 1e-8 radians, phase, and frequency in radians a
// millennium.
var earth = [][][3]float64{{
	{175347046, 0, 0},
	{3341656, 4.6692568, 6283.07585},
	{34894, 4.6261, 12566.1517},
	{3497, 2.7441, 5753.3849},
	{3418, 2.8289, 3.5231},
	{3136, 3.6277, 77713.7715},
	{2676, 4.4181, 7860.4194},
	{2343, 6.1352, 3930.2097},
	{1324, 0.7425, 11506.7698},
	{1273, 2.0371, 529.691},
	{1199, 1.1096, 1577.3435},
	{990, 5.233, 5884.927},
	{902, 2.045, 26.298},
	{857, 3.508, 398.149},
	{780, 1.179, 5223.694},
	{753, 2.533, 5507.553},
	{505, 4.583, 18849.228},
	{492, 4.205, 775.523},
	{357, 2.92, 0.067},
	{317, 5.849, 11790.629},
	{284, 1.899, 796.298},
	{271, 0.315, 10977.079},
	{243, 0.345, 5486.778},
	{206, 4.806, 2544.314},
	{205, 1.869, 5573.143},
	{202, 2.458, 6069.777},
	{156, 0.833, 213.299},
	{132, 3.411, 2942.463},
	{126, 1.083, 20.775},
	{115, 0.645, 0.98},
	{103, 0.636, 4694.003},
	{102, 0.976, 15720.839},
	{102, 4.267, 7.114},
}, {
	{628331966747, 0, 0},
	{206059, 2.678235, 6283.07585},
	{4303, 2.6351, 12566.1517},
	{425, 1.59, 3.523},
}, {
	{52919, 0, 0},
	{8720, 1.0721, 6283.0758},
	{309, 0.867, 12566.152},
}}

// sun returns the Sun's apparent longitude, in degrees give or take whole
// turns, at the Julian ephemeris day jde.
func sun(jde float64) float64 {
	tau := (jde - 2451545) / 365250
	l, power := 0.0, 1.0
	for _, series := range earth {
		s := 0.0
		for _, term := range series {
			s += term[0] * math.Cos(term[1]+term[2]*tau)
		}
		l += s * power
		power *= tau
	}
	deg := l/1e8*180/math.Pi + 180
	t := tau * 10
	rad := func(d float64) float64 { return d * math.Pi / 180 }
	node, sunMean, moonMean := rad(125.04452-1934.136261*t), rad(280.4665+36000.7698*t), rad(218.3165+481267.8813*t)
	nutation := -17.2*math.Sin(node) - 1.32*math.Sin(2*sunMean) - 0.23*math.Sin(2*moonMean) + 0.21*math.Sin(2*node)
	// In arcseconds: VSOP87's equinox to FK5's, nutation, and aberration.
	return deg + (-0.09033+nutation-20.4898)/3600
}

// deltaT returns TT − UT, in seconds, in the year y: measured to 2020 and
// a guess after, as it cannot be known ahead.
func deltaT(y float64) float64 {
	table := []float64{-2.8, 21.2, 24.3, 33.2, 50.5, 63.8, 69.4, 74, 82, 93, 106}
	i := min(max(int((y-1900)/20), 0), len(table)-2)
	f := min(max((y-1900)/20-float64(i), 0), 1)
	return table[i] + (table[i+1]-table[i])*f
}
