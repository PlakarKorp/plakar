package location

import (
	"slices"
	"strings"
	"sync"
)

type Location[T any] struct {
	mtx      sync.Mutex
	items    map[string]T
	fallback string
}

func New[T any](fallback string) *Location[T] {
	return &Location[T]{
		items:    make(map[string]T),
		fallback: fallback,
	}
}

func (l *Location[T]) Register(name string, item T) bool {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if _, ok := l.items[name]; ok {
		return false
	}
	l.items[name] = item
	return true
}

func (l *Location[T]) Names() []string {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	ret := make([]string, 0, len(l.items))
	for name := range l.items {
		ret = append(ret, name)
	}
	slices.Sort(ret)
	return ret
}

func allowedInUri(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
		c == '+' || c == '-' || c == '.'
}

func (l *Location[T]) Lookup(uri string) (proto, location string, item T, ok bool) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	proto = uri
	location = uri

	for i, c := range uri {
		if !allowedInUri(c) {
			if i != 0 && strings.HasPrefix(uri[i:], ":") {
				proto = uri[:i]
				location = uri[i+1:]
				location = strings.TrimPrefix(location, "//")
			}
			break
		}
	}

	if proto == location {
		proto = l.fallback
	}

	item, ok = l.items[proto]
	return
}
