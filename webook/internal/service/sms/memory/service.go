// Copyright 2020 Baidu Inc. All rights reserved.
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

/*
modification history
--------------------
2026/9/7, by v_fangxiang01@baidu.com, create
*/

package memory

import (
	"Clould/webook/internal/service/sms"
	"context"
	"fmt"
)

type Service struct {
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Send(ctx context.Context, tplId string, args []sms.NamedArg, numbers ...string) error {
	fmt.Println(tplId)
	return nil
}
