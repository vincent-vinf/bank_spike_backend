package filter

import (
	"bank_spike_backend/internal/orm"
	"context"
)

type UntrustworthyFilter struct {
	next Filter
	rule *UntrustworthyRule
}

func NewUntrustworthyFilter(rule string) (Filter, error) {
	return &UntrustworthyFilter{}, nil
}

func (f *UntrustworthyFilter) Execute(ctx context.Context, user *orm.User) (res bool, reason string, err error) {
	res = true
	if f.next == nil {
		return
	}
	return f.next.Execute(ctx, user)
}

func (f *UntrustworthyFilter) SetNext(next Filter) {
	f.next = next
}

type UntrustworthyRule struct {
}
