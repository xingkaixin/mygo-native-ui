package lunar

import (
	"math/bits"
	"time"
)

// Date is a day of the Chinese calendar: Year, the Gregorian year its
// 正月初一 falls in; Month, from 1 for 正月 to 12 for 腊月, and Leap for the
// leap month that follows the month of that number; and Day, from 1 to 30.
type Date struct {
	Year, Month, Day int
	Leap             bool
}

// years holds a year of the Chinese calendar from 1900 to 2100 in each
// number, as Chinese calendar programs have long shared it: bits 15 to 4
// are months 1 to 12, set for 30 days and clear for 29; bits 3 to 0 are
// the month a leap month follows, or 0 for none; and bit 16 is set when
// the leap month has 30 days.
var years = [...]uint32{
	0x04bd8, 0x04ae0, 0x0a570, 0x054d5, 0x0d260, 0x0d950, 0x16554, 0x056a0, 0x09ad0, 0x055d2, // 1900
	0x04ae0, 0x0a5b6, 0x0a4d0, 0x0d250, 0x1d255, 0x0b540, 0x0d6a0, 0x0ada2, 0x095b0, 0x14977, // 1910
	0x04970, 0x0a4b0, 0x0b4b5, 0x06a50, 0x06d40, 0x1ab54, 0x02b60, 0x09570, 0x052f2, 0x04970, // 1920
	0x06566, 0x0d4a0, 0x0ea50, 0x16a95, 0x05ad0, 0x02b60, 0x186e3, 0x092e0, 0x1c8d7, 0x0c950, // 1930
	0x0d4a0, 0x1d8a6, 0x0b550, 0x056a0, 0x1a5b4, 0x025d0, 0x092d0, 0x0d2b2, 0x0a950, 0x0b557, // 1940
	0x06ca0, 0x0b550, 0x15355, 0x04da0, 0x0a5b0, 0x14573, 0x052b0, 0x0a9a8, 0x0e950, 0x06aa0, // 1950
	0x0aea6, 0x0ab50, 0x04b60, 0x0aae4, 0x0a570, 0x05260, 0x0f263, 0x0d950, 0x05b57, 0x056a0, // 1960
	0x096d0, 0x04dd5, 0x04ad0, 0x0a4d0, 0x0d4d4, 0x0d250, 0x0d558, 0x0b540, 0x0b6a0, 0x195a6, // 1970
	0x095b0, 0x049b0, 0x0a974, 0x0a4b0, 0x0b27a, 0x06a50, 0x06d40, 0x0af46, 0x0ab60, 0x09570, // 1980
	0x04af5, 0x04970, 0x064b0, 0x074a3, 0x0ea50, 0x06b58, 0x05ac0, 0x0ab60, 0x096d5, 0x092e0, // 1990
	0x0c960, 0x0d954, 0x0d4a0, 0x0da50, 0x07552, 0x056a0, 0x0abb7, 0x025d0, 0x092d0, 0x0cab5, // 2000
	0x0a950, 0x0b4a0, 0x0baa4, 0x0ad50, 0x055d9, 0x04ba0, 0x0a5b0, 0x15176, 0x052b0, 0x0a930, // 2010
	0x07954, 0x06aa0, 0x0ad50, 0x05b52, 0x04b60, 0x0a6e6, 0x0a4e0, 0x0d260, 0x0ea65, 0x0d530, // 2020
	0x05aa0, 0x076a3, 0x096d0, 0x04afb, 0x04ad0, 0x0a4d0, 0x1d0b6, 0x0d250, 0x0d520, 0x0dd45, // 2030
	0x0b5a0, 0x056d0, 0x055b2, 0x049b0, 0x0a577, 0x0a4b0, 0x0aa50, 0x1b255, 0x06d20, 0x0ada0, // 2040
	0x14b63, 0x09370, 0x049f8, 0x04970, 0x064b0, 0x168a6, 0x0ea50, 0x06b20, 0x1a6c4, 0x0aae0, // 2050
	0x092e0, 0x0d2e3, 0x0c960, 0x0d557, 0x0d4a0, 0x0da50, 0x05d55, 0x056a0, 0x0a6d0, 0x055d4, // 2060
	0x052d0, 0x0a9b8, 0x0a950, 0x0b4a0, 0x0b6a6, 0x0ad50, 0x055a0, 0x0aba4, 0x0a5b0, 0x052b0, // 2070
	0x0b273, 0x06930, 0x07337, 0x06aa0, 0x0ad50, 0x14b55, 0x04b60, 0x0a570, 0x054e4, 0x0d160, // 2080
	0x0e968, 0x0d520, 0x0daa0, 0x16aa6, 0x056d0, 0x04ae0, 0x0a9d4, 0x0a2d0, 0x0d150, 0x0f252, // 2090
	0x0d520, // 2100
}

