package resources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"testing"
	"time"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	oauthv1 "github.com/openshift/api/oauth/v1"

	projectv1 "github.com/openshift/api/project/v1"
	alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadFuseFunc: func() (ready *config.Fuse, e error) {
			return config.NewFuse(config.ProductConfig{}), nil
		},
		ReadRHSSOFunc: func() (*config.RHSSO, error) {
			return config.NewRHSSO(config.ProductConfig{
				"NAMESPACE": "fuse",
				"URL":       "fuse.openshift-cluster.com",
			}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
	}
}

func buildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := alpha1.AddToScheme(scheme)
	err = oauthv1.AddToScheme(scheme)
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	err = projectv1.AddToScheme(scheme)
	return scheme, err
}

func TestNewReconciler_ReconcileSubscription(t *testing.T) {
	scheme, err := buildScheme()
	if err != nil {
		t.Fatalf("error creating scheme: %s", err.Error())
	}
	ownerInstall := &integreatlyv1alpha1.RHMI{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
	}

	cases := []struct {
		Name             string
		FakeMPM          marketplace.MarketplaceInterface
		client           k8sclient.Client
		SubscriptionName string
		ExpectErr        bool
		ExpectedStatus   integreatlyv1alpha1.StatusPhase
		Installation     *integreatlyv1alpha1.RHMI
		Target           marketplace.Target
		Validate         func(t *testing.T, mock *marketplace.MarketplaceInterfaceMock)
		Assertion        func(k8sclient.Client) error
	}{
		{
			Name: "test reconcile subscription creates a new subscription  completes successfully ",
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {

					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plan *alpha1.InstallPlan, subscription *alpha1.Subscription, e error) {
					return &alpha1.InstallPlan{Status: alpha1.InstallPlanStatus{Phase: alpha1.InstallPlanPhaseComplete}}, &alpha1.Subscription{}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseCompleted,
			Installation:     &integreatlyv1alpha1.RHMI{},
			Validate: func(t *testing.T, mock *marketplace.MarketplaceInterfaceMock) {
				if len(mock.InstallOperatorCalls()) != 1 {
					t.Fatalf("expected create subscription to be called once but was called %v", len(mock.InstallOperatorCalls()))
				}
				if len(mock.GetSubscriptionInstallPlanCalls()) != 1 {
					t.Fatalf("expected GetSubscriptionInstallPlanCalls to be called once but was called %v", len(mock.GetSubscriptionInstallPlanCalls()))
				}
			},
		},
		{
			Name:   "test reconcile subscription recreates subscription when installation plan not found completes successfully ",
			client: fakeclient.NewFakeClientWithScheme(scheme),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {

					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *alpha1.InstallPlan, subscription *alpha1.Subscription, e error) {
					return nil, &alpha1.Subscription{ObjectMeta: metav1.ObjectMeta{
						// simulate the time has passed
						CreationTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
					}}, k8serr.NewNotFound(alpha1.Resource("installplan"), "my-install-plan")
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseAwaitingOperator,
		},
		{
			Name: "test reconcile subscription returns waiting for operator when catalog source config not ready",
			client: fakeclient.NewFakeClientWithScheme(scheme, &alpha1.CatalogSourceList{
				Items: []alpha1.CatalogSource{
					alpha1.CatalogSource{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "test-ns",
						},
					},
				},
			}),
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseFailed,
			FakeMPM:          marketplace.NewManager(),
			Installation:     ownerInstall,
			ExpectErr:        true,
		},
		{
			Name: "test reconcile subscription returns phase in progress if there is an install plan approved but not completed or failed",
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *alpha1.InstallPlan, subscription *alpha1.Subscription, e error) {
					return &alpha1.InstallPlan{
						Spec: alpha1.InstallPlanSpec{Approved: true},
						Status: alpha1.InstallPlanStatus{
							Phase: alpha1.InstallPlanPhaseInstalling,
						},
					}, &alpha1.Subscription{}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseInProgress,
			Installation:     &integreatlyv1alpha1.RHMI{},
		},
		{
			Name: "test reconcile subscription returns phase failed if unable to retrieve install plans",
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *alpha1.InstallPlan, subscription *alpha1.Subscription, e error) {
					return nil, nil, fmt.Errorf("simulate error gettiing install plans")
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseFailed,
			Installation:     &integreatlyv1alpha1.RHMI{},
			ExpectErr:        true,
		},
		{
			Name: "test reconcile subscription returns phase in progress if there are no install plans for subscription",
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *alpha1.InstallPlan, subscription *alpha1.Subscription, e error) {
					return nil, &alpha1.Subscription{}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseInProgress,
			Installation:     &integreatlyv1alpha1.RHMI{},
		},
		{
			Name: "test reconcile subscription returns phase failed if unable to delete subscription due for re-install ",
			client: &moqclient.SigsClientInterfaceMock{DeleteFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.DeleteOption) error {
				return fmt.Errorf("some error")
			}},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *alpha1.InstallPlan, subscription *alpha1.Subscription, e error) {
					return &alpha1.InstallPlan{
						Status: alpha1.InstallPlanStatus{Phase: alpha1.InstallPlanPhaseFailed},
					}, &alpha1.Subscription{}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseFailed,
			Installation:     &integreatlyv1alpha1.RHMI{},
			ExpectErr:        true,
		},
		{
			Name: "test reconcile subscription returns phase failed if unable to delete csv for re-install ",
			client: &moqclient.SigsClientInterfaceMock{DeleteFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.DeleteOption) error {
				return fmt.Errorf("some error")
			}},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *alpha1.InstallPlan, subscription *alpha1.Subscription, e error) {
					return &alpha1.InstallPlan{
						Status: alpha1.InstallPlanStatus{Phase: alpha1.InstallPlanPhaseFailed},
					}, &alpha1.Subscription{Status: alpha1.SubscriptionStatus{InstalledCSV: "test-csv"}}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseFailed,
			Installation:     &integreatlyv1alpha1.RHMI{},
			ExpectErr:        true,
		},
		{
			Name:   "test reconcile subscription returns phase awaiting operator after successful delete of failed install plan and csv",
			client: fakeclient.NewFakeClientWithScheme(scheme, &alpha1.ClusterServiceVersion{ObjectMeta: metav1.ObjectMeta{Name: "test-csv", Namespace: "test-ns"}}, &alpha1.Subscription{Status: alpha1.SubscriptionStatus{InstalledCSV: "test-csv"}}),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *alpha1.InstallPlan, subscription *alpha1.Subscription, e error) {
					return &alpha1.InstallPlan{
						Status: alpha1.InstallPlanStatus{Phase: alpha1.InstallPlanPhaseFailed},
					}, &alpha1.Subscription{Status: alpha1.SubscriptionStatus{InstalledCSV: "test-csv"}}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseAwaitingOperator,
			Installation:     &integreatlyv1alpha1.RHMI{},
		},
		{
			Name: "test reconcile subscription deletes CSV and subscription if the CSV doesn't have a deployment",
			client: fakeclient.NewFakeClientWithScheme(scheme,
				&alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-csv",
						Namespace: "test-ns",
					},
					Spec: alpha1.ClusterServiceVersionSpec{
						InstallStrategy: alpha1.NamedInstallStrategy{
							StrategyName: alpha1.InstallStrategyNameDeployment,
							StrategySpec: alpha1.StrategyDetailsDeployment{
								DeploymentSpecs: []alpha1.StrategyDeploymentSpec{},
							},
						},
					},
				},
				&alpha1.Subscription{
					Status: alpha1.SubscriptionStatus{
						InstalledCSV: "test-csv",
					},
				},
			),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName, ns string) (*alpha1.InstallPlan, *alpha1.Subscription, error) {
					return &alpha1.InstallPlan{
						Spec: alpha1.InstallPlanSpec{
							Approved: true,
							ClusterServiceVersionNames: []string{
								"test-csv",
							},
						},
						Status: alpha1.InstallPlanStatus{Phase: alpha1.InstallPlanPhaseComplete},
					}, &alpha1.Subscription{Status: alpha1.SubscriptionStatus{InstalledCSV: "test-csv"}}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseAwaitingOperator,
			Installation:     &integreatlyv1alpha1.RHMI{},
			Assertion: func(client k8sclient.Client) error {
				csv := &alpha1.ClusterServiceVersion{}
				err := client.Get(context.TODO(), k8sclient.ObjectKey{
					Name:      "test-csv",
					Namespace: "test-ns",
				}, csv)
				if err == nil || !k8serr.IsNotFound(err) {
					return errors.New("CSV was not deleted")
				}

				sub := &alpha1.Subscription{}
				err = client.Get(context.TODO(), k8sclient.ObjectKey{
					Name:      "something",
					Namespace: "test-ns",
				}, sub)
				if err == nil || !k8serr.IsNotFound(err) {
					return errors.New("Susbcription was not deleted")
				}

				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler := NewReconciler(
				tc.FakeMPM,
			)

			testNamespace := "test-ns"
			manifestsDirectory := "fakemanifestsdirectory"
			cfgMapCsReconciler := marketplace.NewConfigMapCatalogSourceReconciler(manifestsDirectory, tc.client, testNamespace, marketplace.CatalogSourceName)
			status, err := reconciler.ReconcileSubscription(context.TODO(), marketplace.Target{Namespace: testNamespace, Channel: "integreatly", SubscriptionName: tc.SubscriptionName, Package: tc.SubscriptionName}, []string{testNamespace}, backup.NewNoopBackupExecutor(), tc.client, cfgMapCsReconciler, getLogger())
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("expected no error but got one ", err)
			}
			if tc.ExpectedStatus != status {
				t.Fatal("expected phase ", tc.ExpectedStatus, " but got ", status)
			}
			if tc.Validate != nil {
				tc.Validate(t, tc.FakeMPM.(*marketplace.MarketplaceInterfaceMock))
			}
			if tc.Assertion != nil {
				if err := tc.Assertion(tc.client); err != nil {
					t.Errorf("failed assertion: %v", err)
				}
			}
		})
	}
}

