package service

import (
	"Clould/webook/internal/repository"
	"Clould/webook/internal/service/sms"
	"context"
	"fmt"
	"math/rand"
	"strconv"
)

const codeTplId = "100001"

var (
	ErrCodeVerifyTooManyTimes = repository.ErrCodeVerifyTooManyTimes
	ErrCodeSendTooMany        = repository.ErrCodeSendTooMany
)

type CodeService struct {
	repo   *repository.CodeRepository
	smsSvc sms.Service
}

func NewCodeService(repo *repository.CodeRepository, smsSvc sms.Service) *CodeService {
	return &CodeService{
		repo:   repo,
		smsSvc: smsSvc,
	}
}

// 发验证码
func (svc *CodeService) Send(ctx context.Context,
	// 区别业务场景
	biz string,
	// 验证码 谁来生成
	phone string) error {
	// 生成一个验证码
	code := svc.generateCode()
	// 塞进去 Redis
	err := svc.repo.Store(ctx, biz, phone, code)
	if err != nil {
		// 有问题
		return err
	}
	// 验证码过期时间 ttl
	ttl := 10
	err = svc.smsSvc.Send(ctx, codeTplId, []sms.NamedArg{
		{Name: "code", Val: code}, // 验证码
		{Name: "min", Val: strconv.Itoa(ttl)},
	}, phone)
	return nil
}

func (svc *CodeService) Verify(ctx context.Context, biz string,
	phone string, inputCode string) (bool, error) {
	return svc.repo.Verify(ctx, biz, phone, inputCode)
}
func (svc *CodeService) VerifyV1(ctx context.Context,
	biz string, phone, inputCode string) error {
	return nil
}

func (svc *CodeService) generateCode() string {
	// 六位数，num 在 0, 999999 之间，包含 0 和 999999
	num := rand.Intn(1000000)
	// 不够六位的，加上前导 0
	// 000001
	return fmt.Sprintf("%06d", num)
}
