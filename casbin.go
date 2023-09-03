package casbin

import (
	"strconv"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/rogeecn/atom/container"
	"github.com/rogeecn/atom/utils/opt"
	"gorm.io/gorm"
)

type Casbin struct {
	model    model.Model
	Loaded   bool
	Enforcer *casbin.Enforcer
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
		e, err := casbin.NewEnforcer(model)
		if err != nil {
			return nil, err
		}

		return &Casbin{Enforcer: e, model: model}, nil
	}, o.DiOptions()...)
	return err
}

func (c *Casbin) Check(userID, tenantID uint64, path, action string) bool {
	ok, err := c.Enforcer.Enforce(strconv.Itoa(int(userID)), strconv.Itoa(int(tenantID)), path, action)
	if err != nil {
		return false
	}
	return ok
}

func (c *Casbin) LoadPolicies(rules [][]string) (bool, error) {
	return c.Enforcer.AddPoliciesEx(rules)
}

func (c *Casbin) LoadGroups(groups [][]string) (bool, error) {
	return c.Enforcer.AddGroupingPoliciesEx(groups)
}
