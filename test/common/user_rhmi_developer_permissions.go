package common

import (
	goctx "context"
	"fmt"
	"time"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/go-querystring/query"
	"github.com/integr8ly/integreatly-operator/test/resources"
	projectv1 "github.com/openshift/api/project/v1"
	v1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	expectedRhmiDeveloperProjectCount       = 1
	expectedManagedApiDeveloperProjectCount = 0
)

// struct used to create query string for fuse logs endpoint
type LogOptions struct {
	Container string `url:"container"`
	Follow    string `url:"follow"`
	TailLines string `url:"tailLines"`
}

func TestRHMIDeveloperUserPermissions(t TestingTB, ctx *TestingContext) {
	testUser := fmt.Sprintf("%v%02d", DefaultTestUserName, 1)

	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	masterURL := rhmi.Spec.MasterURL
	t.Logf("retrieved console master URL %v", masterURL)

	// get oauth route
	oauthRoute := &v1.Route{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: resources.OpenshiftOAuthRouteName, Namespace: resources.OpenshiftAuthenticationNamespace}, oauthRoute); err != nil {
		t.Fatal("error getting Openshift Oauth Route: ", err)
	}
	t.Log("retrieved openshift-Oauth route")

	// get rhmi developer user tokens
	if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), testUser, DefaultPassword, ctx.HttpClient, TestingIDPRealm, t); err != nil {
		t.Fatalf("error occured trying to get token : %v", err)
	}
	t.Log("retrieved rhmi developer user tokens")
	openshiftClient := resources.NewOpenshiftClient(ctx.HttpClient, masterURL)

	fuseNamespace := fmt.Sprintf("%sfuse", NamespacePrefix)
	if !integreatlyv1alpha1.IsRHOAM(integreatlyv1alpha1.InstallationType(rhmi.Spec.Type)) {
		// test rhmi developer projects are as expected
		t.Log("testing rhmi 2 developer projects")
		if err := testRHMI2DeveloperProjects(masterURL, fuseNamespace, openshiftClient); err != nil {
			t.Fatalf("test failed - %v", err)
		}
	} else {
		// test managed api developer projects are as expected
		t.Log("testing managed api developer projects")
		if err := testManagedApiDeveloperProjects(masterURL, openshiftClient); err != nil {
			t.Fatalf("test failed - %v", err)
		}
	}

	if !integreatlyv1alpha1.IsRHOAM(integreatlyv1alpha1.InstallationType(rhmi.Spec.Type)) {
		// get fuse pods for rhmi developer
		podlist, err := openshiftClient.ListPods(fuseNamespace)
		if err != nil {
			t.Fatalf("error occured while getting pods : %v", err)
		}

		// log through rhmi developer fuse podlist
		for _, p := range podlist.Items {
			if p.Status.Phase == "Running" {
				logOpt := LogOptions{p.Spec.Containers[0].Name, "false", "10"}
				lv, err := query.Values(logOpt)
				if err != nil {
					t.Fatal(err)
				}
				// verify an rhmi developer can access the pods logs
				podPath := fmt.Sprintf(resources.OpenshiftPathListPods, p.Namespace)
				resp, err := openshiftClient.GetRequest(fmt.Sprintf("%s/%s/log?%s", podPath, p.Name, lv.Encode()))
				if err != nil {
					t.Fatalf("error occurred making oc get request: %v", err)
				}
				if resp.StatusCode != 200 {
					t.Fatalf("test-failed - status code %d RHMI developer unable to access fuse logs in pod %s : %v", resp.StatusCode, p.Name, err)
				}
			}
		}
	}

	// Verify RHMI Developer permissions around RHMI & RHMI Config
	verifyRHMIDeveloperRHMIConfigPermissions(t, openshiftClient)
	verifyRHMIDeveloperRHMIPermissions(t, openshiftClient)

	verifyRHMIDeveloper3ScaleRoutePermissions(t, openshiftClient)
}

// Verify that a dedicated admin can edit routes in the 3scale namespace
func verifyRHMIDeveloper3ScaleRoutePermissions(t TestingTB, client *resources.OpenshiftClient) {
	ns := NamespacePrefix + "3scale"
	route := "backend"

	path := fmt.Sprintf(resources.PathGetRoute, ns, route)
	resp, err := client.DoOpenshiftGetRequest(path)
	if err != nil {
		t.Errorf("Failed to get route : %s", err)
	}
	if resp.StatusCode != 403 {
		t.Errorf("RHMI Developer was incorrectly able to get route : %v", resp)
	}
}

