package casbin

import (
	"errors"
	"strconv"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/rogeecn/atom/container"
	"github.com/rogeecn/atom/utils/opt"
	"gorm.io/gorm"
)

type Casbin struct {
	model    model.Model
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

		return &Casbin{adapter: a, enforcer: e, model: model}, nil
	}, o.DiOptions()...)
	return err
}

func (c *Casbin) Reload() (err error) {
	c.enforcer, err = casbin.NewEnforcer(c.model, c.adapter)
	if err != nil {
		return err
	}

	return c.adapter.LoadPolicy(c.model)
}

func (c *Casbin) Check(userID, tenantID int64, path, action string) bool {
	ok, err := c.enforcer.Enforce(strconv.Itoa(int(userID)), strconv.Itoa(int(tenantID)), path, action)
	if err != nil {
		return false
	}
	return ok
}
