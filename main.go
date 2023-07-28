package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/drone/drone-go/drone"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

func main() {
	config := new(oauth2.Config)
	auther := config.Client(
		context.TODO(),
		&oauth2.Token{
			AccessToken: os.Getenv("DRONE_TOKEN"),
		},
	)
	// create the drone client with authenticator
	client := drone.NewClient(os.Getenv("DRONE_SERVER"), auther)

	builds, err := client.BuildList("rancher", "rancher", drone.ListOptions{
		Page: 1,
		Size: 100,
	})
	if err != nil {
		logrus.Fatal(err)
	}

	failedBuilds := make([]*drone.Build, 0)
	for _, build := range builds {
		if build.Status == "failure" {
			failedBuilds = append(failedBuilds, build)
		}
	}

	for _, fb := range failedBuilds {
		fb, err := client.Build("rancher", "rancher", int(fb.Number))
		if err != nil {
			logrus.Fatal(err)
		}

		failedStages := make([]*drone.Stage, 0)
		for _, stage := range fb.Stages {
			if stage.Status == "failure" {
				failedStages = append(failedStages, stage)
			}
		}

		for _, stage := range failedStages {
			for _, step := range stage.Steps {
				if step.Status == "failure" {
					found := false
					logs, err := client.Logs("rancher", "rancher", int(fb.Number), stage.Number, step.Number)
					if err != nil {
						logrus.Fatal(err)
					}

					for i, line := range logs {
						if strings.Contains(line.Message, "undefined: ") {
							found = true
							logrus.Printf("Drone build %d failed due to bad code: %s", fb.Number, line.Message)
							break
						}
						if strings.Contains(line.Message, "The command 'powershell") {
							found = true
							logrus.Printf("Drone build %d failed due to powershell command: %s", fb.Number, line.Message)
							break
						}
						if strings.Contains(line.Message, "File is not `gofmt`-ed (gofmt)") {
							found = true
							logrus.Printf("Drone build %d failed due to gofmt: %s", fb.Number, line.Message)
							break
						}
						if strings.Contains(line.Message, "fatal: unable to access 'https://github.com/rancher/rancher.git/': OpenSSL SSL_connect: Connection reset by peer in connection to github.com:443") {
							found = true
							logrus.Printf("Drone build %d failed due to github clone error: %s", fb.Number, line.Message)
							break
						}
						if strings.Contains(line.Message, "Failed to connect to get.helm.sh") {
							found = true
							logrus.Printf("Drone build %d failed due to helm mirror unreachable: %s", fb.Number, line.Message)
							break
						}
						if strings.Contains(line.Message, "not something we can merge") {
							found = true
							logrus.Printf("Drone build %d failed due to unmergeable code: %s", fb.Number, line.Message)
							break
						}
						if strings.Contains(line.Message, "Encountered dirty repo! Aborting.") {
							found = true
							logrus.Printf("Drone build %d failed due to dirty repo: %s", fb.Number, line.Message)
							break
						}
						if strings.Contains(line.Message, "The scan has revealed issues. Number of issues found") {
							found = true
							logrus.Printf("Drone build %d failed due to fossa: %s", fb.Number, line.Message)
							break
						}
						if strings.Contains(line.Message, "not all images have a source code origin defined. Please provide origin URL's within rancher/pkg/image/origins.go for the following images") {
							found = true
							logrus.Printf("Drone build %d failed due to %s", fb.Number, line.Message)
							break
						}
						if strings.HasPrefix(line.Message, "./scripts/provisioning-tests: line 13:") && strings.HasSuffix(line.Message, "run_rancher") {
							found = true
							logrus.Printf("Drone build %d failed due to %s", fb.Number, line.Message)
							break
						}
						if strings.Contains(line.Message, "error creating and importing a k3d cluster:") {
							found = true
							logrus.Printf("Drone build %d failed due to %s", fb.Number, line.Message)
							break
						}
						if strings.HasPrefix(line.Message, "The command '/bin/sh -c mkdir -p /var/lib/rancher-data/local-catalogs/system-library &&") &&
							strings.Contains(line.Message, "returned a non-zero code: ") {
							found = true
							logrus.Printf("Drone build %d failed due to %s", fb.Number, line.Message)
							break
						}
						if strings.Contains(line.Message, `The command '/bin/sh -c zypper`) {
							found = true
							logrus.Printf("Drone build %d failed due to %s", fb.Number, line.Message)
							break
						}
						if strings.Contains(line.Message, `ERROR: process "/bin/sh -c zypper`) {
							found = true
							logrus.Printf("Drone build %d failed due to %s", fb.Number, line.Message)
							break
						}
						if strings.Contains(line.Message, "--- FAIL:") {
							found = true
							logrus.Printf("Drone build %d failed due to %s", fb.Number, line.Message)
							// do not break, may be more logs
						}
						if strings.Contains(line.Message, "rancher-webhook was not available after") {
							if strings.Contains(logs[i+1].Message, "Stopping rancher server") {
								found = true
								logrus.Printf("Drone build %d failed due to webhook failure", fb.Number)
								break
							}
						}
						if strings.Contains(line.Message, "=================================== FAILURES ===================================") {
							found = true
							logrus.Printf("Drone build %d failed due to %s", fb.Number, logs[i+1].Message)
							break
						}
						if strings.Contains(line.Message, "==================================== ERRORS ====================================") {
							found = true
							logrus.Printf("Drone build %d failed due to %s", fb.Number, logs[i+1].Message)
							break
						}
						if strings.Contains(line.Message, "ERROR test_alert.py::test_alert_access") {
							logrus.Printf("Drone build %d failed due to %s", fb.Number, line.Message)
							found = true
							break
						}
						if strings.Contains(line.Message, "FAIL: Test_Provisioning_MP_ThreeEtcdNodesScaledDownThenDelete") {
							logrus.Printf("Drone build %d failed due to %s", fb.Number, line.Message)
							found = true
							break
						}
						if strings.Contains(line.Message, "FAIL: Test_Operation_Custom_EtcdSnapshotOperationsOnNewNode") {
							logrus.Printf("Drone build %d failed due to %s", fb.Number, line.Message)
							found = true
							break
						}
						if strings.Contains(line.Message, "throw \"EXITED\"") {
							logrus.Printf("Drone build %d failed due to %s", fb.Number, line.Message)
							found = true
							break
						}
					}

					if !found {
						fmt.Printf("Drone build %d failed with unknown error. Dumping logs:\n", fb.Number)
						for _, line := range logs {
							fmt.Print(line.Message)
						}
						panic(1)
					}
				}
			}
		}

	}
}