func TestReconciler_reconcilePullSecret(t *testing.T) {
	scheme, err := buildScheme()
	if err != nil {
		t.Fatalf("error building scheme: %s", err.Error())
	}

	defPullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      integreatlyv1alpha1.DefaultOriginPullSecretName,
			Namespace: integreatlyv1alpha1.DefaultOriginPullSecretNamespace,
		},
		Data: map[string][]byte{
			"test": {'t', 'e', 's', 't'},
		},
	}

	customPullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Data: map[string][]byte{
			"test": {'t', 'e', 's', 't'},
		},
	}

	cases := []struct {
		Name         string
		Client       k8sclient.Client
		Installation *integreatlyv1alpha1.RHMI
		Config       *config.ConfigReadWriterMock
		Validate     func(c k8sclient.Client) error
	}{
		{
			Name:   "test pull secret is reconciled successfully",
			Client: fakeclient.NewFakeClientWithScheme(scheme, defPullSecret, customPullSecret),
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testinstall",
					Namespace: "testinstall",
				},
				Spec: integreatlyv1alpha1.RHMISpec{
					PullSecret: integreatlyv1alpha1.PullSecretSpec{
						Name:      "test",
						Namespace: "test",
					},
				},
			},
			Config: basicConfigMock(),
			Validate: func(c k8sclient.Client) error {
				s := &corev1.Secret{}
				err := c.Get(context.TODO(), k8sclient.ObjectKey{Name: "test", Namespace: "test"}, s)
				if err != nil {
					return err
				}
				if bytes.Compare(s.Data["test"], customPullSecret.Data["test"]) != 0 {
					return fmt.Errorf("expected data %v, but got %v", customPullSecret.Data["test"], s.Data["test"])
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			testReconciler := NewReconciler(nil)
			_, err := testReconciler.ReconcilePullSecret(context.TODO(), "test", "new-pull-secret-name", tc.Installation, tc.Client)
			if err != nil {
				t.Fatal("failed to run pull secret reconcile: ", err)
			}
			if err = tc.Validate(tc.Client); err != nil {
				t.Fatal("test validation failed: ", err)
			}
		})
	}
}