// epoch is 正月初一 of 1900, as days after 1970-01-01.
var epoch = days(time.Date(1900, 1, 31, 0, 0, 0, 0, time.UTC))

// Of returns the date of the Chinese calendar on t's day, in t's location,
// or false outside its years 1900 to 2100, which run from 1900-01-31 to
// 2101-01-28.
func Of(t time.Time) (Date, bool) {
	n := days(t) - epoch
	if n < 0 {
		return Date{}, false
	}
	y := 1900
	for ; y-1900 < len(years) && n >= length(y); y++ {
		n -= length(y)
	}
	if y-1900 == len(years) {
		return Date{}, false
	}
	for m, leap := 1, false; ; {
		k := monthLength(y, m, leap)
		if n < k {
			return Date{Year: y, Month: m, Day: n + 1, Leap: leap}, true
		}
		n -= k
		if !leap && m == int(years[y-1900]&0xf) {
			leap = true
		} else {
			m, leap = m+1, false
		}
	}
}

// String writes d as calendars do, the year by its 干支: 乙巳年闰六月初一.
func (d Date) String() string { return cycle(d.Year-4) + "年" + d.monthName() + d.dayName() }

// length returns how many days year y of the Chinese calendar has.
func length(y int) int {
	n := 12*29 + bits.OnesCount32(years[y-1900]&0xfff0)
	if years[y-1900]&0xf != 0 {
		n += monthLength(y, 0, true)
	}
	return n
}

// monthLength returns how many days month m of year y has, or its leap
// month with leap.
func monthLength(y, m int, leap bool) int {
	bit := uint32(0x10000) >> m
	if leap {
		bit = 0x10000
	}
	if years[y-1900]&bit != 0 {
		return 30
	}
	return 29
}

var (
	stems     = []rune("甲乙丙丁戊己庚辛壬癸")
	branches  = []rune("子丑寅卯辰巳午未申酉戌亥")
	animals   = []rune("鼠牛虎兔龙蛇马羊猴鸡狗猪")
	digits    = []rune("一二三四五六七八九十")
	months    = []rune("正二三四五六七八九十冬腊")
	decades   = []rune("初十廿")
	weekdays  = []rune("日一二三四五六")
	festivals = map[[2]int]string{{1, 1}: "春节", {1, 15}: "元宵", {5, 5}: "端午", {7, 7}: "七夕", {7, 15}: "中元", {8, 15}: "中秋", {9, 9}: "重阳", {12, 8}: "腊八", {12, 23}: "小年"}
)

// cycle returns the n-th 干支 of the sixty, from 甲子.
func cycle(n int) string {
	n = (n%60 + 60) % 60
	return string(stems[n%10]) + string(branches[n%12])
}

// yearName names year y of the Chinese calendar by its 干支 and animal:
// 乙巳年 · 蛇.
func yearName(y int) string {
	return cycle(y-4) + "年 · " + string(animals[((y-4)%12+12)%12])
}

func (d Date) monthName() string {
	name := string(months[d.Month-1]) + "月"
	if d.Leap {
		return "闰" + name
	}
	return name
}

func (d Date) dayName() string {
	switch d.Day {
	case 10:
		return "初十"
	case 20:
		return "二十"
	case 30:
		return "三十"
	}
	return string(decades[(d.Day-1)/10]) + string(digits[(d.Day-1)%10])
}

// festival returns the festival of the Chinese calendar on d, whose next
// day is next, or "".
func festival(d, next Date) string {
	if next.Month == 1 && next.Day == 1 && !next.Leap {
		return "除夕"
	}
	if d.Leap {
		return ""
	}
	return festivals[[2]int{d.Month, d.Day}]
}

// pillars returns the 干支 of t's year, month and day. The month's begins
// at the solar term that starts it, 立春 for 寅月, and so the year's at
// 立春.
func pillars(t time.Time) (year, month, day string) {
	y, m, d := t.Date()
	n := y*12 + int(m) - 1
	if d < termDay(y, m, 0) {
		n--
	}
	// 1900-01-01 was a 甲戌 day, the 11th of the sixty, and 13 lines the
	// months up so that 立春 of 1984, a 甲子 year, begins 丙寅月.
	return cycle((n-1)/12 - 4), cycle(n + 13), cycle(days(t) - days(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)) + 10)
}

// days returns how many days t's day, in its location, is after
// 1970-01-01.
func days(t time.Time) int {
	y, m, d := t.Date()
	return int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400)
}
