// Package goarklog 提供基于 log/slog 的 Goark 日志实现。
//
// 推荐的稳定入口是 NewHandler、NewConfigured、ConfigureDefault、Appender、
// Layout、LayoutOptions、Options 以及各 appender 的 Option 构造函数。YAML 文件结构由
// LoadOptions 和 NewConfigured 系列函数解析，内部解析结构不作为公共 API 暴露。
package goarklog
