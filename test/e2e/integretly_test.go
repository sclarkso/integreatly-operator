package e2e

import (
	"fmt"
	"os"

	"github.com/integr8ly/integreatly-operator/test/common"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("integreatly", func() {

	var (
		restConfig = cfg
		t          = GinkgoT()
	)

	BeforeEach(func() {
		restConfig = cfg
		t = GinkgoT()
	})

	JustBeforeEach(func() {
		if err := common.WaitForRHMIStageToComplete(t, restConfig); err != nil {
			t.Error(err)
		}
	})

	RunTests := func() {

		tests := []common.Tests{
			{
				Type:      fmt.Sprintf("%s ALL TESTS", installType),
				TestCases: common.GetAllTestCases(installType),
			},
			{
				Type:      fmt.Sprintf("%s HAPPY PATH", installType),
				TestCases: common.GetHappyPathTestCases(installType),
			},
			{
				Type:      "FAILURE TESTS",
				TestCases: common.FAILURE_TESTS,
			},
		}

		//Function to be used when Threescale becomes cluster scoped.
		// clusterScoped, err := common.IsClusterScoped(restConfig)
		// if err != nil {
		// 	t.Error(err)
		// }
		// if clusterScoped {
		// 	tests = append(tests, common.Tests{
		// 		Type:      fmt.Sprintf("%s Threescale Cluster Scoped", installType),
		// 		TestCases: common.GetClusterScopedTestCases(installType),
		// 	})
		// }

		if os.Getenv("DESTRUCTIVE") == "true" {
			tests = append(tests, common.Tests{
				Type:      "Destructive Tests",
				TestCases: common.DESTRUCTIVE_TESTS,
			})
		}

		for _, test := range tests {
			Context(test.Type, func() {
				for _, testCase := range test.TestCases {
					currentTest := testCase
					It(currentTest.Description, func() {
						testingContext, err := common.NewTestingContext(restConfig)
						if err != nil {
							t.Fatal("failed to create testing context", err)
						}
						currentTest.Test(t, testingContext)
					})
				}
			})
		}

	}

	RunTests()

})
