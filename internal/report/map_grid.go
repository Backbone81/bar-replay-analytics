package report

import (
	"strconv"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
)

// MapGrid divides the map into cells, so that the report can tell where something happened (Player.LossLocations)
// without coordinates. The cells are at most Config.MapGridCellSize along each side, so they are about the same size on
// every map, as weapon ranges and bases are: a larger map has more cells, not larger ones.
//
// Columns are named A, B, ... from left to right (x), rows are numbered 1, 2, ... from top to bottom (z), as the
// minimap shows the map: A1 is its top left cell.
type MapGrid struct {
	// MapSize is the size of the map in elmos.
	MapSize Size `json:"mapSize"`

	// Columns is the number of columns, along x.
	Columns int `json:"columns"`

	// Rows is the number of rows, along z.
	Rows int `json:"rows"`

	// CellSize is the size of a cell in elmos, rounded down.
	CellSize Size `json:"cellSize"`

	// StartPositions provides where each player started, sorted by player ID. A player of whom the events have no start
	// unit is left out.
	StartPositions []StartPosition `json:"startPositions"`
}

// Size is a size on the map, in elmos.
type Size struct {
	// X is the size along x, from left to right on the minimap.
	X int `json:"x"`

	// Z is the size along z, from top to bottom on the minimap.
	Z int `json:"z"`
}

// StartPosition is where a player started: where its start unit, usually its commander, was spawned.
type StartPosition struct {
	// PlayerID is the ID of the player.
	PlayerID int `json:"playerID"` //nolint:tagliatelle

	// Cell is the cell of the grid the player started in, like "A4".
	Cell string `json:"cell"`

	// X is the position along x in elmos.
	X int `json:"x"`

	// Z is the position along z in elmos.
	Z int `json:"z"`
}

// newMapGrid prepares the map grid of the map of the events file with cells of at most config.MapGridCellSize, and
// the start positions of its numPlayers players: the position of the first unit created for a player which no builder
// started. eventsFile must be valid (events.File.Validate), so the map has a size.
func newMapGrid(eventsFile *events.File, numPlayers int, config Config) MapGrid {
	mapSize := Size{X: eventsFile.Game.MapSizeX, Z: eventsFile.Game.MapSizeZ}
	grid := MapGrid{
		MapSize:        mapSize,
		Columns:        cellCount(mapSize.X, config.MapGridCellSize),
		Rows:           cellCount(mapSize.Z, config.MapGridCellSize),
		StartPositions: []StartPosition{},
	}
	grid.CellSize = Size{X: mapSize.X / grid.Columns, Z: mapSize.Z / grid.Rows}

	started := make([]*StartPosition, numPlayers)
	for _, event := range eventsFile.Events {
		created := event.UnitCreated
		if created == nil || created.BuilderID != nil || created.UnitTeam < 0 || created.UnitTeam >= numPlayers {
			continue
		}
		if started[created.UnitTeam] == nil {
			started[created.UnitTeam] = &StartPosition{
				PlayerID: created.UnitTeam,
				Cell:     grid.cell(created.X, created.Z),
				X:        created.X,
				Z:        created.Z,
			}
		}
	}
	for _, position := range started {
		if position != nil {
			grid.StartPositions = append(grid.StartPositions, *position)
		}
	}
	return grid
}

// cellCount returns the number of cells of at most cellSize to divide length into. cellSize must be positive.
func cellCount(length int, cellSize int) int {
	return max(1, (length+cellSize-1)/cellSize)
}

// cell returns the name of the cell of the position x, z in elmos, like "C2". A position off the map is in the cell at
// the edge nearest to it.
func (g MapGrid) cell(x int, z int) string {
	column := min(max(0, x*g.Columns/g.MapSize.X), g.Columns-1)
	row := min(max(0, z*g.Rows/g.MapSize.Z), g.Rows-1)
	return columnName(column) + strconv.Itoa(row+1)
}

// columnName returns the name of the column with index i, counted from 0, as spreadsheets name them: A to Z, then AA,
// AB, and so on.
func columnName(i int) string {
	name := ""
	for i++; i > 0; i = (i - 1) / 26 {
		name = string(rune('A'+(i-1)%26)) + name
	}
	return name
}
