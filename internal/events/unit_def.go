package events

// UnitDef is a unit def the events refer to, with the fields of the engine's UnitDefs the analysis needs.
type UnitDef struct {
	// Name is the unit def name, e.g. "corcom".
	Name string `json:"name"`

	// TranslatedHumanName is the name the UI shows, e.g. "Cortex Commander".
	TranslatedHumanName string `json:"translatedHumanName"`

	// MetalCost is the metal it costs to build.
	MetalCost float64 `json:"metalCost"`

	// EnergyCost is the energy it costs to build.
	EnergyCost float64 `json:"energyCost"`

	// BuildTime is the build time in build power units.
	BuildTime float64 `json:"buildTime"`

	// IsFactory is true for factories.
	IsFactory bool `json:"isFactory"`

	// IsBuilder is true for units that build: commanders, constructors, nano turrets and factories.
	IsBuilder bool `json:"isBuilder"`

	// CanMove is true for mobile units. BAR sets it for factories too, so they can give their units move orders.
	CanMove bool `json:"canMove"`

	// BuildDistance is how far a builder builds from, in elmos.
	BuildDistance float64 `json:"buildDistance"`

	// CanResurrect is true for units that can resurrect wrecks.
	CanResurrect bool `json:"canResurrect"`

	// ExtractsMetal is the metal extraction factor of a metal extractor, 0 for other units.
	ExtractsMetal float64 `json:"extractsMetal"`

	// EnergyMake is the energy the unit makes per second.
	EnergyMake float64 `json:"energyMake"`

	// WindGenerator is the maximum energy of a wind generator, 0 for other units.
	WindGenerator float64 `json:"windGenerator"`

	// MaxWeaponRange is the range of the unit's longest-range weapon in elmos, 0 without weapons.
	MaxWeaponRange float64 `json:"maxWeaponRange"`

	// CustomParams provides BAR's customParams the analysis needs, as strings as Lua sees them: "techlevel",
	// "unitgroup" (e.g. "builder" or "aa") and "iscommander". A param the unit def does not set is missing.
	CustomParams map[string]string `json:"customParams"`
}
