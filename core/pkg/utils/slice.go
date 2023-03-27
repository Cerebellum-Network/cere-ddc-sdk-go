package utils

func RemoveSorted[T any](slice []T, i int) []T {
	if i < 0 || len(slice) <= i {
		return slice
	}

	return append(slice[:i], slice[i+1:]...)[: len(slice)-1 : len(slice)-1]
}
