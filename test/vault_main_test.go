package test

import (
	"fmt"
	"testing"

	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/packer"
	"github.com/gruntwork-io/terratest/modules/test-structure"
)

const AMI_EXAMPLE_PATH = "../examples/vault-consul-ami/vault-consul.json"

type testCase struct {
	Name       string                           // Name of the test
	Func       func(*testing.T, string, string) // Function that runs test. Receives(t, amiId, sshUserName)
	Enterprise bool                             // Run on ami with enterprise vault installed
}

var testCases = []testCase{
	{
		"TestVaultAutoUnseal",
		runVaultAutoUnsealTest,
		true,
	},
	{
		"TestEnterpriseInstallation",
		runVaultEnterpriseClusterTest,
		true,
	},
	{
		"TestVaultEC2Auth",
		runVaultEC2AuthTest,
		false,
	},
	{
		"TestVaultIAMAuth",
		runVaultIAMAuthTest,
		false,
	},
	{
		"TestVaultWithS3Backend",
		runVaultWithS3BackendClusterTest,
		false,
	},
	{
		"TestVaultPrivateCluster",
		runVaultPrivateClusterTest,
		false,
	},
	{
		"TestVaultPublicCluster",
		runVaultPublicClusterTest,
		false,
	},
}

func TestMainVaultCluster(t *testing.T) {
	t.Parallel()

	amiIds := map[string]string{}

	test_structure.RunTestStage(t, "setup_amis", func() {
		//awsRegion := aws.GetRandomRegion(t, nil, nil)
		awsRegion := "eu-west-1"
		test_structure.SaveString(t, WORK_DIR, "awsRegion", awsRegion)

		tlsCert := generateSelfSignedTlsCert(t)
		saveTlsCert(t, WORK_DIR, tlsCert)

		var identifierToOptions = map[string]*packer.Options{
			"vaultEnterpriseUbuntu":      composeAmiOptions(t, AMI_EXAMPLE_PATH, "ubuntu16-ami", tlsCert, awsRegion, getUrlFromEnv(t)),
			"vaultEnterpriseAmazonLinux": composeAmiOptions(t, AMI_EXAMPLE_PATH, "amazon-linux-ami", tlsCert, awsRegion, getUrlFromEnv(t)),
			"vaultUbuntu":                composeAmiOptions(t, AMI_EXAMPLE_PATH, "ubuntu16-ami", tlsCert, awsRegion, ""),
			"vaultAmazonLinux":           composeAmiOptions(t, AMI_EXAMPLE_PATH, "amazon-linux-ami", tlsCert, awsRegion, ""),
		}

		amiIds = packer.BuildArtifacts(t, identifierToOptions)
		for key, amiId := range amiIds {
			test_structure.SaveString(t, WORK_DIR, fmt.Sprintf("amiId-%s", key), amiId)
		}
	})

	defer test_structure.RunTestStage(t, "delete_amis", func() {
		awsRegion := test_structure.LoadString(t, WORK_DIR, "awsRegion")
		for _, amiId := range amiIds {
			aws.DeleteAmi(t, awsRegion, amiId)
		}
		tlsCert := loadTlsCert(t, WORK_DIR)
		cleanupTlsCertFiles(tlsCert)
	})

	t.Run("group", func(t *testing.T) {
		runTestsOnDifferentPlatforms(t, testCases, amiIds)
	})

}

func runTestsOnDifferentPlatforms(t *testing.T, testCases []testCase, amiIds map[string]string) {
	t.Parallel()
	var amiId string
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(fmt.Sprintf("%sWithUbuntuAmi", testCase.Name), func(t *testing.T) {
			t.Parallel()
			if amiId = amiIds["vaultUbuntu"]; testCase.Enterprise {
				amiId = amiIds["vaultEnterpriseUbuntu"]
			}
			testCase.Func(t, amiId, "ubuntu")
		})
		t.Run(fmt.Sprintf("%sWithAmazonLinuxAmi", testCase.Name), func(t *testing.T) {
			t.Parallel()
			if amiId = amiIds["vaultAmazonLinux"]; testCase.Enterprise {
				amiId = amiIds["vaultEnterpriseAmazonLinux"]
			}
			testCase.Func(t, amiId, "ec2-user")
		})
	}
}
