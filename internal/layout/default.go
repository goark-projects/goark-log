package layout

// DefaultSpringBootPattern 是默认控制台输出格式，风格对齐 Spring Boot。
const DefaultSpringBootPattern = "%d{yyyy-MM-dd HH:mm:ss.SSS}  %5level %pid - " +
	"[%15.15thread] %-40.40logger{1.2*} : %msg%attrs%n"

// NewDefaultLayout 创建默认 Spring Boot 风格布局。
func NewDefaultLayout() Layout {
	layout, _ := NewPatternLayout(DefaultSpringBootPattern)
	return layout
}
