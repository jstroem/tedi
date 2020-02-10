package tedi

type stringSet map[string]struct{}

func (s *stringSet) Add(strs ...string) {
	if len(strs) == 0 {
		return
	}

	if *s == nil {
		*s = stringSet{}
	}
	for _, str := range strs {
		(*s)[str] = struct{}{}
	}
}

func newStringSet(strs ...string) stringSet {
	var s stringSet
	s.Add(strs...)
	return s
}

func (s *stringSet) Has(str string) bool {
	if s == nil {
		return false
	}

	_, ok := (*s)[str]
	return ok
}

func (s *stringSet) List() []string {
	if s == nil {
		return nil
	}

	res := []string{}
	for k := range *s {
		res = append(res, k)
	}
	return res
}

func (s *stringSet) Intersect(b stringSet) stringSet {
	var res stringSet
	if s == nil || len(*s) == 0 || len(b) == 0 {
		return res
	}

	for k := range *s {
		if b.Has(k) {
			res.Add(k)
		}
	}

	return res
}
