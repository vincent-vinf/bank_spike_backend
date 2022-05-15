package singleflight

import "sync"

// 保存请求的返回值供其他请求使用，并用WaitGroup阻塞其他请求
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group 维护一个map，通过key找到请求
// Mutex用来保证map的线程安全
type Group struct {
	m  map[string]*call
	mu sync.Mutex
}

// Do 传入一个key，和一个函数，函数只会被执行第一次
// 后来的请求会被阻塞，待第一个请求完成后，后续的请求使用相同的返回值
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		// 阻塞
		c.wg.Wait()
		return c.val, c.err
	}
	c := &call{}
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()
	// 执行
	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
