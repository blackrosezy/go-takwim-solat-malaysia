package solat

type PrayerTime struct {
	Hijri   string `json:"hijri"`
	Date    string `json:"date"`
	Day     string `json:"day"`
	Imsak   string `json:"imsak"`
	Subuh   string `json:"fajr"`
	Syuruk  string `json:"syuruk"`
	Zuhur   string `json:"dhuhr"`
	Asar    string `json:"asr"`
	Maghrib string `json:"maghrib"`
	Isyak   string `json:"isha"`
}

type TakwimSolat struct {
	PrayerTimes []PrayerTime `json:"prayerTime"`
}