func testRHMI2DeveloperProjects(masterURL, fuseNamespace string, openshiftClient *resources.OpenshiftClient) error {
	var rhmiDevfoundProjects *projectv1.ProjectList
	// five minute time out needed to ensure users have been reconciled by RHMI operator
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
		rhmiDevfoundProjects, err = openshiftClient.ListProjects()
		if err != nil {
			return false, fmt.Errorf("error occured while getting user projects : %w", err)
		}

		// check if projects are as expected for rhmi developer
		if len(rhmiDevfoundProjects.Items) != expectedRhmiDeveloperProjectCount {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		if rhmiDevfoundProjects != nil {
			return fmt.Errorf("unexpected developer project count : %d expected project count : %d , error occurred - %w", len(rhmiDevfoundProjects.Items), expectedRhmiDeveloperProjectCount, err)
		} else {
			return fmt.Errorf("unexpected error occurred when retrieving projects list - %w", err)
		}

	}

	foundNamespace := rhmiDevfoundProjects.Items[0].Name
	if foundNamespace != fuseNamespace {
		return fmt.Errorf("found rhmi developer project: %s expected rhmi developer project : %s", foundNamespace, fuseNamespace)
	}

	return nil
}

func testManagedApiDeveloperProjects(masterURL string, openshiftClient *resources.OpenshiftClient) error {
	var rhmiDevfoundProjects *projectv1.ProjectList
	// five minute time out needed to ensure users have been reconciled by RHMI operator
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
		rhmiDevfoundProjects, err = openshiftClient.ListProjects()
		if err != nil {
			return false, fmt.Errorf("error occured while getting user projects : %w", err)
		}

		// check if projects are as expected for rhmi developer
		if len(rhmiDevfoundProjects.Items) != expectedManagedApiDeveloperProjectCount {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		if rhmiDevfoundProjects != nil {
			return fmt.Errorf("unexpected developer project count : %d expected project count : %d , error occurred - %w", len(rhmiDevfoundProjects.Items), expectedManagedApiDeveloperProjectCount, err)
		} else {
			return fmt.Errorf("unexpected error occurred when retrieving projects list - %w", err)
		}

	}

	return nil
}

func verifyRHMIDeveloperRHMIConfigPermissions(t TestingTB, openshiftClient *resources.OpenshiftClient) {
	t.Log("Verifying RHMI Developer permissions for RHMIConfig Resource")

	expectedPermission := ExpectedPermissions{
		ExpectedCreateStatusCode: 403,
		ExpectedReadStatusCode:   403,
		ExpectedUpdateStatusCode: 403,
		ExpectedDeleteStatusCode: 403,
		ExpectedListStatusCode:   403,
		ListPath:                 fmt.Sprintf(resources.PathListRHMIConfig, RHOAMOperatorNamespace),
		GetPath:                  fmt.Sprintf(resources.PathGetRHMIConfig, RHOAMOperatorNamespace, "rhmi-config"),
		ObjectToCreate: &integreatlyv1alpha1.RHMIConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-rhmi-config",
				Namespace: RHOAMOperatorNamespace,
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1alpha1",
				Kind:       "RHMIConfig",
			},
		},
	}

	verifyCRUDLPermissions(t, openshiftClient, expectedPermission)
}

func verifyRHMIDeveloperRHMIPermissions(t TestingTB, openshiftClient *resources.OpenshiftClient) {
	t.Log("Verifying RHMI Developer permissions for RHMI Resource")

	expectedPermission := ExpectedPermissions{
		ExpectedCreateStatusCode: 403,
		ExpectedReadStatusCode:   403,
		ExpectedUpdateStatusCode: 403,
		ExpectedDeleteStatusCode: 403,
		ExpectedListStatusCode:   403,
		ListPath:                 fmt.Sprintf(resources.PathListRHMI, RHOAMOperatorNamespace),
		GetPath:                  fmt.Sprintf(resources.PathGetRHMI, RHOAMOperatorNamespace, "rhoam"),
		ObjectToCreate: &integreatlyv1alpha1.RHMI{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-rhoam",
				Namespace: RHOAMOperatorNamespace,
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1alpha1",
				Kind:       "RHMI",
			},
		},
	}

	verifyCRUDLPermissions(t, openshiftClient, expectedPermission)
}
