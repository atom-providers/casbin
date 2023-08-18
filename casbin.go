package casbin

import (
	"github.com/rogeecn/atom/container"
	"github.com/rogeecn/atom/utils/opt"
)

func Provide(opts ...opt.Option) error {
	o := opt.New(opts...)
	var config Auth
	if err := o.UnmarshalConfig(&config); err != nil {
		return err
	}
	return container.Container.Provide(func() *Auth {
		return &Auth{}
	}, o.DiOptions()...)
}

func (auth *Auth) GetModel() string {
	return auth.Model
}
