package postgres

func topoOrder[K comparable, T any](items []T, key func(T) K, deps func(T) []K) []T {
	index := map[K]int{}
	for i, it := range items {
		index[key(it)] = i
	}
	done := make([]bool, len(items))
	visiting := make([]bool, len(items))
	out := make([]T, 0, len(items))
	var visit func(i int)
	visit = func(i int) {
		if done[i] || visiting[i] {
			return
		}
		visiting[i] = true
		for _, d := range deps(items[i]) {
			if j, ok := index[d]; ok && j != i {
				visit(j)
			}
		}
		visiting[i] = false
		done[i] = true
		out = append(out, items[i])
	}
	for i := range items {
		visit(i)
	}
	return out
}
