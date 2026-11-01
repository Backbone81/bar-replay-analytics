package events

// GameSpeed is the number of game frames per second.
// See GAME_SPEED in rts/Sim/Misc/GlobalConstants.h.
const GameSpeed = 30

// Position is a position on the map in elmos, rounded down, as [x, z].
type Position [2]int

// X returns the x coordinate.
func (p Position) X() int {
	return p[0]
}

// Z returns the z coordinate.
func (p Position) Z() int {
	return p[1]
}
