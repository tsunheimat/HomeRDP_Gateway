package security

import (
	"context"
	"errors"
	"fmt"
	"log"
)

var (
	Hosts           []string
	HostSelection   string
	ManagedHostList func() ([]string, error)
)

func CheckHost(ctx context.Context, host string) (bool, error) {
	if ManagedHostList == nil {
		return false, errors.New("managed host inventory not configured")
	}

	hosts, err := ManagedHostList()
	if err != nil {
		return false, err
	}
	if len(hosts) == 0 {
		return false, errors.New("no enabled managed hosts configured")
	}

	switch HostSelection {
	case "any":
		fallthrough
	case "signed":
		// todo get from context?
		if HostSelection == "signed" {
			return false, errors.New("cannot verify host in 'signed' mode as token data is missing")
		}
		fallthrough
	case "roundrobin", "unsigned":
		s := getTunnel(ctx)
		if s.User.UserName() == "" {
			return false, errors.New("no valid session info or username found in context")
		}

		log.Printf("Checking host for user %s", s.User.UserName())
		for _, h := range hosts {
			resolved, err := ResolvePreferredUsernameHost(h, s.User.UserName())
			if err != nil {
				log.Printf("Ignoring invalid configured host template %q for user %s", h, s.User.UserName())
				continue
			}
			if resolved == host {
				return true, nil
			}
		}
		return false, fmt.Errorf("invalid host %s", host)
	}

	return false, errors.New("unrecognized host selection criteria")
}
