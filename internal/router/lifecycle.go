package router

import "errors"

// CloseAppenders 按运行期关闭顺序关闭一组 appender。
func CloseAppenders(appenders []Appender, isAsyncAppender AsyncAppenderMatcher) error {
	config := &runtimeConfig{
		all:             appenders,
		isAsyncAppender: isAsyncAppender,
	}
	return config.close()
}

func (c *runtimeConfig) close() error {
	if c == nil {
		return nil
	}
	var joined error
	closed := make(map[string]struct{}, len(c.all))
	if c.isAsyncAppender != nil {
		for _, appender := range c.all {
			if appender != nil && c.isAsyncAppender(appender) {
				closed[appender.Name()] = struct{}{}
				joined = errors.Join(joined, appender.Close())
			}
		}
	}
	for _, appender := range c.all {
		if appender == nil {
			continue
		}
		if _, ok := closed[appender.Name()]; ok {
			continue
		}
		joined = errors.Join(joined, appender.Close())
	}
	return joined
}
