// Copyright 2020 Baidu Inc. All rights reserved.
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

/*
modification history
--------------------
2026/9/8, by v_fangxiang01@baidu.com, create
*/

package ioc

import (
	"Clould/webook/internal/service/sms"
	"Clould/webook/internal/service/sms/aliyunopen"
	"os"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	alisms "github.com/alibabacloud-go/dypnsapi-20170525/v3/client"

	"github.com/alibabacloud-go/tea/tea"

	"github.com/aliyun/credentials-go/credentials"
)

/*
func InitSMSService() sms.Service {
	// 换内存，还是换别的
	return memory.NewService()
}
*/

func InitSMSService() sms.Service {
	// 阿里云
	sc, err := createAliyunSmsClient()
	if err != nil {
		panic("aliyun sms client create fail")
	}

	return aliyunopen.NewService(sc, "云渚科技验证平台")
}

func createAliyunSmsClient() (
	*alisms.Client,
	error,
) {
	// 初始化 credentialsConfig 阿里云管理访问凭证
	credentialsConfig := new(credentials.Config).
		SetType("access_key").
		SetAccessKeyId(os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_ID")).
		SetAccessKeySecret(os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET"))
	akCredential, err := credentials.NewCredential(credentialsConfig)
	if err != nil {
		return nil, err
	}

	cfg := &openapi.Config{Endpoint: tea.String("dypnsapi.aliyuncs.com")}
	cfg.Credential = akCredential

	// 初始化
	c := &alisms.Client{}
	c, err = alisms.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return c, nil

}
