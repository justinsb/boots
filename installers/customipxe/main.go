package customipxe

import (
	"context"
	"strings"

	"github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

type installer struct {
	extraIPXEVars [][]string
}

// pass vars here.
func Installer(dynamicIPXEVars [][]string) job.BootScripter {
	i := installer{
		extraIPXEVars: dynamicIPXEVars,
	}

	return i
}

func (i installer) BootScript(string) job.BootScript {
	return i.setBootScript
}

func (i installer) setBootScript(_ context.Context, j job.Job, s *ipxe.Script) {
	logger := j.Logger.With("installer", "custom_ipxe")

	var cfg *client.InstallerData
	switch {
	case j.OperatingSystem().Installer == "custom_ipxe":
		cfg = j.OperatingSystem().InstallerData
		if cfg == nil {
			s.Echo("Installer data not provided")
			s.Shell()
			logger.Error(ErrEmptyIPXEConfig, "installer data not provided")

			return
		}
	case strings.HasPrefix(j.UserData(), "#!ipxe"):
		cfg = &client.InstallerData{Script: j.UserData()}
	case j.IPXEScriptURL() != "":
		cfg = &client.InstallerData{Chain: j.IPXEScriptURL()}
	default:
		s.Echo("Unknown ipxe configuration")
		s.Shell()
		logger.Error(ErrEmptyIPXEConfig, "unknown ipxe configuration")

		return
	}

	for _, kv := range i.extraIPXEVars {
		s.Set(kv[0], kv[1])
	}
	ipxeScriptFromConfig(logger, cfg, j, s)
}

func ipxeScriptFromConfig(logger log.Logger, cfg *client.InstallerData, j job.Job, s *ipxe.Script) {
	if err := validateConfig(cfg); err != nil {
		s.Echo(err.Error())
		s.Shell()
		logger.Error(err, "validating ipxe config")

		return
	}

	s.PhoneHome("provisioning.104.01")
	s.Set("packet_facility", j.FacilityCode())
	s.Set("packet_plan", j.PlanSlug())

	if cfg.Chain != "" {
		s.Chain(cfg.Chain)
	} else if cfg.Script != "" {
		s.AppendString(strings.TrimPrefix(cfg.Script, "#!ipxe"))
	}
}

func validateConfig(c *client.InstallerData) error {
	if c.Chain == "" && c.Script == "" {
		return ErrEmptyIPXEConfig
	}

	return nil
}
