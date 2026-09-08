// Copyright 2020 Baidu Inc. All rights reserved.
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

/*
modification history
--------------------
2026/9/7, by v_fangxiang01@baidu.com, create
*/

package sms

import "context"

type Service interface {
	Send(ctx context.Context, tplId string, args []NamedArg, numbers ...string) error

	//SendV1(ctx context.Context, tpl string, args []NamedArg, numbers ...string) error
	// 调用者需要知道实现者需要什么类型的参数，是 []string，还是 map[string]string
	//SendV2(ctx context.Context, tpl string, args any, numbers ...string) error
	//SendVV3(ctx context.Context, tpl string, args T, numbers ...string) error
}
type NamedArg struct {
	Val  string
	Name string
}
