package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/metrics"
	"github.com/integr8ly/integreatly-operator/pkg/products/monitoringcommon"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"strings"

	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	rbac "k8s.io/api/rbac/v1"

	"k8s.io/apimachinery/pkg/types"

	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	"github.com/integr8ly/integreatly-operator/version"

	"k8s.io/apimachinery/pkg/api/meta"

	monitoring "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

/* #nosec G101 -- This is a false positive */
const (
	defaultInstallationNamespace = "middleware-monitoring"
	defaultMonitoringName        = "middleware-monitoring"
	packageName                  = "monitoring"
	OpenshiftMonitoringNamespace = "openshift-monitoring"
	grafanaDataSourceSecretName  = "grafana-datasources"
	grafanaDataSourceSecretKey   = "prometheus.yaml"
	defaultBlackboxModule        = "http_2xx"
	manifestPackage              = "integreatly-monitoring"

	// alert manager configuration
	alertManagerRouteName = "alertmanager-route"

	// cluster monitoring federation
	federationServiceMonitorName              = "rhmi-alerts-federate"
	federationRoleBindingName                 = "federation-view"
	clusterMonitoringPrometheusServiceAccount = "prometheus-k8s"
	clusterMonitoringNamespace                = "openshift-monitoring"

	// Cluster infrastructure
	clusterInfraName = "cluster"

	// For Cluster ID
	clusterIDValue = "version"

	// For OpenShift console
	openShiftConsoleRoute     = "console"
	openShiftConsoleNamespace = "openshift-console"
)

