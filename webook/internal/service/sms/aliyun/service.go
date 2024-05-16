package aliyun

import (
	"context"

	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v2/client"
	console "github.com/alibabacloud-go/tea-console/client"
	util "github.com/alibabacloud-go/tea-utils/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/ecodeclub/ekit"
)

type Service struct {
	appId    *string
	signName *string
	client   *dysmsapi.Client
}

func NewService(client *dysmsapi.Client, appId string, signName string) *Service {
	return &Service{
		appId:    &appId,
		signName: &signName,
		client:   client,
	}
}

func CreateClient(accessKeyId *string, accessKeySecret *string) (_result *dysmsapi.Client, _err error) {
	config := &openapi.Config{}
	config.AccessKeyId = accessKeyId
	config.AccessKeySecret = accessKeySecret
	_result = &dysmsapi.Client{}
	_result, _err = dysmsapi.NewClient(config)
	return _result, _err
}

func (s *Service) Send(ctx context.Context, tplId string,
	args []string, numbers ...string) error {
	// 1.发送短信
	sendReq := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  s.toPtrSlice(numbers)[0],
		SignName:      s.signName,
		TemplateCode:  ekit.ToPtr[string](tplId),
		TemplateParam: s.toPtrSlice(args)[0],
	}
	sendResp, _err := s.client.SendSms(sendReq)
	if _err != nil {
		return _err
	}

	code := sendResp.Body.Code
	if !tea.BoolValue(util.EqualString(code, tea.String("OK"))) {
		console.Log(tea.String("错误信息: " + tea.StringValue(sendResp.Body.Message)))
		return _err
	}

	return nil
}

// 将字符串切片转换为指针切片
func (s *Service) toPtrSlice(data []string) []*string {
	result := make([]*string, len(data))
	for i, v := range data {
		value := v
		result[i] = &value
	}
	return result
}
