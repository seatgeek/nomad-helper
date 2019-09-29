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

func DeleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}
