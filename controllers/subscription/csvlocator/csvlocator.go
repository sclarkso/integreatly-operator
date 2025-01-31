package csvlocator

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/registry"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type CSVLocator interface {
	GetCSV(ctx context.Context, client k8sclient.Client, installPlan *olmv1alpha1.InstallPlan) (*olmv1alpha1.ClusterServiceVersion, error)
}

type EmbeddedCSVLocator struct{}

var _ CSVLocator = &EmbeddedCSVLocator{}

func (l *EmbeddedCSVLocator) GetCSV(ctx context.Context, client k8sclient.Client, installPlan *olmv1alpha1.InstallPlan) (*olmv1alpha1.ClusterServiceVersion, error) {
	csv := &olmv1alpha1.ClusterServiceVersion{}

	// The latest CSV is only represented in the new install plan while the upgrade is pending approval
	for _, installPlanResources := range installPlan.Status.Plan {
		if installPlanResources.Resource.Kind == olmv1alpha1.ClusterServiceVersionKind {
			err := json.Unmarshal([]byte(installPlanResources.Resource.Manifest), &csv)
			if err != nil {
				return csv, fmt.Errorf("failed to unmarshal json: %w", err)
			}
		}
	}

	return csv, nil
}

type ConfigMapCSVLocator struct{}

var _ CSVLocator = &ConfigMapCSVLocator{}

type unpackedBundleReference struct {
	Kind                   string `json:"kind"`
	Name                   string `json:"name"`
	Namespace              string `json:"namespace"`
	CatalogSourceName      string `json:"catalogSourceName"`
	CatalogSourceNamespace string `json:"catalogSourceNamespace"`
	Replaces               string `json:"replaces"`
}

func (l *ConfigMapCSVLocator) GetCSV(ctx context.Context, client k8sclient.Client, installPlan *olmv1alpha1.InstallPlan) (*olmv1alpha1.ClusterServiceVersion, error) {
	csv := &olmv1alpha1.ClusterServiceVersion{}

	// The latest CSV is only represented in the new install plan while the upgrade is pending approval
	for _, installPlanResources := range installPlan.Status.Plan {
		if installPlanResources.Resource.Kind != olmv1alpha1.ClusterServiceVersionKind {
			continue
		}

		// Get the reference to the ConfigMap that contains the CSV
		ref := &unpackedBundleReference{}
		err := json.Unmarshal([]byte(installPlanResources.Resource.Manifest), &ref)
		if err != nil {
			return csv, fmt.Errorf("failed to unmarshal json: %w", err)
		}

		// Get the ConfigMap
		csvConfigMap := &corev1.ConfigMap{}
		if err = client.Get(ctx, k8sclient.ObjectKey{
			Name:      ref.Name,
			Namespace: ref.Namespace,
		}, csvConfigMap); err != nil {
			return csv, fmt.Errorf("error retrieving ConfigMap %s/%s: %v", ref.Namespace, ref.Name, err)
		}

		// The ConfigMap may contain other manifests other than the CSV. Iterate
		// through the data and skip the ones that have a kind other than
		// ClusterSeerviceVersion. This is for 4.8 clusters
		csvStr := ""
		for _, resourceStr := range csvConfigMap.Data {
			csvCandidate, err := getCSVfromCM(&csvStr, resourceStr)
			if csvCandidate == nil {
				continue
			}
			if err != nil {
				return csv, err
			}
			csv = csvCandidate
		}

		// ConfigMap may contain CSV as binary data (on 4.9+ clusters). To account for this check
		// if the CSV was found in a data and if not - look for a binary data
		if csvStr == "" {
			for _, resourceByte := range csvConfigMap.BinaryData {
				// Decode base64 into a string
				base64text := make([]byte, base64.StdEncoding.DecodedLen(len(resourceByte)))
				_, err = base64.StdEncoding.Decode(base64text, resourceByte)
				if err != nil {
					return nil, err
				}

				// decompress from gzip
				reader, err := gzip.NewReader(bytes.NewBuffer(base64text))
				if err != nil {
					return nil, err
				}
				result, _ := ioutil.ReadAll(reader)

				csvCandidate, err := getCSVfromCM(&csvStr, string(result))
				if csvCandidate == nil {
					continue
				}
				if err != nil {
					return csv, err
				}
				csv = csvCandidate

			}
		}
	}
	return csv, nil
}