type Reconciler struct {
	Config        *config.Monitoring
	extraParams   map[string]string
	ConfigManager config.ConfigReadWriter
	Log           l.Logger
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.RHMI
	monitoring    *monitoring.ApplicationMonitoring
	*resources.Reconciler
	recorder record.EventRecorder
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {
	if productDeclaration == nil {
		return nil, fmt.Errorf("no product declaration found for monitoring")
	}

	config, err := configManager.ReadMonitoring()
	if err != nil {
		return nil, err
	}

	config.SetNamespacePrefix(installation.Spec.NamespacePrefix)
	if config.GetNamespace() == "" {
		config.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
	}

	if config.GetFederationNamespace() == "" {
		config.SetFederationNamespace(config.GetNamespace() + "-federate")
	}

	return &Reconciler{
		Config:        config,
		extraParams:   make(map[string]string),
		ConfigManager: configManager,
		Log:           logger,
		installation:  installation,
		mpm:           mpm,
		Reconciler:    resources.NewReconciler(mpm).WithProductDeclaration(*productDeclaration),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.MonitoringStage].Products[integreatlyv1alpha1.ProductMonitoring],
		string(integreatlyv1alpha1.VersionMonitoring),
		string(integreatlyv1alpha1.OperatorVersionMonitoring),
	)
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, productStatus *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client, _ quota.ProductConfig, uninstall bool) (integreatlyv1alpha1.StatusPhase, error) {
	operatorNamespace := r.Config.GetOperatorNamespace()
	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), uninstall, func() (integreatlyv1alpha1.StatusPhase, error) {
		r.Log.Info("Phase: Monitoring ReconcileFinalizer")
		// Check if namespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, operatorNamespace, serverClient)
		if k8serr.IsNotFound(err) {
			//namespace is gone, return complete
			return integreatlyv1alpha1.PhaseCompleted, nil
		}

		r.Log.Info("Phase: Monitoring ReconcileFinalizer list blackboxtargets")
		blackboxtargets := &monitoring.BlackboxTargetList{}
		blackboxtargetsListOpts := []k8sclient.ListOption{
			k8sclient.MatchingLabels(map[string]string{r.Config.GetLabelSelectorKey(): r.Config.GetLabelSelector()}),
		}
		err = serverClient.List(ctx, blackboxtargets, blackboxtargetsListOpts...)
		if err != nil {
			r.Log.Info("Phase: Monitoring ReconcileFinalizer blackboxtargets error")
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list blackbox targets: %w", err)
		}
		if len(blackboxtargets.Items) > 0 {
			r.Log.Info("Phase: Monitoring ReconcileFinalizer blackboxtargets list > 0")
			// do something to delete these dashboards
			for _, bbt := range blackboxtargets.Items {
				r.Log.Infof("Phase: Monitoring ReconcileFinalizer try delete blackboxtarget", l.Fields{"target": bbt.Name})
				b := &monitoring.BlackboxTarget{}
				err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: bbt.Name, Namespace: operatorNamespace}, b)
				if k8serr.IsNotFound(err) {
					continue
				}
				if err != nil {
					return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get %s blackbox target: %w", bbt.Name, err)
				}

				err = serverClient.Delete(ctx, b)
				if err != nil && !k8serr.IsNotFound(err) {
					return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to delete %s blackbox target: %w", b.Name, err)
				}
			}
			return integreatlyv1alpha1.PhaseInProgress, nil
		}

		m := &monitoring.ApplicationMonitoring{}
		err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: defaultMonitoringName, Namespace: operatorNamespace}, m)
		if err != nil && !k8serr.IsNotFound(err) {
			r.Log.Info("Phase: Monitoring ReconcileFinalizer error fetch ApplicationMonitoring CR")
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get %s application monitoring custom resource: %w", defaultMonitoringName, err)
		}
		if !k8serr.IsNotFound(err) {
			if m.DeletionTimestamp == nil {
				r.Log.Info("Phase: Monitoring ReconcileFinalizer delete ApplicationMonitoring CR")
				err = serverClient.Delete(ctx, m)
				if err != nil && !k8serr.IsNotFound(err) {
					return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to delete %s application monitoring custom resource: %w", defaultMonitoringName, err)
				}
			}
			return integreatlyv1alpha1.PhaseInProgress, nil
		}

		phase, err := resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetFederationNamespace(), r.Log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetOperatorNamespace(), r.Log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		return integreatlyv1alpha1.PhaseInProgress, nil
	}, r.Log)

	if err != nil || phase == integreatlyv1alpha1.PhaseFailed {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	if uninstall {
		return phase, nil
	}

	isMultiAZCluster, err := resources.IsMultiAZCluster(ctx, serverClient)
	if err != nil {
		r.Log.Warning("Failure when deciding if the cluster is multi-az or not. Defaulted to false: " + err.Error())
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, serverClient, r.Log)
	r.Log.Infof("ReconcileNamespace", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetOperatorNamespace()), err)
		return phase, err
	}

	// In this case due to monitoring reconciler is always installed in the
	// same namespace as the operatorNamespace we pass operatorNamespace as the
	// productNamepace too
	phase, err = r.reconcileSubscription(ctx, serverClient, installation, operatorNamespace, operatorNamespace)
	r.Log.Infof("ReconcileSubscription", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.MonitoringSubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, serverClient, isMultiAZCluster)
	r.Log.Infof("ReconcileComponents", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	phase, err = monitoringcommon.ReconcileAlertManagerSecrets(ctx, serverClient, r.installation, r.Config.GetOperatorNamespace(), alertManagerRouteName)
	r.Log.Infof("ReconcileAlertManagerConfigSecret", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		if err != nil {
			r.Log.Warning("failed to reconcile alert manager config secret " + err.Error())
		}
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile alert manager config secret", err)
		return phase, err
	}

	phase, err = r.populateParams(ctx, serverClient)
	r.Log.Infof("populateParams", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to populate parameters", err)
		return phase, err
	}

	phase, err = r.reconcileDashboards(ctx, serverClient)
	r.Log.Infof("reconcileDashboards", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		if err != nil {
			r.Log.Warning("Failure reconciling dashboards " + err.Error())
		}
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile dashboards", err)
		return phase, err
	}

	phase, err = r.reconcileScrapeConfigs(ctx, serverClient)
	r.Log.Infof("reconcileScrapeConfigs", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile scrape configs", err)
		return phase, err
	}

	phase, err = r.createFederationNamespace(ctx, serverClient, installation)
	r.Log.Infof("labelFederationNamespace", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to label federation namespace", err)
		return phase, err
	}

	phase, err = r.reconcileFederation(ctx, serverClient)
	r.Log.Infof("reconcileFederation", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile federation", err)
		return phase, err
	}

	phase, err = r.newAlertsReconciler(r.Log, r.installation.Spec.Type).ReconcileAlerts(ctx, serverClient)
	r.Log.Infof("reconcilePrometheusRule", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile alerts", err)
		return phase, err
	}

	// creates an alert to check for the presents of sendgrid smtp secret
	phase, err = resources.CreateSmtpSecretExists(ctx, serverClient, installation)
	r.Log.Infof("CreateSmtpSecretExistsRule", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile SendgridSmtpSecretExists alert", err)
		return phase, err
	}

	productStatus.Host = r.Config.GetHost()
	productStatus.Version = r.Config.GetProductVersion()
	productStatus.OperatorVersion = r.Config.GetOperatorVersion()

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, "Failed to update monitoring config", err)
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not update monitoring config: %w", err)
	}

	err = r.updateGrafanaImage(r.Config.GetOperatorNamespace(), ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.MonitoringStage, r.Config.GetProductName())
	r.Log.Info("Reconciled successfully")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

