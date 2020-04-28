package main

import (
	"fmt"

	"github.com/docker/libnetwork/iptables"
)

type iptRule struct {
	table   iptables.Table
	chain   string
	args    []string
}

func programChainRule(rule iptRule, insert bool) error {
	var (
		prefix    []string
		operation string
		condition bool
		doesExist = iptables.Exists(rule.table, rule.chain, rule.args...)
	)

	if insert {
		condition = !doesExist
		prefix = []string{"-A", rule.chain}
		operation = "enable"
	} else {
		condition = doesExist
		prefix = []string{"-D", rule.chain}
		operation = "disable"
	}

	if condition {
		if err := iptables.RawCombinedOutput(append(prefix, rule.args...)...); err != nil {
			return fmt.Errorf("unable to %s rule: %s", operation, err.Error())
		}
	}

	return nil
}