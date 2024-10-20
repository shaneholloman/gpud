package command

import (
	"fmt"
	"time"

	"github.com/leptonai/gpud/config"
	"github.com/leptonai/gpud/version"

	"github.com/urfave/cli"
)

const usage = `
# to quick scan for your machine health status
gpud scan

# to start gpud as a systemd unit
sudo gpud up
`

var (
	logLevel string
	debug    bool
	uid      string

	annotations   string
	listenAddress string

	pprof bool

	retentionPeriod           time.Duration
	refreshComponentsInterval time.Duration

	webEnable        bool
	webAdmin         bool
	webRefreshPeriod time.Duration

	tailLines     int
	createArchive bool

	pollXidEvents bool
	pollGPMEvents bool
	netcheck      bool

	enableAutoUpdate   bool
	autoUpdateExitCode int

	filesToCheck cli.StringSlice

	dockerIgnoreConnectionErrors  bool
	kubeletIgnoreConnectionErrors bool
)

const (
	inProgress  = "\033[33m⌛\033[0m"
	checkMark   = "\033[32m✔\033[0m"
	warningSign = "\033[31m✘\033[0m"
)

func App() *cli.App {
	app := cli.NewApp()

	app.Name = "gpud"
	app.Version = version.Version
	app.Usage = usage
	app.Description = "monitor your GPU/CPU machines and run workloads"

	app.Commands = []cli.Command{

		{
			Name:  "login",
			Usage: "login gpud to lepton.ai (called automatically in gpud up with non-empty --token)",
			UsageText: `# to login gpud to lepton.ai with an existing, running gpud
sudo gpud login --token <LEPTON_AI_TOKEN>
`,
			Action: cmdLogin,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "token",
					Usage: "lepton.ai workspace token for checking in",
				},
				cli.StringFlag{
					Name:  "endpoint",
					Usage: "endpoint for control plane",
					Value: "mothership-machine-mothership-machine-dev.cloud.lepton.ai",
				},
			},
		},
		{
			Name:  "up",
			Usage: "initialize and start gpud in a daemon mode (systemd)",
			UsageText: `# to start gpud as a systemd unit (recommended)
sudo gpud up

# to enable machine monitoring powered by lepton.ai platform
# sign up here: https://lepton.ai
sudo gpud up --token <LEPTON_AI_TOKEN>

# to start gpud without a systemd unit (e.g., mac)
gpud run

# or
nohup sudo gpud run &>> <your log file path> &
`,
			Action: cmdUp,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "token",
					Usage: "lepton.ai workspace token for checking in",
				},
				cli.StringFlag{
					Name:  "endpoint",
					Usage: "endpoint for checking in",
					Value: "mothership-machine-mothership-machine-dev.cloud.lepton.ai",
				},
			},
		},
		{
			Name:  "down",
			Usage: "stop gpud systemd unit",
			UsageText: `# to stop the existing gpud systemd unit
sudo gpud down

# to uninstall gpud
sudo rm /usr/sbin/gpud
sudo rm /etc/systemd/system/gpud.service
`,
			Action: cmdDown,
		},
		{
			Name:   "run",
			Usage:  "starts gpud without any login/checkin ('gpud up' is recommended for linux)",
			Action: cmdRun,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:        "log-level,l",
					Usage:       "set the logging level [debug, info, warn, error, fatal, panic, dpanic]",
					Destination: &logLevel,
				},
				&cli.StringFlag{
					Name:        "listen-address",
					Usage:       "set the listen address",
					Destination: &listenAddress,
					Value:       fmt.Sprintf("0.0.0.0:%d", config.DefaultGPUdPort),
				},
				&cli.StringFlag{
					Name:        "annotations",
					Usage:       "set the annotations",
					Destination: &annotations,
				},
				cli.StringFlag{
					Name:        "uid",
					Usage:       "uid for this machine",
					Destination: &uid,
				},
				&cli.BoolFlag{
					Name:        "pprof",
					Usage:       "enable pprof (default: false)",
					Destination: &pprof,
				},
				&cli.DurationFlag{
					Name:        "retention-period",
					Usage:       "set the time period to retain metrics for (once elapsed, old records are compacted/purged)",
					Destination: &retentionPeriod,
					Value:       config.DefaultRetentionPeriod.Duration,
				},
				&cli.DurationFlag{
					Name:        "refresh-components-interval",
					Usage:       "set the time period to refresh selected components",
					Destination: &refreshComponentsInterval,
					Value:       config.DefaultRefreshComponentsInterval.Duration,
				},
				&cli.BoolTFlag{
					Name:        "web-enable",
					Usage:       "enable local web interface (default: true)",
					Destination: &webEnable,
				},
				&cli.BoolFlag{
					Name:        "web-admin",
					Usage:       "enable admin interface (default: false)",
					Destination: &webAdmin,
				},
				&cli.DurationFlag{
					Name:        "web-refresh-period",
					Usage:       "set the time period to refresh states/metrics",
					Destination: &webRefreshPeriod,
					Value:       time.Minute,
				},
				cli.StringFlag{
					Name:  "endpoint",
					Usage: "endpoint for control plane",
					Value: "mothership-machine-mothership-machine-dev.cloud.lepton.ai",
				},
				&cli.BoolTFlag{
					Name:        "enable-auto-update",
					Usage:       "enable auto update of gpud (default: true)",
					Destination: &enableAutoUpdate,
				},
				&cli.IntFlag{
					Name:        "auto-update-exit-code",
					Usage:       "specifies the exit code to exit with when auto updating (default: -1 to disable exit code)",
					Destination: &autoUpdateExitCode,
					Value:       -1,
				},
				&cli.StringSliceFlag{
					Name:  "files-to-check",
					Usage: "enable 'file' component that returns healthy if and only if all the files exist (default: [], use '--files-to-check=a --files-to-check=b' for multiple files)",
					Value: &filesToCheck,
				},
				&cli.BoolFlag{
					Name:        "docker-ignore-connection-errors",
					Usage:       "ignore connection errors to docker daemon, useful when docker daemon is not running (default: false)",
					Destination: &dockerIgnoreConnectionErrors,
				},
				&cli.BoolFlag{
					Name:        "kubelet-ignore-connection-errors",
					Usage:       "ignore connection errors to kubelet read-only port, useful when kubelet readOnlyPort is disabled (default: false)",
					Destination: &kubeletIgnoreConnectionErrors,
				},
			},
		},

		// operations
		{
			Name:      "update",
			Usage:     "update gpud",
			UsageText: "",
			Action:    cmdUpdate,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "url",
					Usage: "url for getting a package",
				},
				cli.StringFlag{
					Name:  "next-version",
					Usage: "set the next version",
				},
			},
			Subcommands: []cli.Command{
				{
					Name:   "check",
					Usage:  "check availability of new version gpud",
					Action: cmdUpdateCheck,
				},
			},
		},
		{
			Name:  "release",
			Usage: "release gpud",
			Subcommands: []cli.Command{
				{
					Name:   "gen-key",
					Usage:  "generate root or signing key pair",
					Action: cmdReleaseGenKey,
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "root (default: false)",
							Usage: "generate root key",
						},
						cli.BoolFlag{
							Name:  "signing (default: false)",
							Usage: "generate signing key",
						},
						cli.StringFlag{
							Name:  "priv-path",
							Usage: "path of private key",
						},
						cli.StringFlag{
							Name:  "pub-path",
							Usage: "path of public key",
						},
					},
				},
				{
					Name:   "sign-key",
					Usage:  "Sign signing keys with a root key",
					Action: cmdReleaseSignKey,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "root-priv-path",
							Usage: "path of root private key",
						},
						cli.StringFlag{
							Name:  "sign-pub-path",
							Usage: "path of signing public key",
						},
						cli.StringFlag{
							Name:  "sig-path",
							Usage: "output path of signature path",
						},
					},
				},
				{
					Name:   "verify-key-signature",
					Usage:  "Verify a root signture of the signing keys' bundle",
					Action: cmdReleaseVerifyKeySignature,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "root-pub-path",
							Usage: "path of root public key",
						},
						cli.StringFlag{
							Name:  "sign-pub-path",
							Usage: "path of signing public key",
						},
						cli.StringFlag{
							Name:  "sig-path",
							Usage: "path of signature path",
						},
					},
				},
				{
					Name:   "sign-package",
					Usage:  "Sign a package with a signing key",
					Action: cmdReleaseSignPackage,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "package-path",
							Usage: "path of package",
						},
						cli.StringFlag{
							Name:  "sign-priv-path",
							Usage: "path of signing private key",
						},
						cli.StringFlag{
							Name:  "sig-path",
							Usage: "output path of signature path",
						},
					},
				},
				{
					Name:   "verify-package-signature",
					Usage:  "Verify a package signture using a signing key",
					Action: cmdReleaseVerifyPackageSignature,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "package-path",
							Usage: "path of package",
						},
						cli.StringFlag{
							Name:  "sign-pub-path",
							Usage: "path of signing public key",
						},
						cli.StringFlag{
							Name:  "sig-path",
							Usage: "path of signature path",
						},
					},
				},
			},
		},

		// for checking gpud status
		{
			Name:    "status",
			Aliases: []string{"st"},

			Usage:  "checks the status of gpud",
			Action: cmdStatus,
		},
		{
			Name:    "logs",
			Aliases: []string{"l"},

			Usage:  "checks the gpud logs",
			Action: cmdLogs,
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:        "lines,n",
					Usage:       "set the number to tail logs",
					Destination: &tailLines,
					Value:       100,
				},
			},
		},

		{
			Name:    "accelerator",
			Aliases: []string{"a"},

			Usage:  "quick scans the currently installed accelerator",
			Action: cmdAccelerator,
		},

		// for diagnose + quick scanning
		{
			Name:    "diagnose",
			Aliases: []string{"d"},

			Usage: "collects diagnose information",
			UsageText: `# to collect diagnose information
sudo gpud diagnose

# check the auto-generated summary file
cat summary.txt
`,
			Action: cmdDiagnose,
			Flags: []cli.Flag{
				&cli.BoolTFlag{
					Name:        "create-archive (default: true)",
					Usage:       "create .tar archive of diagnose information",
					Destination: &createArchive,
				},
			},
		},
		{
			Name:    "scan",
			Aliases: []string{"s"},

			Usage:  "quick scans the host for any major issues",
			Action: cmdScan,
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:        "lines,n",
					Usage:       "set the number of lines to tail log files (e.g., /var/log/dmesg)",
					Destination: &tailLines,
					Value:       5000,
				},
				&cli.BoolFlag{
					Name:        "debug",
					Usage:       "enable debug mode (default: false)",
					Destination: &debug,
				},
				&cli.BoolFlag{
					Name:        "poll-xid-events",
					Usage:       "enable polling xid events (default: false)",
					Destination: &pollXidEvents,
				},
				&cli.BoolFlag{
					Name:        "poll-gpm-events",
					Usage:       "enable polling gpm events (default: false)",
					Destination: &pollGPMEvents,
				},
				&cli.BoolTFlag{
					Name:        "netcheck",
					Usage:       "enable network connectivity checks to global edge/derp servers (default: true)",
					Destination: &netcheck,
				},
			},
		},
	}

	return app
}