// make the federation namespace discoverable by cluster monitoring
func (r *Reconciler) createFederationNamespace(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	namespace, err := resources.GetNS(ctx, r.Config.GetFederationNamespace(), serverClient)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		_, err := resources.CreateNSWithProjectRequest(ctx, r.Config.GetFederationNamespace(), serverClient, installation, false, true, true)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	resources.PrepareObjectLabels(namespace, installation, false, true, true)

	err = serverClient.Update(ctx, namespace)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// Creates a service monitor that federates metrics about alerts to the cluster
// monitoring stack
func (r *Reconciler) reconcileFederation(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	serviceMonitor := &prometheus.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      federationServiceMonitorName,
			Namespace: r.Config.GetFederationNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, serviceMonitor, func() error {
		serviceMonitor.Labels = map[string]string{
			"k8s-app": federationServiceMonitorName,
			"name":    federationServiceMonitorName,
		}
		serviceMonitor.Spec = prometheus.ServiceMonitorSpec{
			Endpoints: []prometheus.Endpoint{
				{
					Port:   "upstream",
					Path:   "/federate",
					Scheme: "http",
					Params: map[string][]string{
						"match[]": []string{"{__name__=\"ALERTS\",alertstate=\"firing\"}"},
					},
					Interval:      r.Config.GetExtraParamWithDefault(config.MonitoringParamFederateScrapeInterval, config.MonitoringDefaultFederateScrapeInterval),
					ScrapeTimeout: r.Config.GetExtraParamWithDefault(config.MonitoringParamFederateScrapeTimeout, config.MonitoringDefaultFederateScrapeTimeout),
					HonorLabels:   true,
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"application-monitoring": "true",
				},
			},
			NamespaceSelector: prometheus.NamespaceSelector{
				MatchNames: []string{r.Config.GetOperatorNamespace()},
			},
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.Log.Infof("Operation result", l.Fields{"serviceMonitor": federationServiceMonitorName, "res": or})

	roleBinding := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      federationRoleBindingName,
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}

	or, err = controllerutil.CreateOrUpdate(ctx, serverClient, roleBinding, func() error {
		roleBinding.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      clusterMonitoringPrometheusServiceAccount,
				Namespace: clusterMonitoringNamespace,
			},
		}
		roleBinding.RoleRef = rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     bundle.ClusterRoleKind,
			Name:     "view",
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.Log.Infof("Operation result", l.Fields{"roleBinding": federationRoleBindingName, "res": or})

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// Create the integreatly additional scrape config secret which is reconciled
// by the application monitoring operator and passed to prometheus
func (r *Reconciler) reconcileScrapeConfigs(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	templateHelper := monitoringcommon.NewTemplateHelper(r.extraParams)
	threeScaleConfig, err := r.ConfigManager.ReadThreeScale()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reading config: %w", err)
	}

	jobs := strings.Builder{}
	for _, job := range r.Config.GetJobTemplates() {
		// Don't include the 3scale extra scrape config if the product is not installed
		if strings.Contains(job, "3scale") && threeScaleConfig.GetNamespace() == "" {
			r.Log.Info("skipping 3scale additional scrape config")
			continue
		}

		bytes, err := templateHelper.LoadTemplate(job)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error loading template: %w", err)
		}

		jobs.Write(bytes)
		jobs.WriteByte('\n')
	}

	scrapeConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Config.GetAdditionalScrapeConfigSecretName(),
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, scrapeConfigSecret, func() error {
		scrapeConfigSecret.Data = map[string][]byte{
			r.Config.GetAdditionalScrapeConfigSecretKey(): []byte(jobs.String()),
		}
		scrapeConfigSecret.Type = "Opaque"
		scrapeConfigSecret.Labels = map[string]string{
			r.Config.GetLabelSelectorKey(): r.Config.GetLabelSelector(),
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating additional scrape config secret: %w", err)
	}

	r.Log.Info(fmt.Sprintf("operation result of creating additional scrape config secret was %v", or))

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileDashboards(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	for _, dashboard := range r.Config.GetDashboards(integreatlyv1alpha1.InstallationType(r.installation.Spec.Type)) {
		err := r.reconcileGrafanaDashboards(ctx, serverClient, dashboard)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update grafana dashboard %s: %w", dashboard, err)
		}
		r.Log.Infof("Reconcile successful", l.Fields{"grafanaDashboard": dashboard})
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileGrafanaDashboards(ctx context.Context, serverClient k8sclient.Client, dashboard string) (err error) {

	//clusterVersion
	containerCpuMetric, err := metrics.GetContainerCPUMetric(ctx, serverClient, r.Log)
	if err != nil {
		return err
	}

	grafanaDB := &grafanav1alpha1.GrafanaDashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dashboard,
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}
	specJSON, _, err := monitoringcommon.GetSpecDetailsForDashboard(dashboard, r.installation, containerCpuMetric)
	if err != nil {
		return err
	}

	pluginList := monitoringcommon.GetPluginsForGrafanaDashboard(dashboard)

	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, grafanaDB, func() error {
		grafanaDB.Labels = map[string]string{
			"monitoring-key": r.Config.GetLabelSelector(),
		}
		grafanaDB.Spec = grafanav1alpha1.GrafanaDashboardSpec{
			Json: specJSON,
		}
		if len(pluginList) > 0 {
			grafanaDB.Spec.Plugins = pluginList
		}
		return nil
	})
	if err != nil {
		return err
	}
	if opRes != controllerutil.OperationResultNone {
		r.Log.Infof("Operation result", l.Fields{"grafanaDashboard": grafanaDB.Name, "result": opRes})
	}
	return err
}

func (r *Reconciler) reconcileComponents(ctx context.Context, serverClient k8sclient.Client, isMultiAZCluster bool) (integreatlyv1alpha1.StatusPhase, error) {
	r.Log.Info("Reconciling Monitoring Components")

	m := &monitoring.ApplicationMonitoring{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultMonitoringName,
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}

	antiAffinityRequired, err := resources.IsAntiAffinityRequired(ctx, serverClient)
	if err != nil {
		r.Log.Warning("Failure when deciding if monitoring pod anti affinity is required. Defaulted to false: " + err.Error())
	}

	owner.AddIntegreatlyOwnerAnnotations(m, r.installation)
	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, m, func() error {
		m.Spec = monitoring.ApplicationMonitoringSpec{
			LabelSelector:                    r.Config.GetLabelSelector(),
			AdditionalScrapeConfigSecretName: r.Config.GetAdditionalScrapeConfigSecretName(),
			AdditionalScrapeConfigSecretKey:  r.Config.GetAdditionalScrapeConfigSecretKey(),
			PrometheusRetention:              r.Config.GetPrometheusRetention(),
			PrometheusStorageRequest:         r.Config.GetPrometheusStorageRequest(),
			AlertmanagerInstanceNamespaces:   r.Config.GetOperatorNamespace(),
			PrometheusInstanceNamespaces:     r.Config.GetOperatorNamespace(),
			SelfSignedCerts:                  r.installation.Spec.SelfSignedCerts,
		}

		if isMultiAZCluster && integreatlyv1alpha1.IsRHOAM(integreatlyv1alpha1.InstallationType(r.installation.Spec.Type)) {
			m.Spec.Affinity = resources.SelectAntiAffinityForCluster(antiAffinityRequired, map[string]string{
				"prometheus":   "application-monitoring",
				"alertmanager": "application-monitoring",
			})
		} else {
			m.Spec.Affinity = nil
		}

		r.monitoring = m
		return nil
	})
	if err != nil {
		r.Log.Error("Failed reconciling AMO CR", err)
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update applicationmonitoring custom resource: %w", err)
	}

	r.Log.Info("Reconciling Grafana ServiceAccount")
	grafanaServiceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-serviceaccount",
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}

	or, err = controllerutil.CreateOrUpdate(ctx, serverClient, grafanaServiceAccount, func() error {
		serviceAccountAnnotations := grafanaServiceAccount.ObjectMeta.GetAnnotations()
		if serviceAccountAnnotations == nil {
			serviceAccountAnnotations = map[string]string{}
		}
		serviceAccountAnnotations["serviceaccounts.openshift.io/oauth-redirectreference.primary"] = "{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"grafana-route\"}}"
		grafanaServiceAccount.ObjectMeta.SetAnnotations(serviceAccountAnnotations)

		return nil
	})
	if err != nil {
		r.Log.Error("Failed reconciling grafana service account", err)
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update grafana service account: %w", err)
	}

	r.Log.Infof("Operation result", l.Fields{"monitoring": m.Name, "result": or})
	return integreatlyv1alpha1.PhaseCompleted, nil
}

