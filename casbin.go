package casbin

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/rogeecn/atom/container"
	"github.com/rogeecn/atom/utils/opt"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type Casbin struct {
	db       *gorm.DB
	model    model.Model
	Loaded   bool
	Enforcer *casbin.Enforcer

	whitelistMatchItems   []string
	whitelistPatternItems []*regexp.Regexp
	whitelistLastID       uint64
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

		casbin := &Casbin{Enforcer: e, model: model, db: db}
		casbin.Enforcer.AddFunction("is_white", casbin.FuncIsWhite)

		return casbin, nil
	}, o.DiOptions()...)
	return err
}

func (c *Casbin) FuncIsWhite(args ...interface{}) (interface{}, error) {
	type model struct {
		ID    uint64
		Route string
	}
	var lastID model
	if err := c.db.Table("route_whitelists").Select("id").Order("id desc").First(&lastID).Error; err != nil {
		return false, err
	}

	if lastID.ID != c.whitelistLastID {
		var items []model
		if err := c.db.Table("route_whitelists").Select("route").Find(&items).Error; err != nil {
			return false, err
		}
		c.whitelistMatchItems = []string{}
		c.whitelistPatternItems = []*regexp.Regexp{}

		for _, item := range items {
			if strings.Contains(item.Route, "*") {
				c.whitelistPatternItems = append(c.whitelistPatternItems, routeToPattern(item.Route))
				continue
			}

			if strings.Contains(item.Route, "{") && strings.Contains(item.Route, "}") {
				c.whitelistPatternItems = append(c.whitelistPatternItems, routeToPattern(item.Route))
				continue
			}
			c.whitelistMatchItems = append(c.whitelistMatchItems, item.Route)
		}

		c.whitelistLastID = lastID.ID
	}

	route := args[0].(string)
	if lo.Contains(c.whitelistMatchItems, route) {
		return true, nil
	}
	for _, item := range c.whitelistPatternItems {
		if item.MatchString(route) {
			return true, nil
		}
	}

	return false, nil
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

// https://github.com/casbin/casbin/blob/master/util/builtin_operators.go#L175
// KeyMatch3 determines whether key1 matches the pattern of key2 (similar to RESTful path), key2 can contain a *.
// For example, "/foo/bar" matches "/foo/*", "/resource1" matches "/{resource}"
func routeToPattern(route string) *regexp.Regexp {
	route = strings.Replace(route, "/*", "/.*", -1)

	re := regexp.MustCompile(`\{[^/]+\}`)
	route = re.ReplaceAllString(route, "$1[^/]+$2")

	return regexp.MustCompile("^" + route + "$")
}
