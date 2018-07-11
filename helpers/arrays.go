package helpers

func Contains(match string, haystack []string) bool {
	for _, value := range haystack {
		if value == match {
			return true
		}
	}

	return false
}

func ContainsInt(match int, haystack []int) bool {
	for _, value := range haystack {
		if value == match {
			return true
		}
	}

	return false
}
