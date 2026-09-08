// Copyright 2020 Baidu Inc. All rights reserved.
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

/*
modification history
--------------------
2026/9/7, by v_fangxiang01@baidu.com, create
*/

package aliyunopen

import (
	"Clould/webook/internal/service/sms"
	"context"
	"encoding/json"
	"fmt"

	dysmsapi "github.com/alibabacloud-go/dypnsapi-20170525/v3/client"
	util "github.com/alibabacloud-go/tea-utils/service"
	"github.com/alibabacloud-go/tea/tea"
)

type Service struct {
	client   *dysmsapi.Client
	signName string
}

func NewService(c *dysmsapi.Client, signName string) *Service {
	return &Service{
		client:   c,
		signName: signName,
	}
}

func (s *Service) Send(ctx context.Context, tplId string, args []sms.NamedArg, numbers ...string) error {
	// 传入的是 JSON
	req := &dysmsapi.SendSmsVerifyCodeRequest{}
	argsMap := make(map[string]string, len(args))
	for _, arg := range args {
		argsMap[arg.Name] = arg.Val
	}
	bCode, err := json.Marshal(argsMap)
	if err != nil {
		return err
	}
	req.SignName = new(s.signName)
	req.TemplateCode = new(tplId)
	req.TemplateParam = new(string(bCode)) // 按你模板实际格式填

	for _, n := range numbers {
		req.PhoneNumber = new(n)

		resp, err := s.client.SendSmsVerifyCode(req)
		if err != nil {
			return err
		}
		code := resp.Body.Code
		if !tea.BoolValue(util.EqualString(code, new("OK"))) {
			return fmt.Errorf("发送失败，code: %s, 原因：%s",
				tea.StringValue(resp.Body.Code),
				tea.StringValue(resp.Body.Message))
		}
	}
	return nil
}