func TestReconciler_ReconcileOauthClient(t *testing.T) {
	scheme, err := buildScheme()
	if err != nil {
		t.Fatalf("error building scheme: %s", err.Error())
	}
	existingClient := &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		GrantMethod:  oauthv1.GrantHandlerAuto,
		Secret:       "test",
		RedirectURIs: []string{"http://test.com"},
	}
	cases := []struct {
		Name           string
		OauthClient    *oauthv1.OAuthClient
		ExpectErr      bool
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		client         k8sclient.Client
		Installation   *integreatlyv1alpha1.RHMI
	}{
		{
			Name: "test oauth client is reconciled correctly when it does not exist",
			OauthClient: &oauthv1.OAuthClient{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				GrantMethod:  oauthv1.GrantHandlerAuto,
				Secret:       "test",
				RedirectURIs: []string{"http://test.com"},
			},
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RHMI",
					APIVersion: integreatlyv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-install",
				},
			},
			client:         fakeclient.NewFakeClientWithScheme(scheme),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
		},
		{
			Name:        "test oauth client is reconciled correctly when it does exist",
			OauthClient: existingClient,
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RHMI",
					APIVersion: integreatlyv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-install",
				},
			},
			client:         fakeclient.NewFakeClientWithScheme(scheme, existingClient),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler := NewReconciler(nil)
			phase, err := reconciler.ReconcileOauthClient(context.TODO(), tc.Installation, tc.OauthClient, tc.client)
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("expected no error but got one ", err)
			}
			if tc.ExpectedStatus != phase {
				t.Fatal("expected phase ", tc.ExpectedStatus, " but got ", phase)
			}
		})
	}
}

