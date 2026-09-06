package goarklog

import configlookup "goark.dev/log/internal/lookup"

// PropertyResolver 是 boot 配置系统需要适配的最小读取接口。
type PropertyResolver interface {
	GetProperty(key string) (string, bool)
}

// PropertyMap 是测试和轻量嵌入场景可直接使用的配置适配器。
type PropertyMap map[string]string

func (m PropertyMap) GetProperty(key string) (string, bool) {
	value, ok := m[key]
	return value, ok
}

// LookupFunc 根据键解析配置变量。
type LookupFunc = configlookup.Func

// LookupResolver 负责解析配置中的 ${namespace:key} 变量。
type LookupResolver = configlookup.Resolver

// NewLookupResolver 创建带默认 lookup 的解析器。
func NewLookupResolver() *LookupResolver {
	return configlookup.NewResolver()
}
