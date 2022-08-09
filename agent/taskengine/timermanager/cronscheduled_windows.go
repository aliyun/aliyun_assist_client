package timermanager

// Embed TZ database for Windows which does not contain it by default
import (
	_ "time/tzdata"
)
