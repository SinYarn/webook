// Copyright 2020 Baidu Inc. All rights reserved.
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

/*
modification history
--------------------
2026/9/8, by v_fangxiang01@baidu.com, create
*/

package web

type Result struct {
	// 这个叫做业务错误码
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}
