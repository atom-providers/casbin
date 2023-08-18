package casbin

import (
	"errors"

	"github.com/atom-providers/log"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/rogeecn/atom/container"
	"github.com/rogeecn/atom/utils/opt"
	"gorm.io/gorm"
)

type Casbin struct {
	adapter  *Adapter
	enforcer *casbin.Enforcer
}

func Provide(opts ...opt.Option) error {
	o := opt.New(opts...)
	var config Config
	if err := o.UnmarshalConfig(&config); err != nil {
		return err
	}

	err := container.Container.Provide(func(db *gorm.DB) (*Casbin, error) {
		model, err := model.NewModelFromString(config.Model)
		if err != nil {
			return nil, err
		}

		a, err := NewAdapterByDBUseTableName(db, "", config.AdapterTableName)
		if err != nil {
			return nil, err
		}

		if _, ok := any(a).(persist.Adapter); !ok {
			return nil, errors.New("adapter must implement persist.Adapter")
		}

		e, err := casbin.NewEnforcer(model, a)
		if err != nil {
			return nil, err
		}

		return &Casbin{adapter: a, enforcer: e}, nil
	}, o.DiOptions()...)
	return err
}

func (c *Casbin) Check(sub, obj, act string) bool {
	ok, err := c.enforcer.Enforce(sub, obj, act)
	if err != nil {
		log.Error(err)
		return false
	}
	return ok
}
