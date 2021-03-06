// +build linux

package native

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/seccomp"
	"github.com/opencontainers/specs"
)

func loadSeccompProfile(path string) (*configs.Seccomp, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Opening seccomp profile failed: %v", err)
	}

	var config specs.Seccomp
	if err := json.Unmarshal(f, &config); err != nil {
		return nil, fmt.Errorf("Decoding seccomp profile failed: %v", err)
	}

	return setupSeccomp(&config)
}

func setupSeccomp(config *specs.Seccomp) (newConfig *configs.Seccomp, err error) {
	if config == nil {
		return nil, nil
	}

	// No default action specified, no syscalls listed, assume seccomp disabled
	if config.DefaultAction == "" && len(config.Syscalls) == 0 {
		return nil, nil
	}

	newConfig = new(configs.Seccomp)
	newConfig.Syscalls = []*configs.Syscall{}

	// if config.Architectures == 0 then libseccomp will figure out the architecture to use
	if len(config.Architectures) > 0 {
		newConfig.Architectures = []string{}
		for _, arch := range config.Architectures {
			newArch, err := seccomp.ConvertStringToArch(string(arch))
			if err != nil {
				return nil, err
			}
			newConfig.Architectures = append(newConfig.Architectures, newArch)
		}
	}

	// Convert default action from string representation
	newConfig.DefaultAction, err = seccomp.ConvertStringToAction(string(config.DefaultAction))
	if err != nil {
		return nil, err
	}

	// Loop through all syscall blocks and convert them to libcontainer format
	for _, call := range config.Syscalls {
		newAction, err := seccomp.ConvertStringToAction(string(call.Action))
		if err != nil {
			return nil, err
		}

		newCall := configs.Syscall{
			Name:   call.Name,
			Action: newAction,
			Args:   []*configs.Arg{},
		}

		// Loop through all the arguments of the syscall and convert them
		for _, arg := range call.Args {
			newOp, err := seccomp.ConvertStringToOperator(string(arg.Op))
			if err != nil {
				return nil, err
			}

			newArg := configs.Arg{
				Index:    arg.Index,
				Value:    arg.Value,
				ValueTwo: arg.ValueTwo,
				Op:       newOp,
			}

			newCall.Args = append(newCall.Args, &newArg)
		}

		newConfig.Syscalls = append(newConfig.Syscalls, &newCall)
	}

	return newConfig, nil
}