// CreateResource Creates a generic kubernetes resource from a template
func (r *Reconciler) createResource(ctx context.Context, resourceName string, serverClient k8sclient.Client) (runtime.Object, error) {
	if r.extraParams == nil {
		r.extraParams = map[string]string{}
	}
	r.extraParams["MonitoringKey"] = r.Config.GetLabelSelector()
	r.extraParams["Namespace"] = r.Config.GetOperatorNamespace()

	templateHelper := monitoringcommon.NewTemplateHelper(r.extraParams)
	resource, err := templateHelper.CreateResource(resourceName)

	if err != nil {
		return nil, fmt.Errorf("createResource failed: %w", err)
	}

	metaObj, err := meta.Accessor(resource)
	if err == nil {
		owner.AddIntegreatlyOwnerAnnotations(metaObj, r.installation)
	}

	err = serverClient.Create(ctx, resource)
	if err != nil {
		if !k8serr.IsAlreadyExists(err) {
			return nil, fmt.Errorf("error creating resource: %w", err)
		}
	}

	return resource, nil
}

// Read the credentials of the Prometheus instance in the openshift-monitoring
// namespace from the grafana datasource secret
func (r *Reconciler) readFederatedPrometheusCredentials(ctx context.Context, serverClient k8sclient.Client) (*monitoring.GrafanaDataSourceSecret, error) {
	secret := &corev1.Secret{}

	selector := k8sclient.ObjectKey{
		Namespace: OpenshiftMonitoringNamespace,
		Name:      grafanaDataSourceSecretName,
	}

	err := serverClient.Get(ctx, selector, secret)
	if err != nil {
		return nil, err
	}

	prometheusConfig := secret.Data[grafanaDataSourceSecretKey]
	datasources := monitoring.GrafanaDataSourceSecret{}

	err = json.Unmarshal(prometheusConfig, &datasources)
	if err != nil {
		return nil, err
	}

	return &datasources, err
}

