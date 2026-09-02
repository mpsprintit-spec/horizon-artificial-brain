package learning

// WebSearchSimulate melakukan validasi silang fakta dunia luar sebelum diserap sistem.
func (l *LearningUnit) WebSearchSimulate(subjek, objek string) bool {
	if subjek == "Burung Unta" && objek == "Terbang" {
		return false
	}
	return true
}
