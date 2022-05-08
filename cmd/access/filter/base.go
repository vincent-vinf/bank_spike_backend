package filter

import (
	"bank_spike_backend/internal/orm"
	"context"
	"encoding/json"
	"sort"
)

var (
	Map = make(map[string]func(rule string) (Filter, error))
)

func init() {
	Map["base"] = NewBaseFilter
	Map["untrustworthy"] = NewUntrustworthyFilter
}

type Filter interface {
	Execute(ctx context.Context, user *orm.User) (res bool, reason string, err error)
	SetNext(next Filter)
}

type BaseFilter struct {
	next Filter
	rule *BaseRule
}

func NewBaseFilter(rule string) (Filter, error) {
	r := &BaseRule{}
	err := json.Unmarshal([]byte(rule), r)
	if err != nil {
		return nil, err
	}
	// 排序后使用二分查找
	sort.StringSlice(r.WorkStatus.Not).Sort()
	return &BaseFilter{
		rule: r,
	}, nil
}

func (f *BaseFilter) Execute(ctx context.Context, user *orm.User) (res bool, reason string, err error) {
	if user.Age < f.rule.Age.Min || (f.rule.Age.Max != 0 && user.Age > f.rule.Age.Max) {
		return false, "age does not meet the requirement", nil
	}
	pos := sort.SearchStrings(f.rule.WorkStatus.Not, user.WorkStatus)
	if pos != len(f.rule.WorkStatus.Not) && f.rule.WorkStatus.Not[pos] == user.WorkStatus {
		return false, "work status does not meet the requirement", nil
	}
	res = true
	if f.next == nil {
		return
	}
	return f.next.Execute(ctx, user)
}

func (f *BaseFilter) SetNext(next Filter) {
	f.next = next
}

// {"age":{"min":0, "max":20}, "workStatus":{"not":["无业"]}}
// {\"age\":{\"min\":0, \"max\":20}, \"workStatus\":{\"not\":[\"无业\"]}}

type BaseRule struct {
	// 年龄需要在此范围内，闭区间,为0则为无要求
	Age struct {
		Min int `json:"min"`
		Max int `json:"max"`
	} `json:"age"`
	WorkStatus struct {
		Not []string `json:"not"`
	} `json:"workStatus"`
}

type Info struct {
	Name string `json:"name"`
	Rule string `json:"rule"`
}
