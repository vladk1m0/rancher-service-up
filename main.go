package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"rancher-service-up/rancher"
	"runtime"
	"strings"
)

const (
	appName    = "Rancher service upgrade tool"
	appVersion = "1.0.0"
)

var (
	goVersion = runtime.Version()
)

func initLogger(flog *log.Logger, logfile string) {
	// Verbose logging with file name and line number
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func usage() {
	fmt.Printf("\n%s version: %s\n", appName, appVersion)
	fmt.Printf("Usage: rancher-service-up [OPTIONS]\n\n")
	flag.PrintDefaults()
	os.Exit(0)
}

func panicExit() {
	if r := recover(); r != nil {
		fmt.Printf("error: %s\n", r)
		os.Exit(1)
	}
}

func ensureOptionValue(opt *string, envVar string, msg string) {
	if strings.TrimSpace(*opt) == "" || strings.Contains(*opt, envVar) {
		*opt = os.Getenv(envVar)
		if strings.TrimSpace(*opt) == "" {
			fmt.Printf("\n%s!\n", msg)
			usage()
		}
	}
}

func main() {
	defer panicExit()

	opts := struct {
		debug                *bool
		version              *bool
		url                  *string
		key                  *string
		secret               *string
		env                  *string
		stack                *string
		service              *string
		image                *string
		startFirst           *bool
		batchSize            *int
		batchInterval        *int
		upgradeTimeout       *int
		waiteUpgradeFinish   *bool
		noWaiteUpgradeFinish *bool
		finishUpgrade        *bool
		noFinishUpgrade      *bool
		upgradeSidekicks     *bool
		newSidekickImage     *rancher.SidekickImageParams
	}{}

	opts.debug = flag.Bool("debug", false, "Enable debug mode.")
	opts.version = flag.Bool("version", false, "Display app version.")

	opts.url = flag.String("url", "RANCHER_URL", "The URL for your Rancher server, eg: http://rancher:8080. (Required!)")
	opts.key = flag.String("key", "RANCHER_ACCESS_KEY", "The environment or account API key. (Required!)")
	opts.secret = flag.String("secret", "RANCHER_SECRET_KEY", "The secret for the access API key. (Required!)")
	opts.env = flag.String("env", "Default", "The name of the environment in Rancher.")
	opts.stack = flag.String("stack", "CI_PROJECT_NAMESPACE", "The name of the stack in Rancher. (Required!)")
	opts.service = flag.String("service", "CI_PROJECT_NAME", "The name of the service in Rancher to upgrade. (Required!)")
	opts.image = flag.String("image", "", "The new image[:tag] of the service in Rancher to upgrade. (Required!)")

	opts.startFirst = flag.Bool("start-first", false, "Should Rancher start new containers before stopping the old ones.")
	opts.batchSize = flag.Int("batch-size", 1, "Number of containers to upgrade at once.")
	opts.batchInterval = flag.Int("batch-interval", 2, "Number of seconds to wait between upgrade batches.")
	opts.upgradeTimeout = flag.Int("upgrade-timeout", 3*60, "How long to wait, in seconds, for the upgrade to finish before exiting.")
	opts.waiteUpgradeFinish = flag.Bool("wait-for-upgrade-to-finish", true, "Wait for Rancher to finish the upgrade before this tool exits.")
	opts.finishUpgrade = flag.Bool("finish-upgrade", true, "Mark the upgrade as finished after it completes.")

	opts.upgradeSidekicks = flag.Bool("upgrade-sidekicks", false, "Upgrade service sidekicks at the same time.")
	opts.newSidekickImage = new(rancher.SidekickImageParams)
	flag.Var(opts.newSidekickImage, "new-sidekick-image", "If specified, replace the sidekick image[:tag] with this one during the upgrade.")

	flag.Usage = usage
	flag.Parse()

	if *opts.version {
		fmt.Printf("\n%s version: %s\n", appName, appVersion)
		fmt.Printf("Usage: rancher-service-up [OPTIONS]\n\n")
		os.Exit(0)
	}

	ensureOptionValue(opts.url, "RANCHER_URL", "Required Rancher URL")
	ensureOptionValue(opts.key, "RANCHER_ACCESS_KEY", "Required Rancher access key")
	ensureOptionValue(opts.secret, "RANCHER_SECRET_KEY", "Required Rancher access secret")
	ensureOptionValue(opts.env, "", "Required Rancher environment name")
	ensureOptionValue(opts.stack, "CI_PROJECT_NAMESPACE", "Required Rancher stack name")
	ensureOptionValue(opts.service, "CI_PROJECT_NAME", "Required Rancher service name")

	api, err := rancher.NewClient(*opts.debug, *opts.url, *opts.key, *opts.secret)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Find the environment id in Rancher
	env, err := api.GetEnv(*opts.env)
	if err != nil {
		log.Fatal(err.Error())
	}
	if *opts.debug {
		log.Printf("Environment ID = [%s], Name = [%s]\n", env.ID, env.Name)
	}

	// Find the stack in the environment
	stack, err := api.GetStack(env.ID, *opts.stack)
	if err != nil {
		log.Fatal(err.Error())
	}
	if *opts.debug {
		log.Printf("Stack ID = [%s], Name = [%s]\n", stack.ID, stack.Name)
	}

	// Find the service in the stack
	service, err := api.GetService(env.ID, stack.ID, *opts.service)
	if err != nil {
		log.Fatal(err.Error())
	}
	if *opts.debug {
		log.Printf("Service ID = [%s], Name = [%s]\n", service.ID, service.Name)
	}

	// Is the service eligible for upgrade?
	if service.State == "upgraded" {
		err := api.FinishUpgrade(service)
		if err != nil {
			log.Fatal(err.Error())
		}

		err = api.WaitForServiceState(service, *opts.upgradeTimeout, "active")
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	// Start the upgrade
	if *opts.debug {
		log.Printf("Upgrading %s/%s in environment %s.\n", stack.Name, service.Name, env.Name)
	}

	req, err := api.NewUpgradeRequest(
		service,
		*opts.batchSize,
		*opts.batchInterval,
		*opts.startFirst,
		*opts.image,
		*opts.upgradeSidekicks,
		opts.newSidekickImage,
	)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = api.UpgradeService(service, req)
	if err != nil {
		log.Fatal(err.Error())
	}

	if *opts.waiteUpgradeFinish {
		err := api.WaitForServiceState(service, *opts.upgradeTimeout, "upgraded")
		if err != nil {
			log.Println(err.Error())
		}

		// Get current service state while upgrade
		state, err := api.GetServiceStatus(service)
		if err != nil {
			log.Fatal(err.Error())
		}

		// If error while upgrade
		if state == "unhealthy" {
			err = api.RollbackUpgrade(service)
			if err != nil {
				log.Fatal(err.Error())
			}

			err := api.WaitForServiceState(service, *opts.upgradeTimeout, "active")
			if err != nil {
				log.Fatal(err.Error())
			}

			log.Println("Upgrade rollback :(")

		} else if *opts.finishUpgrade {
			err = api.FinishUpgrade(service)
			if err != nil {
				log.Fatal(err.Error())
			}

			err = api.WaitForServiceState(service, *opts.upgradeTimeout, "active")
			if err != nil {
				log.Fatal(err.Error())
			}

			log.Println("Upgrade succeed :)")
		} else {
			log.Println("Upgrade succeed :)")
		}
	} else {
		log.Println("Upgrade succeed :)")
	}
}