func TestReconciler_ReconcileNamespace(t *testing.T) {
	scheme, err := buildScheme()
	if err != nil {
		t.Fatalf("error creating scheme: %s", err.Error())
	}
	nsName := "test-ns"
	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name: "install",
			UID:  types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
	}
	cases := []struct {
		Name           string
		client         k8sclient.Client
		Installation   *integreatlyv1alpha1.RHMI
		ExpectErr      bool
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		ExpectLabel    bool
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name: "Test namespace reconcile completes without error",
			client: fakeclient.NewFakeClientWithScheme(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						OwnerLabelKey: string(installation.GetUID()),
					},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			}),
			Installation:   installation,
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			ExpectLabel:    true,
		},
		{
			Name: "Test namespace reconcile returns waiting when ns not ready",
			client: fakeclient.NewFakeClientWithScheme(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						OwnerLabelKey: string(installation.GetUID()),
					},
				},
				Status: corev1.NamespaceStatus{},
			}),
			Installation:   installation,
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			ExpectLabel:    true,
		},
		{
			Name: "Test namespace reconcile returns waiting when ns is terminating",
			client: fakeclient.NewFakeClientWithScheme(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						OwnerLabelKey: string(installation.GetUID()),
					},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceTerminating,
				},
			}),
			Installation:   &integreatlyv1alpha1.RHMI{},
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			ExpectLabel:    true,
		},
		{
			Name: "Test namespace reconcile return error when pulling secret fails",
			client: fakeclient.NewFakeClientWithScheme(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						OwnerLabelKey: string(installation.GetUID()),
					},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			}),
			Installation: &integreatlyv1alpha1.RHMI{
				Spec: integreatlyv1alpha1.RHMISpec{
					PullSecret: integreatlyv1alpha1.PullSecretSpec{
						Name:      "test",
						Namespace: "test",
					},
				},
			},
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectErr:      true,
			ExpectLabel:    false,
		},
		{
			Name: "Test if label is added to an existing namespace",
			client: fakeclient.NewFakeClientWithScheme(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
				},
			}),
			Installation:   installation,
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			ExpectErr:      false,
			ExpectLabel:    true,
		},
		{
			Name: "Test if label is changed to false when namespace is reconciled",
			client: fakeclient.NewFakeClientWithScheme(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"openshift.io/user-monitoring": "true",
					},
				},
			}),
			Installation:   installation,
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			ExpectErr:      false,
			ExpectLabel:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler := NewReconciler(
				tc.FakeMPM,
			)
			phase, err := reconciler.ReconcileNamespace(context.TODO(), "test-ns", tc.Installation, tc.client, getLogger())
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("expected no error but got one ", err)
			}
			if tc.ExpectedStatus != phase {
				t.Fatal("expected phase ", tc.ExpectedStatus, " but got ", phase)
			}
			labelExists, labelIsTrue, err := verifyNSLabelExistsAndIsTrue(tc.client)
			if err != nil {
				t.Fatal("error when verifying namespace label exists")
			}
			if !labelExists && tc.ExpectLabel {
				t.Fatal("error because label was not applied to namespace")
			}
			if labelIsTrue && tc.ExpectLabel {
				t.Fatal("error when verifying namespace label was changed to false during reconcile")
			}
		})
	}
}

func verifyNSLabelExistsAndIsTrue(client k8sclient.Client) (bool, bool, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}
	err := client.Get(context.TODO(), k8sclient.ObjectKey{Name: ns.Name}, ns)
	if err != nil {
		return false, false, err
	}
	labelExists, labelIsTrue := false, false
	if ns.Labels["openshift.io/user-monitoring"] != "" {
		labelExists = true
	}
	if ns.Labels["openshift.io/user-monitoring"] != "false" && labelExists {
		labelIsTrue = true
	}
	return labelExists, labelIsTrue, nil
}
