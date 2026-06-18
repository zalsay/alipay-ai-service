package a2a

import (
	"github.com/zalsay/alipay-ai-service/internal/alipay"
	"github.com/zalsay/alipay-ai-service/internal/config"
)

type Service struct {
	cfg    config.Config
	client *alipay.Client
}

func NewService(cfg config.Config) *Service {
	return &Service{cfg: cfg, client: alipay.NewClient(cfg)}
}
