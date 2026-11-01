package events

// UnitDef is written the first time a unit type shows up, before the event that uses it, so the file needs no unit def
// files to read. The fields are the engine's UnitDefs entries, except where noted.
type UnitDef struct {
	// Name is the unit def name the other events use, e.g. "corcom".
	Name string `json:"name"`

	// HumanName is the name the UI shows, e.g. "Cortex Commander".
	HumanName string `json:"humanName"`

	// MetalCost is the metal it costs to build.
	MetalCost float64 `json:"metalCost"`

	// EnergyCost is the energy it costs to build.
	EnergyCost float64 `json:"energyCost"`

	// BuildTime is the build time in build power units.
	BuildTime float64 `json:"buildTime"`

	// TechLevel is BAR's customParams.techlevel, 1 if not set.
	TechLevel int `json:"techLevel"`

	// UnitGroup is BAR's customParams.unitgroup, e.g. "builder" or "aa". Is empty if not set.
	UnitGroup string `json:"unitGroup,omitempty"`

	// IsFactory is true for factories.
	IsFactory bool `json:"isFactory,omitempty"`

	// IsBuilder is true for builders that are not factories: commanders, constructors and nano turrets.
	IsBuilder bool `json:"isBuilder,omitempty"`

	// IsCommander is BAR's customParams.iscommander.
	IsCommander bool `json:"isCommander,omitempty"`

	// IsMobile is true for units that can move, factories excluded.
	IsMobile bool `json:"isMobile,omitempty"`

	// BuildDistance is how far a builder that is not a factory builds from, in elmos. Is 0 for other units.
	BuildDistance float64 `json:"buildDistance,omitempty"`

	// CanResurrect is true for units that can resurrect wrecks.
	CanResurrect bool `json:"canResurrect,omitempty"`

	// CanFly is true for aircraft.
	CanFly bool `json:"canFly,omitempty"`

	// ExtractsMetal is the metal extraction factor of a metal extractor. Is 0 for other units.
	ExtractsMetal float64 `json:"extractsMetal,omitempty"`

	// EnergyMake is the energy the unit makes per second. Is 0 if it makes none.
	EnergyMake float64 `json:"energyMake,omitempty"`

	// WindGenerator is the maximum energy of a wind generator. Is 0 for other units.
	WindGenerator float64 `json:"windGenerator,omitempty"`

	// MaxWeaponRange is the range of the unit's longest-range weapon in elmos. Is 0 without weapons.
	MaxWeaponRange float64 `json:"maxWeaponRange,omitempty"`
}
