package service

import (
	"Clould/webook/internal/service/sms"
	"context"
)

type CodeService struct {
	repo *repository.CodeRepository
	sms  sms.Service
}

// 发验证码
func (svc *CodeService) Send(ctx context.Context,
	// 区别业务场景
	biz string,
	// 验证码 谁来生成
	phone string) error {

}

func (svc *CodeService) Verify(ctx context.Context,
	biz string, phone, inputCode string) (bool, error) {

}

func (svc *CodeService) VerifyV1(ctx context.Context,
	biz string, phone, inputCode string) error {

}
