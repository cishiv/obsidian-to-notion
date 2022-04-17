package utils

/**
Map: Define a generic map function that accepts a slice<T> and a function(T): V,
that applies f(T):V to each element of slice<T> such that slice<T> is transformed to slice<V>
**/
func Map[T any, V any](vs []T, f func(T) V) []V {
	transform := make([]V, len(vs))
	for i, v := range vs {
		transform[i] = f(v)
	}
	return transform
}

/**
Contains: Assert whether slice<T> contains e<T>
**/
func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}