// Populate the extra params for templating
func (r *Reconciler) populateParams(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Obtain the prometheus credentials from openshift-monitoring
	datasources, err := r.readFederatedPrometheusCredentials(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if len(datasources.DataSources) < 1 {
		return integreatlyv1alpha1.PhaseFailed, errors.New("cannot obtain prometheus credentials")
	}

	// Obtain the 3scale config and namespace
	threeScaleConfig, err := r.ConfigManager.ReadThreeScale()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.extraParams["threescale_namespace"] = threeScaleConfig.GetNamespace()
	r.extraParams["namespace-prefix"] = r.installation.Spec.NamespacePrefix
	r.extraParams["openshift_monitoring_namespace"] = OpenshiftMonitoringNamespace
	r.extraParams["openshift_monitoring_prometheus_username"] = datasources.DataSources[0].BasicAuthUser
	r.extraParams["openshift_monitoring_prometheus_password"] = datasources.DataSources[0].BasicAuthPassword
	r.extraParams["openshift_monitoring_federate_scrape_interval"] = r.Config.GetExtraParamWithDefault(config.MonitoringParamFederateScrapeInterval, config.MonitoringDefaultFederateScrapeInterval)
	r.extraParams["openshift_monitoring_federate_scrape_timeout"] = r.Config.GetExtraParamWithDefault(config.MonitoringParamFederateScrapeTimeout, config.MonitoringDefaultFederateScrapeTimeout)

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func CreateBlackboxTarget(ctx context.Context, name string, target monitoring.BlackboxtargetData, cfg *config.Monitoring, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) error {
	if cfg.GetOperatorNamespace() == "" {
		// Retry later
		return nil
	}

	if target.Url == "" {
		// Retry later if the URL is not yet known
		return nil
	}

	// default policy is to require a 2xx http return code
	module := target.Module
	if module == "" {
		module = defaultBlackboxModule
	}

	// prepare the template
	extraParams := map[string]string{
		"Namespace":     cfg.GetOperatorNamespace(),
		"MonitoringKey": cfg.GetLabelSelector(),
		"name":          name,
		"url":           target.Url,
		"service":       target.Service,
		"module":        module,
	}

	templateHelper := monitoringcommon.NewTemplateHelper(extraParams)
	obj, err := templateHelper.CreateResource("blackbox/target.yaml")
	if err != nil {
		return fmt.Errorf("error creating resource from template: %w", err)
	}

	metaObj, err := meta.Accessor(obj)
	if err == nil {
		owner.AddIntegreatlyOwnerAnnotations(metaObj, installation)
	}
	// try to create the blackbox target. If if fails with already exist do nothing
	err = serverClient.Create(ctx, obj)
	if err != nil {
		if k8serr.IsAlreadyExists(err) {
			// The target already exists. Nothing else to do
			return nil
		}
		return fmt.Errorf("error creating blackbox target: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		SubscriptionName: constants.MonitoringSubscriptionName,
		Namespace:        operatorNamespace,
	}

	catalogSourceReconciler, err := r.GetProductDeclaration().PrepareTarget(
		r.Log,
		serverClient,
		marketplace.CatalogSourceName,
		&target,
	)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return r.Reconciler.ReconcileSubscription(
		ctx,
		target,
		[]string{productNamespace},
		backup.NewNoopBackupExecutor(),
		serverClient,
		catalogSourceReconciler,
		r.Log,
	)

}

func (r *Reconciler) getPagerDutySecret(ctx context.Context, serverClient k8sclient.Client) (string, error) {

	var secret string

	pagerdutySecret := &corev1.Secret{}
	err := serverClient.Get(ctx, types.NamespacedName{Name: r.installation.Spec.PagerDutySecret,
		Namespace: r.installation.Namespace}, pagerdutySecret)

	if err != nil {
		return "", fmt.Errorf("could not obtain pagerduty credentials secret: %w", err)
	}

	if len(pagerdutySecret.Data["PAGERDUTY_KEY"]) != 0 {
		secret = string(pagerdutySecret.Data["PAGERDUTY_KEY"])
	} else if len(pagerdutySecret.Data["serviceKey"]) != 0 {
		secret = string(pagerdutySecret.Data["serviceKey"])
	}

	if secret == "" {
		return "", fmt.Errorf("secret key is undefined in pager duty secret")
	}

	return secret, nil
}

func (r *Reconciler) getDMSSecret(ctx context.Context, serverClient k8sclient.Client) (string, error) {

	var secret string

	dmsSecret := &corev1.Secret{}
	err := serverClient.Get(ctx, types.NamespacedName{Name: r.installation.Spec.DeadMansSnitchSecret,
		Namespace: r.installation.Namespace}, dmsSecret)

	if err != nil {
		return "", fmt.Errorf("could not obtain dead mans snitch credentials secret: %w", err)
	}

	if len(dmsSecret.Data["SNITCH_URL"]) != 0 {
		secret = string(dmsSecret.Data["SNITCH_URL"])
	} else if len(dmsSecret.Data["url"]) != 0 {
		secret = string(dmsSecret.Data["url"])
	} else {
		return "", fmt.Errorf("url is undefined in dead mans snitch secret")
	}

	return secret, nil
}

// prepareEmailAddresses converts a space separated string into a comma separated
// string. Example:
//
// "foo@example.org bar@example.org" -> "foo@example.org, bar@example.org"
func prepareEmailAddresses(list string) string {
	addresses := strings.Split(strings.TrimSpace(list), " ")
	return strings.Join(addresses, ", ")
}

func (r *Reconciler) updateGrafanaImage(operatorNamespace string, ctx context.Context, serverClient k8sclient.Client) error {
	grafana := &grafanav1alpha1.Grafana{}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: "grafana", Namespace: operatorNamespace}, grafana)
	if err != nil {
		return err
	}

	grafana.Spec.BaseImage = fmt.Sprintf("%s:%s", constants.GrafanaImage, constants.GrafanaVersion)

	// Hotfix to unblock 1.10. TODO: improve this solution by the next release
	if len(grafana.Spec.Containers) > 0 {
		grafana.Spec.Containers[0].Image = "registry.redhat.io/openshift4/ose-oauth-proxy@sha256:9a5ee95f8e99a63a4ad0e8b01683ac03c75337bbbe3d504d199a97f9921eb0c1"
	}

	err = serverClient.Update(ctx, grafana)
	if err != nil {
		return err
	}

	return nil
}
