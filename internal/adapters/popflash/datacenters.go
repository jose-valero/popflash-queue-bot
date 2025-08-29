package popflash

import "fmt"

// Enum-like map: datacenter id → display name ("from 2015 popflash api structure :D")
var dcNames = map[int]string{
	1:  "NYC, NY",
	3:  "Düsseldorf, Germany",
	4:  "Dallas, TX",
	5:  "Stockholm, Sweden",
	6:  "London, UK",
	7:  "Los Angeles, CA",
	8:  "Chicago, IL",
	9:  "Strasbourg, France",
	10: "Amsterdam, NL",
	11: "Sydney, Australia",
	12: "Warsaw, Poland",
	13: "Madrid, Spain",
	14: "İstanbul, Turkey",
	15: "Seattle, WA",
	16: "Singapore",
	18: "Paris, France",
	19: "Helsinki, Finland",
	20: "Toronto, Canada",
	21: "Miami, FL",
	22: "Tokyo, Japan",
	23: "Johannesburg, South Africa",
	24: "Seoul, South Korea",
	25: "São Paulo, Brazil",
	26: "Copenhagen, Denmark",
	27: "Mumbai, India",
	28: "Denver, CO",
	29: "Atlanta, GA",
	31: "Milan, Italy",
	32: "Prague, Czechia",
	33: "Bucharest, Romania",
	34: "Dublin, Ireland",
	35: "Oslo, Norway",
	36: "Auckland, New Zealand",
	37: "Hong Kong",
	38: "Santiago, Chile",
}

// dcName returns the pretty location, falling back to "DC <id>"
func dcName(dc *int) string {
	if dc == nil {
		return "—"
	}
	if name, ok := dcNames[*dc]; ok {
		return name
	}
	return fmt.Sprintf("DC %d", *dc)
}