func getCSVfromCM(csvStr *string, resourceStr string) (*olmv1alpha1.ClusterServiceVersion, error) {
	csv := &olmv1alpha1.ClusterServiceVersion{}

	// Decode the manifest
	reader := strings.NewReader(resourceStr)
	resource, decodeErr := registry.DecodeUnstructured(reader)
	if decodeErr != nil {
		return csv, decodeErr
	}

	// If the kind is not CSV, skip it
	if resource.GetKind() != olmv1alpha1.ClusterServiceVersionKind {
		return nil, nil
	}

	csvStr = &resourceStr
	// Encode the unstructured CSV as Json to decode it back to the
	// structured object
	resourceJSON, err := resource.MarshalJSON()
	if err != nil {
		return csv, err
	}

	if err := json.Unmarshal(resourceJSON, csv); err != nil {
		return csv, fmt.Errorf("failed to unmarshall yaml: %v", err)
	}
	return csv, nil
}

type CachedCSVLocator struct {
	cache map[string]*olmv1alpha1.ClusterServiceVersion

	locator CSVLocator
}

var _ CSVLocator = &CachedCSVLocator{}

func NewCachedCSVLocator(innerLocator CSVLocator) *CachedCSVLocator {
	return &CachedCSVLocator{
		cache:   map[string]*olmv1alpha1.ClusterServiceVersion{},
		locator: innerLocator,
	}
}

func (l *CachedCSVLocator) GetCSV(ctx context.Context, client k8sclient.Client, installPlan *olmv1alpha1.InstallPlan) (*olmv1alpha1.ClusterServiceVersion, error) {
	key := fmt.Sprintf("%s/%s", installPlan.Namespace, installPlan.Name)

	if found, ok := l.cache[key]; ok {
		return found, nil
	}

	csv, err := l.locator.GetCSV(ctx, client, installPlan)
	if err != nil {
		return nil, err
	}

	if csv != nil {
		l.cache[key] = csv
	}

	return csv, nil
}

type ConditionalCSVLocator struct {
	Condition func(installPlan *olmv1alpha1.InstallPlan) CSVLocator
}

func NewConditionalCSVLocator(condition func(installPlan *olmv1alpha1.InstallPlan) CSVLocator) *ConditionalCSVLocator {
	return &ConditionalCSVLocator{
		Condition: condition,
	}
}

var _ CSVLocator = &ConditionalCSVLocator{}

func (l *ConditionalCSVLocator) GetCSV(ctx context.Context, client k8sclient.Client, installPlan *olmv1alpha1.InstallPlan) (*olmv1alpha1.ClusterServiceVersion, error) {
	locator := l.Condition(installPlan)
	if locator == nil {
		return nil, fmt.Errorf("no csvlocator found for installplan %s", installPlan.Name)
	}

	return locator.GetCSV(ctx, client, installPlan)
}

func SwitchLocators(conditions ...func(*olmv1alpha1.InstallPlan) CSVLocator) func(*olmv1alpha1.InstallPlan) CSVLocator {
	return func(installPlan *olmv1alpha1.InstallPlan) CSVLocator {
		for _, condition := range conditions {
			if locator := condition(installPlan); locator != nil {
				return locator
			}
		}

		return nil
	}
}

func ForReference(installPlan *olmv1alpha1.InstallPlan) CSVLocator {
	for _, installPlanResources := range installPlan.Status.Plan {
		if installPlanResources.Resource.Kind != olmv1alpha1.ClusterServiceVersionKind {
			continue
		}

		// Get the reference to the ConfigMap that contains the CSV
		ref := &unpackedBundleReference{}
		err := json.Unmarshal([]byte(installPlanResources.Resource.Manifest), &ref)
		if err != nil || ref.Name == "" || ref.Namespace == "" {
			return nil
		}

		return &ConfigMapCSVLocator{}
	}

	return nil
}

func ForEmbedded(installPlan *olmv1alpha1.InstallPlan) CSVLocator {
	csv := &olmv1alpha1.ClusterServiceVersion{}

	// The latest CSV is only represented in the new install plan while the upgrade is pending approval
	for _, installPlanResources := range installPlan.Status.Plan {
		if installPlanResources.Resource.Kind == olmv1alpha1.ClusterServiceVersionKind {
			err := json.Unmarshal([]byte(installPlanResources.Resource.Manifest), &csv)
			if err != nil || csv.Name == "" || csv.Namespace == "" {
				return nil
			}

			return &EmbeddedCSVLocator{}
		}
	}

	return nil
}
