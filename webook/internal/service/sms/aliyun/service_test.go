package aliyun

import (
	"context"
	"testing"

	env "github.com/alibabacloud-go/darabonba-env/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/go-playground/assert/v2"
)

/*type Service struct {
}

func NewService(client *sms.Client, appId string, signName string) *Service {
	return &Service{}
}

func (s *Service) Send(ctx context.Context, tplId string,
	args []string, numbers ...string) error {

}*/

func TestSender(t *testing.T) {
	c, err := CreateClient(env.GetEnv(tea.String("ALIBABA_CLOUD_ACCESS_KEY_ID")), env.GetEnv(tea.String("ALIBABA_CLOUD_ACCESS_KEY_SECRET")))
	if err != nil {
		t.Fatal()
	}
	s := NewService(c, "SMS_154950909", "阿里云短信测试")

	testCases := []struct {
		name    string
		tplId   string
		params  []string
		numbers []string
		wantErr error
	}{
		{
			name:   "阿里云短信测试",
			tplId:  "SMS_154950909",
			params: []string{"{\"code\":\"0803\"}"},
			// 改成你的手机号码
			numbers: []string{"18656898059"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			er := s.Send(context.Background(), tc.tplId, tc.params, tc.numbers...)
			assert.Equal(t, tc.wantErr, er)
		})
	}

}
