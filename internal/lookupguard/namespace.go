package lookupguard

import "strings"

// BlockedNamespace 判断 namespace 是否属于核心库禁止注册的远程解析能力。
func BlockedNamespace(namespace string) bool {
	switch strings.ToLower(strings.TrimSpace(namespace)) {
	case "jndi", "ldap", "rmi":
		return true
	default:
		return false
	}
}
