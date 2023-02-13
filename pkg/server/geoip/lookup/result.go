package lookup

type Result struct {
	CountryCode                  string
	CountryName                  string
	ContinentCode                string
	CityName                     string
	Latitude                     float64
	Longitude                    float64
	AutonomousSystemNumber       uint32
	AutonomousSystemOrganization string
}
