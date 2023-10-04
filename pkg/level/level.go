package level


func Level(score float32) int {
	level := 0
	switch{
	case score < 25: 
		level = 1
	case score < 45: 
		level = 2
	case score < 65: 
		level = 3
	case score < 85: 
		level = 4
	default: 
		level = 5
	}
	return level
}
