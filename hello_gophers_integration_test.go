package test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
)

const dbDirStage = "../terraoneclone/live/stage/data-stores/mysql"
const appDirStage = "../terraone"

func TestHelloGophersAppStage(t *testing.T) {
	t.Parallel()

	// Deploy the MySQL DB
	dbOpts := createDbOpts(t, dbDirStage)
	defer terraform.Destroy(t, dbOpts)
	terraform.InitAndApply(t, dbOpts)

	// Deploy the Hello Gophers app
	helloOpts := createHelloOpts(dbOpts, appDirStage)
	defer terraform.Destroy(t, helloOpts)
	terraform.InitAndApply(t, helloOpts)

	// Validate working of Hello Gophers app
	validateHelloApp(t, helloOpts)
}

func createDbOpts(t *testing.T, terraformDir string) *terraform.Options {
	uniqueId := random.UniqueId()
	// Declare test-friendly varibales for the backend
	testBucket := "batoto-bitange"
	testRegion := "us-east-1"
	dbStateKey := fmt.Sprintf("%s/%s/terraform.tfstate", t.Name(), uniqueId)

	// Point terraform.Options at the passed in directory
	return &terraform.Options{
		TerraformDir: terraformDir,
		// Set the db credentials variables
		Vars: map[string]interface{}{
			"db_name": fmt.Sprintf("test%s", uniqueId),
			// "db_username": "gophercon",
			// "db_password": "africa",
		},
		// Set the backend config variables
		BackendConfig: map[string]interface{}{
			"bucket":  testBucket,
			"region":  testRegion,
			"key":     dbStateKey,
			"encrypt": true,
		},
		// Reconfigure backend
		Reconfigure: true,
		Upgrade:     true,
	}
}

func createHelloOpts(dbOpts *terraform.Options, terraformDir string) *terraform.Options {
	return &terraform.Options{
		TerraformDir: terraformDir,

		Vars: map[string]interface{}{
			"db_remote_state_bucket": dbOpts.BackendConfig["bucket"],
			"db_remote_state_key":    dbOpts.BackendConfig["key"],
			"environment":            dbOpts.Vars["db_name"],
		},
		// Reconfigure backend
		Reconfigure: true,
	}
}

func validateHelloApp(t *testing.T, helloOpts *terraform.Options) {
	albDnsName := terraform.OutputRequired(t, helloOpts, "alb_dns_name")
	url := fmt.Sprintf("http://%s", albDnsName)

	maxRetries := 10
	timeBetweenRetries := 10 * time.Second

	http_helper.HttpGetWithRetryWithCustomValidation(
		t,
		url,
		nil,
		maxRetries,
		timeBetweenRetries,
		func(status int, body string) bool {
			return status == 200 && strings.Contains(body, "Hello, World")
		},
	)
}
