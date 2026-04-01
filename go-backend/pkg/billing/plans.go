package billing

import (
	"errors"
)

var (
	ErrPlanNotFound = errors.New("plan not found")
)

const (
	PlanFree       = "free"
	PlanPro        = "pro"
	PlanBusiness   = "business"
	PlanEnterprise = "enterprise"
)

var plans = map[string]Plan{
	PlanFree: {
		ID:          PlanFree,
		Name:        "Free",
		Price:       0,
		PriceYearly: 0,
		Interval:    BillingIntervalMonthly,
		Features: []string{
			"1,000 messages per month",
			"1 user",
			"1 agent",
			"2 channels",
			"100MB storage",
			"Community support",
			"Built-in skills",
		},
		Limits: PlanLimits{
			MaxUsers:        1,
			MaxAgents:       1,
			MaxMessages:     1000,
			MaxChannels:     2,
			MaxStorageBytes: 100 * 1024 * 1024,
		},
	},
	PlanPro: {
		ID:          PlanPro,
		Name:        "Pro",
		Price:       2900,
		PriceYearly: 29000,
		Interval:    BillingIntervalMonthly,
		Features: []string{
			"10,000 messages per month",
			"5 users",
			"5 agents",
			"5 channels",
			"5GB storage",
			"Email support",
			"Built-in skills",
			"Custom skills",
		},
		Limits: PlanLimits{
			MaxUsers:        5,
			MaxAgents:       5,
			MaxMessages:     10000,
			MaxChannels:     5,
			MaxStorageBytes: 5 * 1024 * 1024 * 1024,
		},
	},
	PlanBusiness: {
		ID:          PlanBusiness,
		Name:        "Business",
		Price:       9900,
		PriceYearly: 99000,
		Interval:    BillingIntervalMonthly,
		Features: []string{
			"50,000 messages per month",
			"25 users",
			"20 agents",
			"15 channels",
			"50GB storage",
			"Priority support",
			"Built-in skills",
			"Custom skills",
			"Marketplace access",
			"SSO",
			"Audit logs",
		},
		Limits: PlanLimits{
			MaxUsers:        25,
			MaxAgents:       20,
			MaxMessages:     50000,
			MaxChannels:     15,
			MaxStorageBytes: 50 * 1024 * 1024 * 1024,
		},
	},
	PlanEnterprise: {
		ID:          PlanEnterprise,
		Name:        "Enterprise",
		Price:       -1,
		PriceYearly: -1,
		Interval:    BillingIntervalMonthly,
		Features: []string{
			"Unlimited messages",
			"Unlimited users",
			"Unlimited agents",
			"All channels",
			"Custom storage",
			"Dedicated support",
			"Built-in skills",
			"Custom skills",
			"Marketplace access",
			"SSO",
			"Audit logs",
			"Custom integrations",
			"SLA guarantee",
			"Dedicated account manager",
		},
		Limits: PlanLimits{
			MaxUsers:        -1,
			MaxAgents:       -1,
			MaxMessages:     -1,
			MaxChannels:     -1,
			MaxStorageBytes: -1,
		},
	},
}

func GetPlans() []Plan {
	result := make([]Plan, 0, len(plans))
	order := []string{PlanFree, PlanPro, PlanBusiness, PlanEnterprise}
	for _, id := range order {
		if p, ok := plans[id]; ok {
			result = append(result, p)
		}
	}
	return result
}

func GetPlan(id string) (Plan, error) {
	plan, ok := plans[id]
	if !ok {
		return Plan{}, ErrPlanNotFound
	}
	return plan, nil
}

func GetPlanLimits(planID string) PlanLimits {
	plan, err := GetPlan(planID)
	if err != nil {
		return plans[PlanFree].Limits
	}
	return plan.Limits
}

func IsUnlimited(value int) bool {
	return value < 0
}

func IsStorageUnlimited(value int64) bool {
	return value < 0
}
