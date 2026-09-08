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
	ali "Clould/webook/internal/service/sms"
	"context"
	"os"
	"testing"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	sms "github.com/alibabacloud-go/dypnsapi-20170525/v3/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/credentials-go/credentials"
	"github.com/stretchr/testify/assert"
)

func TestService_SendSMS(t *testing.T) {
	// 环境变量获

	// 初始化 credentialsConfig 阿里云管理访问凭证
	credentialsConfig := new(credentials.Config).
		SetType("access_key").
		SetAccessKeyId(os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_ID")).
		SetAccessKeySecret(os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET"))
	akCredential, err := credentials.NewCredential(credentialsConfig)
	if err != nil {
		return
	}

	config := &openapi.Config{Endpoint: tea.String("dypnsapi.aliyuncs.com")}
	config.Credential = akCredential

	// 初始化
	c := &sms.Client{}
	c, err = sms.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

	service := NewService(c, "云渚科技验证平台")
	tests := []struct {
		signName string
		tplCode  string
		phone    []string
		min      string
		wantErr  error
		args     []ali.NamedArg
	}{
		{
			signName: "云渚科技验证平台",
			tplCode:  "100001",
			phone:    []string{"19204969061"}, // []string{"18656898059", "19204969061", "17750408969"},
			min:      "10",
			args: []ali.NamedArg{
				{Name: "code", Val: "1520"}, // DYPNS 占位，由阿里云生成验证码
				{Name: "min", Val: "9"},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.signName, func(t *testing.T) {
			err := service.Send(context.Background(), tc.tplCode, tc.args, tc.phone...)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}
