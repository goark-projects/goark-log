package goarklog

import (
	"context"
	"fmt"
)

// ScriptEvaluator 是脚本过滤器的执行器契约。
type ScriptEvaluator interface {
	Evaluate(ctx context.Context, event Event) (bool, error)
}

// ScriptEvaluatorFunc 把函数适配为 ScriptEvaluator。
type ScriptEvaluatorFunc func(ctx context.Context, event Event) (bool, error)

// Evaluate 执行脚本判断函数。
func (f ScriptEvaluatorFunc) Evaluate(ctx context.Context, event Event) (bool, error) {
	if f == nil {
		return false, fmt.Errorf("goark-log: script evaluator func is nil")
	}
	return f(ctx, event)
}

// ScriptFilterOption 调整 ScriptFilter。
type ScriptFilterOption func(*scriptFilterSettings)

type scriptFilterSettings struct {
	onError FilterDecision
	outcome filterOutcome
}

// WithScriptFilterOnMatch 设置脚本匹配时的裁决。
func WithScriptFilterOnMatch(decision FilterDecision) ScriptFilterOption {
	return func(settings *scriptFilterSettings) {
		settings.outcome.onMatch = decision
	}
}

// WithScriptFilterOnMismatch 设置脚本不匹配时的裁决。
func WithScriptFilterOnMismatch(decision FilterDecision) ScriptFilterOption {
	return func(settings *scriptFilterSettings) {
		settings.outcome.onMismatch = decision
	}
}

// WithScriptFilterOnError 设置脚本执行失败时的裁决。
func WithScriptFilterOnError(decision FilterDecision) ScriptFilterOption {
	return func(settings *scriptFilterSettings) {
		settings.onError = decision
	}
}

// ScriptFilter 用调用方提供的执行器过滤日志事件。
type ScriptFilter struct {
	evaluator ScriptEvaluator
	onError   FilterDecision
	outcome   filterOutcome
}

// NewScriptFilter 创建脚本过滤器。
func NewScriptFilter(evaluator ScriptEvaluator, options ...ScriptFilterOption) (*ScriptFilter, error) {
	if evaluator == nil {
		return nil, fmt.Errorf("goark-log: script evaluator is nil")
	}
	settings := scriptFilterSettings{
		onError: FilterDeny,
		outcome: filterOutcome{
			onMatch:    FilterNeutral,
			onMismatch: FilterDeny,
		},
	}
	for _, option := range options {
		if option != nil {
			option(&settings)
		}
	}
	return &ScriptFilter{evaluator: evaluator, onError: settings.onError, outcome: settings.outcome}, nil
}

// Decide 执行脚本判断，脚本错误默认按拒绝处理。
func (f *ScriptFilter) Decide(ctx context.Context, event Event) FilterDecision {
	if f == nil || f.evaluator == nil {
		return FilterNeutral
	}
	matched, err := f.evaluator.Evaluate(ctx, event)
	if err != nil {
		return f.onError
	}
	return f.outcome.decide(matched)
}
