package slicesx

func Map[T any, R any](slice []T, f func(T) R) []R {
	result := make([]R, len(slice))
	for i, v := range slice {
		result[i] = f(v)
	}
	return result
}

func MapWithError[T any, R any](slice []T, f func(T) (R, error)) ([]R, error) {
	var err error
	result := make([]R, len(slice))
	for i, v := range slice {
		result[i], err = f(v)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
