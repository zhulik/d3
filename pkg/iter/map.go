package iter

func ErrMap[T any, R any](iter []T, fn func(T) (R, error)) ([]R, error) {
	var result []R
	for _, item := range iter {
		mapped, err := fn(item)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func ErrFilterMap[T any, R any](iter []T, fn func(T) (R, bool, error)) ([]R, error) {
	var result []R
	for _, item := range iter {
		mapped, ok, err := fn(item)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		result = append(result, mapped)
	}
	return result, nil
}
